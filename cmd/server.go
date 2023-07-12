package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/cedana/cedana-cli/server"
	"github.com/cedana/cedana-cli/utils"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// We want to reuse the orchestrator code to act as a MiTM listener
// between the NATS orchestrator and clients. Helpful for debugging and
// seeing what's going on.
var serverCmd = &cobra.Command{
	Use:    "server",
	Short:  "Start the background NATS orchestration server",
	Hidden: true, // command is used by docker container or a systemd service and not directly run
	RunE: func(cmd *cobra.Command, args []string) error {
		orchId, exists := os.LookupEnv("CEDANA_ORCH_ID")
		if !exists {
			return fmt.Errorf("CEDANA_ORCH_ID not set")
		}
		jobId, exists := os.LookupEnv("CEDANA_JOB_ID")
		if !exists {
			return fmt.Errorf("CEDANA_JOB_ID not set")
		}

		clientId, exists := os.LookupEnv("CEDANA_CLIENT_ID")
		if !exists {
			return fmt.Errorf("CEDANA_CLIENT_ID not set")
		}

		config, err := utils.InitCedanaConfig()
		if err != nil {
			return err
		}

		logger := utils.GetLogger()

		opts := []nats.Option{nats.Name(fmt.Sprintf("Cedana orchestrator %s", "cedana_orchestrator"))}
		opts = setupConnOptions(opts, &logger)

		opts = append(opts, nats.Token(config.Connection.AuthToken))

		nc, err := nats.Connect(config.Connection.NATSUrl, opts...)
		if err != nil {
			return fmt.Errorf("could not connect to NATS: %v", err)
		}

		orch := server.NewOrchestrator(
			orchId,
			jobId,
			clientId,
			nc,
			&logger,
		)

		err = orch.Start()
		if err != nil {
			return err
		}

		return nil
	},
}

func setupConnOptions(opts []nats.Option, logger *zerolog.Logger) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectHandler(func(nc *nats.Conn) {
		logger.Info().Msgf("Disconnected: will attempt reconnects for %.0fm", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		logger.Info().Msgf("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		logger.Info().Msgf("Exiting: %v", nc.LastError())
	}))

	return opts
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
