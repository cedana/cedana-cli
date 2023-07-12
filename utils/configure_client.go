package utils

import (
	cedana "github.com/nravic/cedana-orch/types"
)

type ConfigClient struct {
	CedanaManaged bool          `json:"cedana_managed" mapstructure:"cedana_managed"`
	Client        Client        `json:"client" mapstructure:"client"`
	ActionScripts ActionScripts `json:"action_scripts" mapstructure:"action_scripts"`
	Connection    Connection    `json:"connection" mapstructure:"connection"`
	Docker        Docker        `json:"docker" mapstructure:"docker"`
	SharedStorage SharedStorage `json:"shared_storage" mapstructure:"shared_storage"`
}

type Client struct {
	ProcessName          string `json:"process_name" mapstructure:"process_name"`
	LeaveRunning         bool   `json:"leave_running" mapstructure:"leave_running"`
	SignalProcessPreDump bool   `json:"signal_process_pre_dump" mapstructure:"signal_process_pre_dump"`
	SignalProcessTimeout int    `json:"signal_process_timeout" mapstructure:"signal_process_timeout"`
}

type ActionScripts struct {
	PreDump    string `json:"pre_dump" mapstructure:"pre_dump"`
	PostDump   string `json:"post_dump" mapstructure:"post_dump"`
	PreRestore string `json:"pre_restore" mapstructure:"pre_restore"`
}

type Docker struct {
	LeaveRunning  bool   `json:"leave_running" mapstructure:"leave_running"`
	ContainerName string `json:"container_name" mapstructure:"container_name"`
	CheckpointID  string `json:"checkpoint_id" mapstructure:"checkpoint_id"`
}

type SharedStorage struct {
	// only useful for multi-machine checkpoint/restore
	MountPoint     string `json:"mount_point" mapstructure:"mount_point"`
	DumpStorageDir string `json:"dump_storage_dir" mapstructure:"dump_storage_dir"`
}

// client config builder
func BuildClientConfig(jobFile *cedana.JobFile) *ConfigClient {
	logger := GetLogger()

	cfg, err := InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not get cedana config")
	}

	dir := cfg.SharedStorage.DumpStorageDir
	if dir == "" {
		dir = "~/.cedana/"
	}

	c := &ConfigClient{
		CedanaManaged: true,
		Client: Client{
			LeaveRunning: true, // minimally invasive
		},
		Connection: Connection{
			NATSUrl:   cfg.Connection.NATSUrl,
			NATSPort:  cfg.Connection.NATSPort,
			AuthToken: cfg.Connection.AuthToken,
		},
		SharedStorage: SharedStorage{
			MountPoint:     cfg.SharedStorage.MountPoint,
			DumpStorageDir: cfg.SharedStorage.DumpStorageDir,
		},
	}

	if jobFile.Task.C != nil {
		c.Client.ProcessName = jobFile.Task.C[0]
	}
	return c
}
