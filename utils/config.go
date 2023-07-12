package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/viper"
	"k8s.io/utils/strings/slices"
)

var ValidProviders = []string{
	"aws",
	"gcp",
	"azure",
	"paperspace",
}

type CedanaConfig struct {
	SelfServe        bool     `json:"self_serve" mapstructure:"self_serve"`
	EnabledProviders []string `json:"enabed_providers" mapstructure:"enabled_providers"`
	// configures cedana to spin up a new instance and restore on instance failure
	KeepRunning bool `json:"keep_running" mapstructure:"keep_running"`

	// provider specific configurations
	AWSConfig        AWSConfig        `json:"aws" mapstructure:"aws"`
	PaperspaceConfig PaperspaceConfig `json:"paperspace" mapstructure:"paperspace"`

	Checkpoint    Checkpoint          `json:"checkpoint" mapstructure:"checkpoint"`
	SharedStorage SharedStorageConfig `json:"shared_storage" mapstructure:"shared_storage"`
	Connection    Connection          `json:"connection" mapstructure:"connection"`
}

type Connection struct {
	NATSUrl   string `json:"nats_url" mapstructure:"nats_url"`
	NATSPort  int    `json:"nats_port" mapstructure:"nats_port"`
	AuthToken string `json:"auth_token" mapstructure:"auth_token"`
}

type AWSConfig struct {
	SSHKeyPath              string   `json:"ssh_key_path" mapstructure:"ssh_key_path"` // path to AWS identity key
	EnabledRegions          []string `json:"enabled_regions" mapstructure:"enabled_regions"`
	EnabledInstanceFamilies []string `json:"enabled_instance_families" mapstructure:"enabled_instance_families"`
	LaunchTemplateName      string   `json:"launch_template" mapstructure:"launch_template"`
	ImageId                 string   `json:"image_id" mapstructure:"image_id"` // AMI image id
	User                    string   `json:"user" mapstructure:"user"`         // user if using a custom AMI
}

type PaperspaceConfig struct {
	APIKey         string   `json:"api_key" mapstructure:"api_key"`
	SSHKeyPath     string   `json:"ssh_key_path" mapstructure:"ssh_key_path"`
	EnabledRegions []string `json:"enabled_regions" mapstructure:"enabled_regions"`
	TemplateId     string   `json:"template_id" mapstructure:"template_id"`
	User           string   `json:"user" mapstructure:"user"`
}

type SharedStorageConfig struct {
	DumpStorageDir string `json:"dump_storage_dir" mapstructure:"dump_storage_dir"`
	MountPoint     string `json:"mount_point" mapstructure:"mount_point"`
}

type Checkpoint struct {
	HeartbeatEnabled  bool `json:"heartbeat_enabled" mapstructure:"heartbeat_enabled"`
	HeartbeatInterval int  `json:"heartbeat_interval_seconds" mapstructure:"heartbeat_interval_seconds"`
}

func InitCedanaConfig() (*CedanaConfig, error) {
	// we want absolute paths for the config, and sometimes (if deployed in the cloud for instance)
	// this gets run as root.

	// Hack to get around putting the config under the user account.
	// TODO NR - needs fixing
	var username string
	username = os.Getenv("SUDO_USER")
	if username == "" {
		username = os.Getenv("USER")
	}

	u, err := user.Lookup(username)
	if err != nil {
		return nil, err
	}

	homedir := u.HomeDir

	viper.AddConfigPath(filepath.Join(homedir, ".cedana/"))
	viper.SetConfigType("json")
	// change config if dev environment
	if os.Getenv("CEDANA_ENV") == "dev" {
		viper.SetConfigName("cedana_config_dev")
	} else {
		viper.SetConfigName("cedana_config")
	}

	viper.AutomaticEnv()

	var config CedanaConfig
	err = viper.ReadInConfig()
	if err != nil {
		panic("fatal error loading config file. Make sure that config exists in $HOME/.cedana/cedana_config.json .")
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Println(err)
		return nil, err
	}

	if err := isEnabledProvidersValid(config); err != nil {
		return nil, err
	}

	return &config, nil
}

func isEnabledProvidersValid(config CedanaConfig) error {
	var invalid int = 0
	for _, enabled := range config.EnabledProviders {
		if !slices.Contains(ValidProviders, enabled) {
			invalid++
		}
	}
	if invalid > 0 {
		return fmt.Errorf("Invalid providers: %v", invalid)
	}
	return nil
}

// Used in bootstrap to create a placeholder config
func CreateCedanaConfig(path string, regions []string) error {
	sc := &CedanaConfig{
		AWSConfig: AWSConfig{
			EnabledRegions:     regions,
			SSHKeyPath:         "~/home/path-to-aws-key",
			LaunchTemplateName: "foo",
		},
		Connection: Connection{
			NATSUrl:  "demo.nats.io",
			NATSPort: 8222,
		},
	}

	// marshal sc into path
	b, err := json.Marshal(sc)
	if err != nil {
		return fmt.Errorf("err: %v, could not marshal spot config struct to file", err)
	}
	err = os.WriteFile(path, b, 0o644)
	if err != nil {
		return fmt.Errorf("err: %v, could not write file to path %s", err, path)
	}

	return err
}
