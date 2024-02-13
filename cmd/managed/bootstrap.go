package managed

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cedana/cedana-cli/utils"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"golang.org/x/term"

	ory "github.com/ory/client-go"
	"github.com/ory/x/cmdx"
	"github.com/ory/x/stringsx"
	"github.com/tidwall/sjson"
)

var registerCmd = &cobra.Command{
	Use:   "login",
	Short: "login user with managed platform for access to Cedana",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.GetLogger()

		err := createConfig()
		if err != nil {
			logger.Fatal().Err(err).Msg("could not create config")
		}

		r := BuildRunner()

		// using arg, set url
		if args[0] == "" {
			return fmt.Errorf("no url provided")
		}

		viper.Set("managed_config.market_service_url", args[0])
		err = viper.WriteConfig()
		if err != nil {
			logger.Fatal().Err(err).Msg("could not write config")
		}

		// reload config
		r.cfg, err = utils.InitCedanaConfig()
		if err != nil {
			logger.Fatal().Err(err).Msg("could not set up config")
		}

		// if authToken unset, redirect user and get token
		if r.cfg.ManagedConfig.AuthToken == "" {
			sessionToken, err := r.signin(cmd, cmd.Context(), "")
			if err != nil {
				return err
			}
			fmt.Printf("sessionToken: %s\n", sessionToken)
		}

		return err
	},
}

// returns sessionToken
func (r *Runner) signin(cmd *cobra.Command, ctx context.Context, sessionToken string) (string, error) {
	// init new ory client
	cfg := ory.NewConfiguration()
	cfg.Servers = ory.ServerConfigurations{
		{
			URL: "https://auth.cedana.com",
		},
	}
	oryClient := ory.NewAPIClient(cfg)
	req := oryClient.FrontendAPI.CreateNativeLoginFlow(ctx)
	if len(sessionToken) > 0 {
		req = req.XSessionToken(sessionToken).Aal("aal2")
	}

	flow, _, err := req.Execute()
	if err != nil {
		return "", err
	}

	var form interface{} = &ory.UpdateLoginFlowWithPasswordMethod{}
	method := "password"
	if len(sessionToken) > 0 {
		var foundTOTP bool
		var foundLookup bool
		for _, n := range flow.Ui.Nodes {
			if n.Group == "totp" {
				foundTOTP = true
			} else if n.Group == "lookup_secret" {
				foundLookup = true
			}
		}
		if !foundLookup && !foundTOTP {
			return "", errors.New("only TOTP and lookup secrets are supported for two-step verification in the CLI")
		}

		method = "lookup_secret"
		if foundTOTP {
			form = &ory.UpdateLoginFlowWithTotpMethod{}
			method = "totp"
		}
	}

	type PasswordReader struct{}

	pwReader := func() ([]byte, error) {
		return term.ReadPassword(int(os.Stdin.Fd()))
	}
	if p, ok := cmd.Context().Value(PasswordReader{}).(passwordReader); ok {
		pwReader = p
	}

	if err := renderForm(bufio.NewReader(cmd.InOrStdin()), pwReader, cmd.ErrOrStderr(), flow.Ui, method, form); err != nil {
		return "", err
	}

	var body ory.UpdateLoginFlowBody
	switch e := form.(type) {
	case *ory.UpdateLoginFlowWithTotpMethod:
		body.UpdateLoginFlowWithTotpMethod = e
	case *ory.UpdateLoginFlowWithPasswordMethod:
		body.UpdateLoginFlowWithPasswordMethod = e
	default:
		panic("unexpected type")
	}

	login, _, err := oryClient.FrontendAPI.UpdateLoginFlow(ctx).XSessionToken(sessionToken).
		Flow(flow.Id).UpdateLoginFlowBody(body).Execute()
	if err != nil {
		return "", err
	}

	sessionToken = stringsx.Coalesce(*login.SessionToken, sessionToken)
	_, _, err = oryClient.FrontendAPI.ToSession(ctx).XSessionToken(sessionToken).Execute()
	if err == nil {
		return sessionToken, nil
	}

	if e, ok := err.(interface{ Body() []byte }); ok {
		switch gjson.GetBytes(e.Body(), "error.id").String() {
		case "session_aal2_required":
			return r.signin(cmd, ctx, sessionToken)
		}
	}
	return "", err
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrap cedana with cloud providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := BuildRunner()

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

		r.logger.Info().Msgf("cinfo = %+v", cInfo)
		err := r.bootstrap(cInfo, true)
		if err != nil {
			return err
		}

		for _, info := range cInfo {
			switch info.Name {
			case "aws":
				r.logger.Info().Msgf("setting credentials for AWS")
				err = r.setCredentialsAWS()
				if err != nil {
					return err
				}
			}
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

func (r *Runner) register(email string) (*registerResponse, error) {
	reg := registerRequest{
		Email: email,
	}

	jsonBody, err := json.Marshal(reg)
	if err != nil {
		return nil, err
	}

	url := r.cfg.ManagedConfig.MarketServiceUrl + "/registration"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var regResp registerResponse
	err = json.Unmarshal(body, &regResp)
	if err != nil {
		return nil, err
	}

	r.logger.Info().Msgf("Registered user email %s, received unique token %s", reg.Email, regResp.Token)
	r.logger.Info().Msgf("setting info in config...")

	viper.Set("managed_config.username", reg.Email)
	viper.Set("managed_config.user_id", regResp.Owner)

	viper.WriteConfig()

	return &regResp, nil

}

type validateRegistrationRequest struct {
	Password string `json:"password"`
	Confirm  string `json:"confirm_password"`
	Token    string `json:"token"`
}

func (r *Runner) validateRegistration(password, confirm, uid, token string) error {
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

	url := r.cfg.ManagedConfig.MarketServiceUrl + "/registration/" + uid + "/validation"

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

type CloudInfo struct {
	Name    string   `json:"name"`
	Regions []string `json:"regions"`
}

type bootstrapRequest struct {
	SessionToken string      `json:"-"`
	CloudInfo    []CloudInfo `json:"cloud_info"`
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

	url := r.cfg.ManagedConfig.MarketServiceUrl + "/" + "/bootstrap"

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

	url := r.cfg.ManagedConfig.MarketServiceUrl + "/" + "/cloud/" + "aws" + "/credentials"

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	r.logger.Info().Msgf("AWS credentials set with response %s", string(body))

	return nil
}

type passwordReader = func() ([]byte, error)

func getLabel(attrs *ory.UiNodeInputAttributes, node *ory.UiNode) string {
	if attrs.Name == "identifier" {
		return fmt.Sprintf("%s: ", "Email")
	} else if node.Meta.Label != nil {
		return fmt.Sprintf("%s: ", node.Meta.Label.Text)
	} else if attrs.Label != nil {
		return fmt.Sprintf("%s: ", attrs.Label.Text)
	}
	return fmt.Sprintf("%s: ", attrs.Name)
}

func renderForm(stdin *bufio.Reader, pwReader passwordReader, stderr io.Writer, ui ory.UiContainer, method string, out interface{}) (err error) {
	for _, message := range ui.Messages {
		_, _ = fmt.Fprintf(stderr, "%s\n", message.Text)
	}

	for _, node := range ui.Nodes {
		for _, message := range node.Messages {
			_, _ = fmt.Fprintf(stderr, "%s\n", message.Text)
		}
	}

	values := json.RawMessage(`{}`)
	for k := range ui.Nodes {
		node := ui.Nodes[k]
		if node.Group != method && node.Group != "default" {
			continue
		}

		switch node.Type {
		case "input":
			attrs := node.Attributes.UiNodeInputAttributes
			switch attrs.Type {
			case "button":
				continue
			case "submit":
				continue
			}

			if attrs.Name == "traits.consent.tos" {
				for {
					ok, err := cmdx.AskScannerForConfirmation(getLabel(attrs, &node), stdin, stderr)
					if err != nil {
						return err
					}
					if ok {
						break
					}
				}
				values, err = sjson.SetBytes(values, attrs.Name, time.Now().UTC().Format(time.RFC3339))
				if err != nil {
					return err
				}
				continue
			}

			if strings.Contains(attrs.Name, "traits.details") {
				continue
			}

			switch attrs.Type {
			case "hidden":
				continue
			case "checkbox":
				result, err := cmdx.AskScannerForConfirmation(getLabel(attrs, &node), stdin, stderr)
				if err != nil {
					return err
				}

				values, err = sjson.SetBytes(values, attrs.Name, result)
				if err != nil {
					return err
				}
			case "password":
				var password string
				for password == "" {
					_, _ = fmt.Fprint(stderr, getLabel(attrs, &node))
					v, err := pwReader()
					if err != nil {
						return err
					}
					password = strings.ReplaceAll(string(v), "\n", "")
					fmt.Println("")
				}

				values, err = sjson.SetBytes(values, attrs.Name, password)
				if err != nil {
					return err
				}
			default:
				var value string
				for value == "" {
					_, _ = fmt.Fprint(stderr, getLabel(attrs, &node))
					v, err := stdin.ReadString('\n')
					if err != nil {
						return err
					}
					value = strings.ReplaceAll(v, "\n", "")
				}

				values, err = sjson.SetBytes(values, attrs.Name, value)
				if err != nil {
					return err
				}
			}
		default:
			// Do nothing
		}
	}

	values, err = sjson.SetBytes(values, "method", method)
	if err != nil {
		return err
	}

	return err
}

func init() {
	managedCmd.AddCommand(registerCmd)
	managedCmd.AddCommand(bootstrapCmd)
}
