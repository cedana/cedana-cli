package utils

import (
	cedana "github.com/cedana/cedana-cli/types"
	core "github.com/cedana/cedana/utils"
)

// Used to passthrough cedana configuration to the daemon
func BuildClientConfig(jobFile *cedana.JobFile) *core.Config {
	logger := GetLogger()

	cfg, err := InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not get cedana config")
	}

	dir := cfg.SharedStorage.DumpStorageDir
	if dir == "" {
		dir = "~/.cedana/"
	}

	c := &core.Config{
		CedanaManaged: true,
		Client: core.Client{
			LeaveRunning: true, // minimally invasive
		},
		Connection: core.Connection{
			NATSUrl:       cfg.Connection.NATSUrl,
			NATSPort:      cfg.Connection.NATSPort,
			NATSAuthToken: cfg.Connection.AuthToken,
		},
		SharedStorage: core.SharedStorage{
			DumpStorageDir: cfg.SharedStorage.DumpStorageDir,
		},
	}

	if jobFile.Task.C != nil {
		c.Client.Task = jobFile.Task.C[0]
	}
	return c
}
