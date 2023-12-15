package managed

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cedana/cedana-cli/utils"
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrap system for access to Cedana",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.InitCedanaConfig()
		if err != nil {
			return err
		}

		user := cfg.ManagedConfig.Username
		if user == "" {
			return fmt.Errorf("username not set in config")
		}

		logger := utils.GetLogger()
		logger.Info().Msg("bootstrapping system with sample config...")

		// TODO: should take cues for other bootstraps
		err = createConfig()
		if err != nil {
			logger.Fatal().Err(err).Msg("could not create config")
		}

		r := BuildRunner()

		err = bootstrapRequest(user)
		if err != nil {
			return err
		}

		err = setCredentialsRequest(user)
		if err != nil {
			return err
		}

		return err
	},
}

func createConfig() error {
	homeDir := os.Getenv("HOME")
	configFolderPath := filepath.Join(homeDir, ".cedana")
	// check that $HOME/.cedana folder exists - create if it doesn't
	_, err := os.Stat(configFolderPath)
	if err != nil {
		err = os.Mkdir(configFolderPath, 0o755)
		if err != nil {
			return err
		}
	}

	_, err = os.OpenFile(filepath.Join(homeDir, "/.cedana/cedana_config.json"), 0, 0o644)
	if errors.Is(err, os.ErrNotExist) {
		// copy template, use viper to set programatically
		err = utils.CreateCedanaConfig(filepath.Join(configFolderPath, "cedana_config.json"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) bootstrapRequest(user string) error {
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "Bootstrap",
		"params": []interface{}{
			user,
			[]map[string]interface{}{
				{
					"name":    "aws",
					"regions": []string{"us-east-1"},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := r.cfg.ManagedConfig.MarketServiceUrl + "/rpc"

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user))

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return err
	}

	// Read the response body
	respBody, err := json.Marshal(resp.Body)
	if err != nil {
		return err
	}

	// Print the response body
	fmt.Printf("Bootstrap completed with response: %s\n", string(respBody))

	return nil
}

func (r *Runner) setCredentialsRequest(user string) error {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "SetCredentials",
		"params": []interface{}{
			"random-user",
			"aws",
			os.Getenv("CLOUD_ACCESS_KEY_ID"),
			os.Getenv("CLOUD_SECRET_ACCESS_KEY"),
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := r.cfg.ManagedConfig.MarketServiceUrl + "/rpc"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user))

	return err
}

func init() {
	managedCmd.AddCommand(bootstrapCmd)
}
