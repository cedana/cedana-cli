package cmd

import (
	"context"
	"fmt"

	"github.com/cedana/cedana-cli/pkg/config"
	"github.com/cedana/cedana-cli/pkg/flags"
	"github.com/cedana/cedana-cli/pkg/logging"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	rootCmd = &cobra.Command{
		Use:   "cedana-cli",
		Short: "Instance brokerage and orchestration system for Cedana",
		Long: `
 ________  _______   ________  ________  ________   ________
|\   ____\|\  ___ \ |\   ___ \|\   __  \|\   ___  \|\   __  \
\ \  \___|\ \   __/|\ \  \_|\ \ \  \|\  \ \  \\ \  \ \  \|\  \
 \ \  \    \ \  \_|/_\ \  \ \\ \ \   __  \ \  \\ \  \ \   __  \
  \ \  \____\ \  \_|\ \ \  \_\\ \ \  \ \  \ \  \\ \  \ \  \ \  \
   \ \_______\ \_______\ \_______\ \__\ \__\ \__\\ \__\ \__\ \__\
    \|_______|\|_______|\|_______|\|__|\|__|\|__| \|__|\|__|\|__|

    ` +
			"\n Instance Brokerage, Orchestration and Migration System for Cedana." +
			"\n Property of Cedana, Corp.\n",

		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			conf, _ := cmd.Flags().GetString(flags.ConfigFlag.Full)
			confDir, _ := cmd.Flags().GetString(flags.ConfigDirFlag.Full)
			if err := config.Init(config.InitArgs{
				Config:    conf,
				ConfigDir: confDir,
			}); err != nil {
				return fmt.Errorf("Failed to initialize config: %w", err)
			}

			logging.SetLevel(config.Global.LogLevel)

			return nil
		},
	}
)

func Execute(ctx context.Context, version string) error {
	ctx = log.With().Str("context", "cmd").Logger().WithContext(ctx)

	rootCmd.Version = version
	rootCmd.Long = rootCmd.Long + "\n " + version
	rootCmd.SilenceUsage = true // only show usage when true usage error

	return rootCmd.ExecuteContext(ctx)
}

// init initializes the command and flags
func init() {
	rootCmd.AddCommand(docGenCmd)
	rootCmd.AddCommand(listCmd)
}
