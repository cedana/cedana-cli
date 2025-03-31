package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cedana/cedana/pkg/utils"
	"github.com/spf13/viper"
)

const (
	DIR_NAME   = ".cedana"
	FILE_NAME  = "cli-config"
	FILE_TYPE  = "json"
	DIR_PERM   = 0o755
	FILE_PERM  = 0o644
	ENV_PREFIX = "CEDANA_CLI"

	DEFAULT_SOCK_PERMS = 0o666

	DEFAULT_LOG_LEVEL = "info"
)

// The default global config. This will get overwritten
// by the config file or env vars during startup, if they exist.
var Global Config = Config{
	// NOTE: Don't specify default address here as it depends on default protocol.
	// Use above constants for default address for each protocol.
	Connection: Connection{
		URL:       "",
		AuthToken: "",
	},
}

func init() {
	setDefaults()
	bindEnvVars()
	viper.Unmarshal(&Global)
}

type InitArgs struct {
	Config    string
	ConfigDir string
}

func Init(args InitArgs) error {
	user, err := utils.GetUser()
	if err != nil {
		return err
	}

	var configDir string
	if args.ConfigDir == "" {
		homeDir := user.HomeDir
		configDir = filepath.Join(homeDir, DIR_NAME)
	} else {
		configDir = args.ConfigDir
	}

	viper.AddConfigPath(configDir)
	viper.SetConfigPermissions(FILE_PERM)
	viper.SetConfigType(FILE_TYPE)
	viper.SetConfigName(FILE_NAME)

	// Create config directory if it does not exist
	_, err = os.Stat(configDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(configDir, DIR_PERM)
		if err != nil {
			return err
		}
	}
	uid, _ := strconv.Atoi(user.Uid)
	gid, _ := strconv.Atoi(user.Gid)
	os.Chown(configDir, uid, gid)

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("Config file %s is either outdated or invalid. Please delete or update it: %w", viper.ConfigFileUsed(), err)
		}
	}

	if args.Config != "" {
		reader := strings.NewReader(args.Config)
		err = viper.MergeConfig(reader)
		if err != nil {
			return fmt.Errorf("Provided config string is invalid: %w", err)
		}
	} else {
		err = viper.SafeWriteConfig() // Will only overwrite if file does not exist, ignore other errors
		if err != nil {
			if _, ok := err.(viper.ConfigFileAlreadyExistsError); !ok {
				return fmt.Errorf("Failed to write config file: %w", err)
			}
		}
	}

	err = viper.UnmarshalExact(&Global)
	if err != nil {
		return fmt.Errorf("Config file %s is either outdated or invalid. Please delete or update it: %w", viper.ConfigFileUsed(), err)
	}

	return nil
}

// Loads the global defaults into viper
func setDefaults() {
	for _, field := range utils.ListLeaves(Config{}) {
		tag := utils.GetTag(Config{}, field, FILE_TYPE)
		defaultVal := utils.GetValue(Global, field)
		viper.SetDefault(tag, defaultVal)
	}
	viper.SetTypeByDefaultValue(true)
}

// Add bindings for env vars so env vars can be used as backup
// when a value is not found in config. Goes through all the json keys
// in the config type and binds an env var for it. The env var
// is prefixed with the envVarPrefix, all uppercase.
//
// Example: The field `cli.wait_for_ready` will bind to env var `CEDANA_CLI_WAIT_FOR_READY`.
func bindEnvVars() {
	for _, field := range utils.ListLeaves(Config{}) {
		tag := utils.GetTag(Config{}, field, FILE_TYPE)
		envVar := ENV_PREFIX + "_" + strings.ToUpper(strings.ReplaceAll(tag, ".", "_"))

		// get env aliases from struct tag
		aliasesStr := utils.GetTag(Config{}, field, "env_aliases")
		aliases := []string{tag, envVar}
		aliases = append(aliases, strings.Split(aliasesStr, ",")...)

		viper.MustBindEnv(aliases...)
	}

	viper.AutomaticEnv()
}
