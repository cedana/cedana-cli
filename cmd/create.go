/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/cedana/cedana-cli/client"
	"github.com/spf13/cobra"
)

// workloadCmd represents the workload command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new resource",
	Long:  `Create a new resource with a json payload`,
	Run: func(cmd *cobra.Command, args []string) {
		payloadPath, err := cmd.Flags().GetString("payload")
		if err != nil {
			fmt.Printf("Error retrieving payload flag: %v\n", err)
			return
		}
		contentType, err := cmd.Flags().GetString("contentType")
		if err != nil {
			fmt.Printf("Error retrieving contentType flag: %v\n", err)
			return
		}
		payloadData, err := os.ReadFile(payloadPath)
		if err != nil {
			fmt.Printf("Error reading payload file %s: %v\n", payloadPath, err)
			return
		}
		resp, err := client.CreateWorkload(payloadData, contentType)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println(resp)
	},
}

var createWorkloadCmd = &cobra.Command{
	Use:   "workload",
	Short: "Create a new workload",
	Long:  `Create a new workload with the provided configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		payloadPath, err := cmd.Flags().GetString("payload")
		if err != nil {
			fmt.Printf("Error retrieving payload flag: %v\n", err)
			return
		}
    
		contentType, err := cmd.Flags().GetString("contentType")
		if err != nil {
			fmt.Printf("Error retrieving contentType flag: %v\n", err)
			return
		}
    
		payloadData, err := os.ReadFile(payloadPath)
		if err != nil {
			fmt.Printf("Error reading payload file %s: %v\n", payloadPath, err)
			return
		}
    
		resp, err := client.CreateWorkload(payloadData, contentType)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println(resp)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createWorkloadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	createWorkloadCmd.PersistentFlags().String("payload", "", "workload payload path")
	createWorkloadCmd.PersistentFlags().String("contentType", "", "Can be either json or yaml")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// workloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
