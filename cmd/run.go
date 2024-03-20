package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/cedana/cedana-cli/utils"
	"github.com/rs/xid"
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

// TODO NR - take a label as input instead
var runTaskCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a previously setup task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()
		handlerID := xid.New()
		err := r.createInstance(CreateInstanceRequest{
			PollHandlerID: handlerID.String(),
			TaskID:        args[0],
		})
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not run task")
		}
	},
}

var pollTaskCmd = &cobra.Command{
	Use:   "poll",
	Short: "Poll for task status",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()
		handlerID := xid.New()
		err := r.createInstance(CreateInstanceRequest{
			PollHandlerID: handlerID.String(),
		})
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not run task")
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
	PollHandlerID string `json:"poll_handler_id"`
	TaskID        string `json:"task_id"`
	Label         string `json:"label"`
}

type CreateInstanceResponse struct {
	PollHandlerID string `json:"poll_handler_id"`
	InstanceID    string `json:"instance_id"`
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
	fmt.Printf("Tail setup logs with command: cedana-cli poll setup %s", str.PollHandlerID)

	return nil
}

type pollCreateInstanceRequest struct {
	PollHandlerID string `json:"poll_handler_id"`
}

func (r *Runner) pollCreateInstance(pollReq pollCreateInstanceRequest) (string, error) {
	jsonBody, err := json.Marshal(pollReq)
	if err != nil {
		return "", err
	}

	url := r.cfg.MarketServiceUrl + "/instance" + "/events"

	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func init() {
	RootCmd.AddCommand(setupTaskCmd)
	RootCmd.AddCommand(listTasksCmd)
	RootCmd.AddCommand(runTaskCmd)

	setupTaskCmd.Flags().StringVarP(&id, "id", "i", "", "id for task")
}
