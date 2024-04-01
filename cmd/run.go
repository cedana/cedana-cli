package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/cedana/cedana-cli/utils"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cedana "github.com/cedana/cedana-cli/types"
)

type Runner struct {
	ctx       context.Context
	cfg       *utils.CedanaConfig
	logger    *zerolog.Logger
	providers map[string]cedana.Provider
}

type Task struct {
	Owner  string `json:"owner"`
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Config string `json:"config"`
}

type JobFile struct {
	WorkDirPath       string            `mapstructure:"work_dir_path"` // path to a bucket you want mounted
	UserInstanceSpecs UserInstanceSpecs `mapstructure:"instance_specs"`
	SetupCommands     []string          `mapstructure:"setup"`
	Task              string            `mapstructure:"task"` // task to run on instance
}

type UserInstanceSpecs struct {
	InstanceType string  `mapstructure:"instance_type"`
	Memory       int     `mapstructure:"memory_gb"`
	VCPUs        int     `mapstructure:"cpu_cores"`
	VRAM         int     `mapstructure:"vram_gb"`
	GPU          string  `mapstructure:"gpu"`
	MaxPrice     float64 `mapstructure:"max_price_usd_hour"`
}

func BuildRunner() *Runner {
	l := utils.GetLogger()

	config, err := utils.InitCedanaConfig()
	if err != nil {
		l.Fatal().Err(err).Msg("could not set up config")
	}

	return &Runner{
		ctx:       context.Background(),
		cfg:       config,
		logger:    &l,
		providers: make(map[string]cedana.Provider),
	}
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "create, manage and destroy tasks on Cedana",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("must run with subcommand/s")
	},
}

var createTaskCmd = &cobra.Command{
	Use:   "create",
	Short: "Setup a task [jobfile] with [label] to run on Cedana",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r := BuildRunner()

		file, err := os.Open(args[0])
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not open job file")
			return err
		}
		defer file.Close()

		contents, err := io.ReadAll(file)
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not read job file")
			return err
		}

		// attempt to marshal, if it fails exit
		var job JobFile
		err = yaml.Unmarshal(contents, &job)
		if err != nil {
			r.logger.Fatal().Err(err).Msg("contents of job file are not valid yaml")
		}

		// encode and send as string
		encodedJob := base64.StdEncoding.EncodeToString(contents)

		err = r.createTask(encodedJob, args[1])
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not setup task")
		}

		return err
	},
}

var listTasksCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()

		err := r.listTask()
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not list tasks")
		}
	},
}

var instanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "create, manage and destroy instances on Cedana",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("must be run with subcommand/s")
	},
}

var createInstanceCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an instance for a task [task_id]",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()
		err := r.createInstance(cmd.Context(), CreateInstanceRequest{
			TaskID: args[0],
		})
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not create instance")
		}
	},
}

var listInstancesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all instances associated w/ owner",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()
		err := r.listInstances(cmd.Context())
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not list instances")
		}
	},
}

type createTaskRequest struct {
	TaskConfig string `json:"task_config"`
	TaskLabel  string `json:"label"`
}

type createTaskResponse struct {
	TaskID string `json:"task_id"`
}

func (r *Runner) createTask(encodedJob, taskLabel string) error {
	st := createTaskRequest{
		TaskConfig: encodedJob,
		TaskLabel:  taskLabel,
	}

	jsonBody, err := json.Marshal(st)
	if err != nil {
		return err
	}

	url := r.cfg.MarketServiceUrl + "/task"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var str createTaskResponse
	err = json.NewDecoder(resp.Body).Decode(&str)
	if err != nil {
		return err
	}

	r.logger.Info().Msgf("Task created with ID: %s", str.TaskID)

	return nil
}

type listTaskResponse struct {
	Tasks []Task `json:"tasks"`
}

func (r *Runner) listTask() error {
	url := r.cfg.MarketServiceUrl + "/task"
	fmt.Println(url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var str listTaskResponse
	err = json.NewDecoder(resp.Body).Decode(&str)
	if err != nil {
		return err
	}

	for _, t := range str.Tasks {
		fmt.Printf("Task ID: %s, Label: %s\n", t.ID, t.Label)
	}

	return nil
}

type CreateInstanceRequest struct {
	TaskID string `json:"task_id"`
	Label  string `json:"label"`
}

type CreateInstanceResponse struct {
	InstanceID string `json:"instance_id"`
}

// creates and sets up an instance
func (r *Runner) createInstance(ctx context.Context, instanceReq CreateInstanceRequest) error {
	r.logger.Info().Msgf("Creating instance for task: %s", instanceReq.TaskID)
	jsonBody, err := json.Marshal(instanceReq)
	if err != nil {
		return err
	}

	url := r.cfg.MarketServiceUrl + "/instance" + "/create"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var str CreateInstanceResponse
	err = json.NewDecoder(resp.Body).Decode(&str)
	if err != nil {
		return err
	}

	r.logger.Info().Msgf("instance created with ID: %s, starting setup...", str.InstanceID)

	time.Sleep(1 * time.Minute)
	// set up
	err = r.setupInstance(ctx, str.InstanceID)
	if err != nil {
		return err
	}

	return nil
}

func (r *Runner) setupInstance(ctx context.Context, instanceID string) error {
	// Interrupt handler to gracefully close the WebSocket connection.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	host := strings.Split(r.cfg.MarketServiceUrl, "://")[1]

	u := url.URL{Scheme: "wss", Host: host, Path: "/instance/setup/" + instanceID + "/ws"}
	r.logger.Info().Msgf("Connecting to %s", u.String())

	reqHeader := http.Header{}
	reqHeader.Add("Authorization", "Bearer "+r.cfg.AuthToken)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), reqHeader)
	if err != nil {
		return err
	}

	defer c.Close()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				r.logger.Error().Err(err).Msg("read:")
				return
			}
			r.logger.Info().Msgf("recv: %s", message)
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			r.logger.Info().Msg("interrupt")
			err := c.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				r.logger.Error().Err(err).Msg("write close:")
				return err
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}

type ListInstancesResponse struct {
	InstanceID string `json:"instance_id"`
	ProviderID string `json:"provider_id"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Price      string `json:"price"`
	Region     string `json:"region"`
	Provider   string `json:"provider"`
}

func (r *Runner) listInstances(ctx context.Context) error {
	url := r.cfg.MarketServiceUrl + "/instance"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var str ListInstancesResponse
	err = json.NewDecoder(resp.Body).Decode(&str)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	RootCmd.AddCommand(taskCmd)
	RootCmd.AddCommand(instanceCmd)

	// ideal flow/path
	taskCmd.AddCommand(createTaskCmd)
	instanceCmd.AddCommand(createInstanceCmd)
}
