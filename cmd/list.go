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
	Short: "List all existing components of a resource",
	Long:  `List all existing instances of resource. A resource can be a pod, node or a cluster (Lists cluster by default)`,
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
	Short: "List all existing clusters under given org",
	Long:  `List all existing clusters of a given org.`,
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
	Short: "List all existing nodes under given cluster",
	Long:  `List all existing nodes of a given cluster.`,
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
	Short: "List all existing pods under given namespace of a cluster",
	Long:  `List all existing pods of a given cluster under a specific namespace.`,
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
	listPodCmd.PersistentFlags().String("cluster", "", "The name of the cluster")
	listPodCmd.PersistentFlags().String("namespace", "", "The kubernetes namespace the resource belongs to")
	listNodeCmd.PersistentFlags().String("cluster", "", "The name of the cluster")
	listNodeCmd.PersistentFlags().String("namespace", "", "The kubernetes namespace the resource belongs to")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// podCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
