/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/cedana/cedana-cli/client"
	"github.com/spf13/cobra"
)

// nodeCmd represents the node command
var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		clusterName, err := cmd.Flags().GetString("cluster")
		if err != nil {
			fmt.Printf("Error retrieving cluster flag: %v\n", err)
			return
		}
		nodes, err := client.GetClusterNodes(clusterName, cedanaURL, cedanaAuthToken)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Print the nodes
		fmt.Printf("Found %d nodes:\n", len(nodes))
		for _, node := range nodes {
			fmt.Printf("- %s (%s): %s\n", node.Name, node.InstanceType, node.ID)
		}
	},
}

func init() {
	rootCmd.AddCommand(nodeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	nodeCmd.PersistentFlags().String("cluster", "", "A help for foo")
	nodeCmd.MarkFlagRequired("cluster")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// nodeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
