//package config

//// XXX: Config file should have a version field to manage future changes to schema

//type (
//	// Cedana configuration. Each of the below fields can also be set
//	// through an environment variable with the same name, prefixed, and in uppercase. E.g.
//	// `Metrics.ASR` can be set with `CEDANA_METRICS_ASR`. The `env_aliases` tag below specifies
//	// alternative (alias) environment variable names (comma-separated).
//	Config struct {
//		//Connection settings
//		Connection Connection `json:"connection" key:"connection" yaml:"connection" mapstructure:"connection"`
//		// Address to use for incoming/outgoing connections
//		Address string `json:"address" key:"address" yaml:"address" mapstructure:"address"`
//		// LogLevel is the default log level used by the server
//		LogLevel string `json:"log_level" key:"log_level" yaml:"log_level" mapstructure:"log_level"`
//	}
//	Connection struct {
//		// URL is your unique Cedana endpoint URL
//		URL string `json:"url" key:"url" yaml:"url" mapstructure:"url" env_aliases:"CEDANA_URL"`
//		// AuthToken is your authentication token for the Cedana endpoint
//		AuthToken string `json:"auth_token" key:"auth_token" yaml:"auth_token" mapstructure:"auth_token" env_aliases:"CEDANA_AUTH_TOKEN"`
//	}
//)
