package managed

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
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "register user with managed platform for access to Cedana",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.GetLogger()

		err := createConfig()
		if err != nil {
			logger.Fatal().Err(err).Msg("could not create config")
		}

		r := BuildRunner()

		// if username and pass is unset, prompt for it
		if r.cfg.ManagedConfig.UserID == "" {
			prompt := promptui.Prompt{
				Label: "Enter email address to register with",
			}
			result, err := prompt.Run()
			if err != nil {
				r.logger.Fatal().Err(err).Msg("error reading prompt input")
			}

			token, err := r.register(result)
			if err != nil {
				r.logger.Fatal().Err(err).Msg("could not register user")
			}

			prompt = promptui.Prompt{
				Label: "Enter password",
				Mask:  '*',
			}

			password, err := prompt.Run()
			if err != nil {
				r.logger.Fatal().Err(err).Msg("error reading prompt input")
			}

			confirmPrompt := promptui.Prompt{
				Label: "Confirm password",
				Mask:  '*',
				Validate: func(input string) error {
					if input != password {
						return fmt.Errorf("passwords do not match")
					}
					return nil
				},
			}

			_, err = confirmPrompt.Run()
			if err != nil {
				r.logger.Fatal().Err(err).Msg("error reading prompt input")
			}

			err = r.validateRegistration(password, password, token)
			if err != nil {
				r.logger.Fatal().Err(err).Msg("could not finish registering user")
			}
		}

		if r.cfg.ManagedConfig.AuthToken == "" {
			r.logger.Info().Msgf("JWT Token missing, generating...")
			jwt, err := r.generateJWT(r.cfg.ManagedConfig.Password)
			if err != nil {
				r.logger.Fatal().Err(err).Msg("could not generate JWT token")
			}

			r.logger.Info().Msgf("Generated 24 hour expiring JWT: %s. All further requests are automatically authenticated using this token.", jwt)

			viper.Set("managed_config.auth_token", jwt)
			err = viper.WriteConfig()
			if err != nil {
				r.logger.Fatal().Err(err).Msg("could not write jwt to file")
			}
		}
		return err
	},
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrap cedana with cloud providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := BuildRunner()

		if r.cfg.EnabledProviders == nil || len(r.cfg.EnabledProviders) == 0 {
			return fmt.Errorf("no providers specified in config, add provider-specific config, regions and try again.")
		}

		// assemble cloudInfo from enabledProviders
		var cInfo []CloudInfo
		for _, provider := range r.cfg.EnabledProviders {
			var info CloudInfo
			switch provider {
			case "aws":
				info.Name = "aws"
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

		r.logger.Info().Msgf("sending provider specs to cedana market bootstrap...")
		err := r.bootstrap(cInfo, true)
		if err != nil {
			return err
		}
		return nil
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
		username := ""
		prompt := promptui.Prompt{
			Label: "Enter username",
		}
		username, err = prompt.Run()
		if err != nil {
			return err
		}
		// copy template, use viper to set programatically
		err = utils.CreateCedanaConfig(filepath.Join(configFolderPath, "cedana_config.json"), username)
		if err != nil {
			return err
		}
	}
	return nil
}

type registerRequest struct {
	Email string `json:"email"`
}

type registerResponse struct {
	Token string `json:"token"`
	Owner string `json:"owner"`
}

func (r *Runner) register(email string) (string, error) {
	reg := registerRequest{
		Email: email,
	}

	jsonBody, err := json.Marshal(reg)
	if err != nil {
		return "", err
	}

	url := r.cfg.ManagedConfig.MarketServiceUrl + "/registration"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var regResp registerResponse
	err = json.Unmarshal(body, &regResp)
	if err != nil {
		return "", err
	}

	r.logger.Info().Msgf("Registered user email %s, received unique token %s", reg.Email, regResp.Token)
	r.logger.Info().Msgf("setting info in config...")

	viper.Set("managed_config.username", reg.Email)
	viper.Set("managed_config.user_id", regResp.Owner)

	viper.WriteConfig()

	return regResp.Token, nil

}

type validateRegistrationRequest struct {
	Password string `json:"password"`
	Confirm  string `json:"confirm_password"`
	Token    string `json:"token"`
}

func (r *Runner) validateRegistration(password, confirm, token string) error {
	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	vrr := validateRegistrationRequest{
		Password: password,
		Confirm:  confirm,
		Token:    token,
	}

	jsonBody, err := json.Marshal(vrr)
	if err != nil {
		return err
	}

	url := r.cfg.ManagedConfig.MarketServiceUrl + "/registration/" + r.cfg.ManagedConfig.UserID + "/validation"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	r.logger.Info().Msgf("password set successfully")

	return nil
}

type generateJWTRequest struct {
	Password string `json:"password"`
}

type generateJWTResponse struct {
	JWT string `json:"jwt"`
}

func (r *Runner) generateJWT(password string) (string, error) {
	gjwt := generateJWTRequest{
		Password: password,
	}

	jsonBody, err := json.Marshal(gjwt)
	if err != nil {
		return "", err
	}

	url := r.cfg.ManagedConfig.MarketServiceUrl + "/registration/" + r.cfg.ManagedConfig.UserID + "/jwt"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status code: %d and error: %s", resp.StatusCode, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var gjwtResp generateJWTResponse
	err = json.Unmarshal(body, &gjwtResp)
	if err != nil {
		return "", err
	}

	r.logger.Info().Msgf("Received JWT: %s", gjwtResp.JWT)

	return gjwtResp.JWT, nil
}

type CloudInfo struct {
	Name    string   `json:"name"`
	Regions []string `json:"regions"`
}

type bootstrapRequest struct {
	SessionToken string      `json:"-"`
	CloudInfo    []CloudInfo `json:"cloudInfo"`
	LeaveRunning bool        `json:"leaveRunning"`
}

func (r *Runner) bootstrap(cloudInfo []CloudInfo, leaveRunning bool) error {
	br := bootstrapRequest{
		SessionToken: r.cfg.ManagedConfig.AuthToken,
		CloudInfo:    cloudInfo,
		LeaveRunning: leaveRunning,
	}

	jsonBody, err := json.Marshal(br)
	if err != nil {
		return err
	}

	url := r.cfg.ManagedConfig.MarketServiceUrl + r.cfg.ManagedConfig.UserID + "/bootstrap"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.ManagedConfig.AuthToken)

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

func init() {
	managedCmd.AddCommand(registerCmd)
}
