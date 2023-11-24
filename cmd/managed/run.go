package managed

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/cedana/cedana-cli/cmd"
	"github.com/cedana/cedana-cli/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	cedana "github.com/cedana/cedana-cli/types"
)

var jobFile string

type Runner struct {
	ctx       context.Context
	cfg       *utils.CedanaConfig
	logger    *zerolog.Logger
	providers map[string]cedana.Provider
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

var runJobManaged = &cobra.Command{
	Use:   "managed",
	Short: "Run your workloads on the Cedana system.",
}

var setupTaskCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup a task to run on Cedana",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := BuildRunner()

		if jobFile == "" {
			r.logger.Fatal().Msg("job file not specified")
		}

		file, err := os.Open(jobFile)
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not open job file")
		}
		defer file.Close()

		contents, err := io.ReadAll(file)
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not read job file")
		}

		encodedJob := base64.StdEncoding.EncodeToString(contents)

		err = r.setupTaskRequest(encodedJob, args[0])
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not setup task")
		}
	},
}

func (r *Runner) setupTaskRequest(encodedJob, taskLabel string) error {
	// Define the request body
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "SetupTask",
		"params": []interface{}{
			r.cfg.ManagedConfig.Username,
			encodedJob,
			taskLabel,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "http://localhost:1325/rpc", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer random-user")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return err
}

func init() {
	cmd.RootCmd.AddCommand(runJobManaged)
	runJobManaged.AddCommand(setupTaskCmd)

	setupTaskCmd.Flags().StringVarP(&jobFile, "job", "j", "", "job file")
}
