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
	"time"

	"github.com/cedana/cedana-cli/utils"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	cedana "github.com/cedana/cedana-cli/types"
)

var id string

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

var setupTaskCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup a task to run on Cedana",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()

		if id == "" {
			r.logger.Fatal().Msg("job file not specified")
		}

		file, err := os.Open(args[0])
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not open job file")
		}
		defer file.Close()

		contents, err := io.ReadAll(file)
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not read job file")
		}

		encodedJob := base64.StdEncoding.EncodeToString(contents)

		err = r.setupTask(encodedJob, args[0])
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not setup task")
		}
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

var createInstanceCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an instance for a task [task_id]",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()
		err := r.createInstance(CreateInstanceRequest{
			TaskID: args[0],
		})
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not create instance")
		}
	},
}

var setupInstanceCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup an instance [instance_id]",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()
		err := r.setupInstance(args[0])
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not setup instance")
		}
	},
}

type setupTaskRequest struct {
	TaskConfig string `json:"task_config"`
	TaskLabel  string `json:"label"`
}

type setupTaskResponse struct {
	TaskID string `json:"task_id"`
}

func (r *Runner) setupTask(encodedJob, taskLabel string) error {
	st := setupTaskRequest{
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

	var str setupTaskResponse
	err = json.NewDecoder(resp.Body).Decode(&str)
	if err != nil {
		return err
	}

	fmt.Printf("Task created with ID: %s", str.TaskID)

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

func (r *Runner) createInstance(instanceReq CreateInstanceRequest) error {
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

	fmt.Printf("Instance created with ID: %s", str.InstanceID)

	return nil
}

func (r *Runner) setupInstance(instanceID string) error {
	// Interrupt handler to gracefully close the WebSocket connection.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: r.cfg.MarketServiceUrl, Path: "/instance/setup/" + instanceID + "/ws"}
	r.logger.Info().Msgf("Connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
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

func init() {
	RootCmd.AddCommand(setupTaskCmd)
	RootCmd.AddCommand(listTasksCmd)
	RootCmd.AddCommand(createInstanceCmd)
	RootCmd.AddCommand(setupInstanceCmd)

	setupTaskCmd.Flags().StringVarP(&id, "id", "i", "", "id for task")
}
