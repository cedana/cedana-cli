/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/cedana/cedana-cli/client"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
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

var listClusterCmd = &cobra.Command{
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

// nodeCmd represents the node command
var listNodeCmd = &cobra.Command{
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

// podCmd represents the pod command
var listPodCmd = &cobra.Command{
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
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listPodCmd)
	listCmd.AddCommand(listClusterCmd)
	listCmd.AddCommand(listNodeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// podCmd.PersistentFlags().String("foo", "", "A help for foo")
	listPodCmd.PersistentFlags().String("cluster", "", "A help for foo")
	listPodCmd.PersistentFlags().String("namespace", "", "A help for foo")
	listNodeCmd.PersistentFlags().String("cluster", "", "A help for foo")
	listNodeCmd.PersistentFlags().String("namespace", "", "A help for foo")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// podCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
