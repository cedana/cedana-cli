# Configuration 

`cedana-cli` configuration lives in in ~/.cedana/cli-config.json. This file is automatically created the first time you use a `cedana-cli` command. You can also create it manually.

You may also override the configuration using environment variables. The environment variables are prefixed with `CEDANA_CLI` and are in uppercase. For example, `connection.url` can be set with `CEDANA_CLI_CONNECTION_URL`.


```go
type (
	// Cedana configuration. Each of the below fields can also be set
	// through an environment variable with the same name, prefixed, and in uppercase. E.g.
	// `Metrics.ASR` can be set with `CEDANA_METRICS_ASR`. The `env_aliases` tag below specifies
	// alternative (alias) environment variable names (comma-separated).
	Config struct {
		// LogLevel is the default log level used by the server
		LogLevel string `json:"log_level" key:"log_level" yaml:"log_level" mapstructure:"log_level"`
		// Connection settings
		Connection Connection `json:"connection" key:"connection" yaml:"connection" mapstructure:"connection"`
	}

	Connection struct {
		// URL is your unique Cedana endpoint URL
		URL string `json:"url" key:"url" yaml:"url" mapstructure:"url" env_aliases:"CEDANA_URL"`
		// AuthToken is your authentication token for the Cedana endpoint
		AuthToken string `json:"auth_token" key:"auth_token" yaml:"auth_token" mapstructure:"auth_token" env_aliases:"CEDANA_AUTH_TOKEN"`
	}
)
```
