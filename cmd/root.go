package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	cedanaConfigFile string
	cedanaURL        string
	cedanaAuthToken  string
	logger           zerolog.Logger
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
	}
)

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

// init initializes the command and flags
func init() {
	// Initialize logger
	logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cedanaConfigFile, "cedana-config", "c", "", "path to cedana-config json file")
	rootCmd.AddCommand(docGenCmd)
}

// initConfig reads environment variables and initializes the configuration
func initConfig() {
	var ok bool

	// Get Cedana URL from environment
	cedanaURL, ok = os.LookupEnv("CEDANA_URL")
	if !ok {
		logger.Error().Msg("CEDANA_URL not set")
		os.Exit(1)
	}

	// Get auth token from environment
	cedanaAuthToken, ok = os.LookupEnv("CEDANA_AUTH_TOKEN")
	if !ok {
		logger.Error().Msg("CEDANA_AUTH_TOKEN not set")
		os.Exit(1)
	}
}
