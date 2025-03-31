/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/cedana/cedana-cli/client"
	"github.com/cedana/cedana/pkg/style"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// Parent list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all existing components of a resource",
	Args:  cobra.ArbitraryArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var listClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "List all active managed clusters for the organization",
	Run: func(cmd *cobra.Command, args []string) {
		clusters, err := client.ListClusters()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// TODO pretty print
		fmt.Printf("Found %d clusters:\n", len(clusters))
		for _, cluster := range clusters {
			fmt.Printf("- %s: %s\n", cluster.Name, cluster.ID)
		}
	},
}

var listNodeCmd = &cobra.Command{
	Use:     "node",
	Short:   "List all existing nodes under given cluster",
	Aliases: []string{"ls"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, err := cmd.Flags().GetString("cluster")
		if err != nil {
			return fmt.Errorf("failed to get cluster flag: %w", err)
		}

		nodes, err := client.GetClusterNodes(clusterName)
		if err != nil {
			return err
		}

		if len(nodes) == 0 {
			fmt.Println("No nodes to show")
			return nil
		}

		tableWriter := table.NewWriter()
		tableWriter.SetStyle(style.TableStyle)
		tableWriter.SetOutputMirror(os.Stdout)
		tableWriter.Style().Options.SeparateRows = false

		tableWriter.AppendHeader(table.Row{
			"Name",
			"Instance Type",
			"ID",
		})

		for _, node := range nodes {
			tableWriter.AppendRow(table.Row{
				node.Name,
				node.InstanceType,
				node.ID,
			})
		}

		tableWriter.Render()

		fmt.Printf("\n%d nodes found\n", len(nodes))
		return nil
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
		pods, err := client.GetClusterPods(clusterName, clusterNamespace)
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
