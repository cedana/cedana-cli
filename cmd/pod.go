/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/cedana/cedana-cli/client"
	"github.com/spf13/cobra"
)

// podCmd represents the pod command
var podCmd = &cobra.Command{
	Use:   "pod",
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
		clusterNamespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			fmt.Printf("Error retrieving namespace flag: %v\n", err)
			return
		}
		pods, err := client.GetClusterPods(clusterName, clusterNamespace, cedanaURL, cedanaAuthToken)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Found %d pods in namespace %s :\n", len(pods), clusterNamespace)
		for _, pod := range pods {
			fmt.Printf("%s : %s\n",
				pod.Name,
				//		pod.ID,
				//		pod.Namespace,
				pod.Status,
				//		pod.NodeID,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(podCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// podCmd.PersistentFlags().String("foo", "", "A help for foo")
	podCmd.PersistentFlags().String("cluster", "", "A help for foo")
	podCmd.PersistentFlags().String("namespace", "", "A help for foo")
	podCmd.MarkFlagRequired("cluster")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// podCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
