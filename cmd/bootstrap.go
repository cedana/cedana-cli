package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cedana/cedana-cli/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var username string
var password string

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrap cedana with cloud providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := createConfig()
		if err != nil {
			return err
		}

		r := BuildRunner()

		if r.cfg.AuthToken == "" {
			return fmt.Errorf("no auth token detected, please login first with cedana-cli login")
		}

		if r.cfg.MarketServiceUrl == "" {
			return fmt.Errorf("market service URL not set in config")
		}

		if r.cfg.EnabledProviders == nil || len(r.cfg.EnabledProviders) == 0 {
			return fmt.Errorf("no providers specified in config, add provider-specific config and enabled providers, regions and try again.")
		}

		// assemble cloudInfo from enabledProviders
		var cInfo []CloudInfo
		for _, provider := range r.cfg.EnabledProviders {
			var info CloudInfo
			switch provider {
			case "aws":
				info.Name = "aws"
				r.logger.Info().Msgf("setting up AWS...")
				if r.cfg.AWSConfig.EnabledRegions == nil || len(r.cfg.AWSConfig.EnabledRegions) == 0 {
					return fmt.Errorf("no regions specified in config, add regions and try again.")
				}
				info.Regions = r.cfg.AWSConfig.EnabledRegions
			case "azure":
				info.Name = "azure"
				return fmt.Errorf("azure not yet supported")
			case "gcp":
				info.Name = "gcp"
				return fmt.Errorf("gcp not yet supported")
			case "paperspace":
				info.Name = "paperspace"
				if r.cfg.PaperspaceConfig.EnabledRegions == nil || len(r.cfg.PaperspaceConfig.EnabledRegions) == 0 {
					return fmt.Errorf("no regions specified in config, add regions and try again.")
				}
				info.Regions = r.cfg.PaperspaceConfig.EnabledRegions
			}

			cInfo = append(cInfo, info)
		}

		err = r.bootstrap(cInfo, true)
		if err != nil {
			return err
		}

		for _, info := range cInfo {
			switch info.Name {
			case "aws":
				r.logger.Info().Msgf("setting credentials for AWS...")
				err = r.setCredentialsAWS()
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to cedana. Create an account at https://auth.cedana.com/ui/registration",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := BuildRunner()

		if r.cfg.AuthToken != "" {
			err := validateAuthToken()
			if err != nil {
				return err
			}
		}

		// auth token not set, prompt for username and password
		if (username == "") || (password == "") {
			return fmt.Errorf("no username or password specified!")
		}

		// Get UI action flow URL
		actionUrl, err := getActionURL("https://auth.cedana.com/self-service/login/api")
		if err != nil {
			return fmt.Errorf("could not get actionUrl for authentication")
		}

		token, err := authenticate(actionUrl, username, password)
		if err != nil {
			r.logger.Fatal().Err(err).Msgf("could not authenticate with cedana server")
		}

		fmt.Println("Token:", token)

		// set token in config
		viper.Set("auth_token", token)
		err = viper.WriteConfig()
		if err != nil {
			return err
		}

		return nil
	},
}

func validateAuthToken() error {
	return nil
}

func getActionURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	ui, ok := result["ui"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	action, ok := ui["action"].(string)
	if !ok {
		return "", fmt.Errorf("action URL not found")
	}

	return action, nil
}

func authenticate(actionUrl, email, password string) (string, error) {
	authData := map[string]string{
		"identifier": email,
		"password":   password,
		"method":     "password",
	}
	data, err := json.Marshal(authData)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", actionUrl, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	token, ok := result["session_token"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	return token, nil
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
		err = utils.CreateCedanaConfig(filepath.Join(configFolderPath, "cedana_config.json"), username)
		if err != nil {
			return err
		}
	}
	return nil
}

type CloudInfo struct {
	Name    string   `json:"name"`
	Regions []string `json:"regions"`
}

type bootstrapRequest struct {
	SessionToken string      `json:"-"`
	CloudInfo    []CloudInfo `json:"cloud_info"`
}

func (r *Runner) bootstrap(cloudInfo []CloudInfo, leaveRunning bool) error {
	br := bootstrapRequest{
		SessionToken: r.cfg.AuthToken,
		CloudInfo:    cloudInfo,
	}

	jsonBody, err := json.Marshal(br)
	if err != nil {
		return err
	}

	url := r.cfg.MarketServiceUrl + "/" + "/bootstrap"

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

	if err != nil {
		return fmt.Errorf("request failed with status code: %d and error: %s", resp.StatusCode, err.Error())
	}

	r.logger.Info().Msgf("Bootstrap completed")
	return nil
}

type setSupportedcloudRequest struct {
	Name    string   `json:"name"`
	Regions []string `json:"regions"`
}

type setCredentialsRequestAWS struct {
	AccessKeyID string `json:"access_key_id"`
	SecretKey   string `json:"secret_access_key"`
}

func (r *Runner) setCredentialsAWS() error {
	if r.cfg.AWSConfig.AccessKeyID == "" || r.cfg.AWSConfig.SecretAccessKey == "" {
		return fmt.Errorf("AWS credentials not set")
	}

	scr := setCredentialsRequestAWS{
		AccessKeyID: r.cfg.AWSConfig.AccessKeyID,
		SecretKey:   r.cfg.AWSConfig.SecretAccessKey,
	}

	jsonBody, err := json.Marshal(scr)
	if err != nil {
		return err
	}

	url := r.cfg.MarketServiceUrl + "/cloud/" + "aws" + "/credentials"

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
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

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	r.logger.Info().Msgf("aws credentials set, ssh key created!")

	return nil
}

func (r *Runner) getCredentials(cloud string) {

}

func init() {
	RootCmd.AddCommand(bootstrapCmd)
	RootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "username")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "password")
}
