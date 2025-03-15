/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/cedana/cedana-cli/client"
	"github.com/spf13/cobra"
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		clusters, err := client.ListClusters(cedanaURL, cedanaAuthToken)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Print the clusters
		fmt.Printf("Found %d clusters:\n", len(clusters))
		for _, cluster := range clusters {
			fmt.Printf("- %s: %s\n", cluster.Name, cluster.ID)
		}
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clusterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clusterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
