package managed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

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

func bootstrapRequest(user string) error {
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

	// Create the HTTP request
	req, err := http.NewRequest("POST", "http://localhost:1325/rpc", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer random-user")

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

func setCredentialsRequest(user string) error {
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

	req, err := http.NewRequest("POST", "http://localhost:1325/rpc", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer random-user")

	return err
}

func init() {
	managedCmd.AddCommand(bootstrapCmd)
}
