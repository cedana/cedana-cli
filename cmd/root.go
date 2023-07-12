package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	rootCmd = &cobra.Command{
		Use:   "cedana-cli",
		Short: "Instance brokerage and orchestration system for Cedana",
		Long: `________  _______   ________  ________  ________   ________     
|\   ____\|\  ___ \ |\   ___ \|\   __  \|\   ___  \|\   __  \    
\ \  \___|\ \   __/|\ \  \_|\ \ \  \|\  \ \  \\ \  \ \  \|\  \   
 \ \  \    \ \  \_|/_\ \  \ \\ \ \   __  \ \  \\ \  \ \   __  \  
  \ \  \____\ \  \_|\ \ \  \_\\ \ \  \ \  \ \  \\ \  \ \  \ \  \ 
   \ \_______\ \_______\ \_______\ \__\ \__\ \__\\ \__\ \__\ \__\
    \|_______|\|_______|\|_______|\|__|\|__|\|__| \|__|\|__|\|__|
                                                                 
                                                                 
                                                                 ` + "\n Instance Brokerage and Orchestration System." +
			"\n Property of Cedana, Corp.",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize()
}
