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

	"github.com/cedana/cedana-cli/utils"
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

var runTaskCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a previously setup task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()

		err := r.runTask(args[0])
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

	url := r.cfg.MarketServiceUrl + "/" + "/task"

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
	url := r.cfg.MarketServiceUrl + "/" + "/task"

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

type runTaskResponse struct {
	InstanceID string `json:"cloud_instance_id"`
}

func (r *Runner) runTask(taskLabel string) error {
	url := r.cfg.MarketServiceUrl + "/" + "/task/" + taskLabel + "/run"

	req, err := http.NewRequest("POST", url, nil)
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

	var rtr runTaskResponse
	err = json.NewDecoder(resp.Body).Decode(&rtr)
	if err != nil {
		return err
	}

	fmt.Println("Task started on instance with ID: ", rtr.InstanceID)

	return nil
}

func init() {
	RootCmd.AddCommand(setupTaskCmd)
	RootCmd.AddCommand(listTasksCmd)
	RootCmd.AddCommand(runTaskCmd)

	setupTaskCmd.Flags().StringVarP(&id, "id", "i", "", "id for task")
}
