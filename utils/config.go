package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/viper"
)

var ValidProviders = []string{
	"aws",
	"gcp",
	"azure",
	"paperspace",
	"local",
}

type CedanaConfig struct {
	MarketServiceUrl string           `json:"market_service_url" mapstructure:"market_service_url"`
	AuthToken        string           `json:"auth_token" mapstructure:"auth_token"`
	EnabledProviders []string         `json:"enabled_providers" mapstructure:"enabled_providers"`
	KeepRunning      bool             `json:"keep_running" mapstructure:"keep_running"`
	AWSConfig        AWSConfig        `json:"aws" mapstructure:"aws"`
	PaperspaceConfig PaperspaceConfig `json:"paperspace" mapstructure:"paperspace"`
}

type AWSConfig struct {
	AccessKeyID             string   `json:"access_key_id" mapstructure:"access_key_id"`
	SecretAccessKey         string   `json:"secret_access_key" mapstructure:"secret_access_key"`
	EnabledRegions          []string `json:"enabled_regions" mapstructure:"enabled_regions"`
	EnabledInstanceFamilies []string `json:"enabled_instance_families" mapstructure:"enabled_instance_families"`
}

type PaperspaceConfig struct {
	APIKey         string   `json:"api_key" mapstructure:"api_key"`
	SSHKeyPath     string   `json:"ssh_key_path" mapstructure:"ssh_key_path"`
	EnabledRegions []string `json:"enabled_regions" mapstructure:"enabled_regions"`
	TemplateId     string   `json:"template_id" mapstructure:"template_id"`
	User           string   `json:"user" mapstructure:"user"`
}

/*
configFile represents an override to the location of the cedana config file
*/
var configFile string = ""

/*
SetConfigFile overrides the path to the cedana config file
*/
func SetConfigFile(c string) {
	configFile = c
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

	viper.SetConfigType("json")
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(filepath.Join(homedir, ".cedana/"))
		// change config if dev environment
		if os.Getenv("CEDANA_ENV") == "dev" {
			viper.SetConfigName("cedana_config_dev")
		} else {
			viper.SetConfigName("cedana_config")
		}
	}

	viper.AutomaticEnv()

	var config CedanaConfig
	err = viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %s. Make sure that config exists and that it's formatted correctly", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &config, nil
}

// Used in bootstrap to create a placeholder config
func CreateCedanaConfig(path, username string) error {
	sc := &CedanaConfig{
		MarketServiceUrl: "https://market.cedana.com",
		AuthToken:        "",
		EnabledProviders: []string{"aws"},
		AWSConfig:        AWSConfig{},
		PaperspaceConfig: PaperspaceConfig{},
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
