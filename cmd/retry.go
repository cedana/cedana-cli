package cmd

import (
	"github.com/cedana/cedana-cli/db"
	"github.com/cedana/cedana-cli/utils"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Retry setup for a failed job",
	Long:  "Checks job status and retries if necessary",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db := db.NewDB()
		l := utils.GetLogger()

		jobID := args[0]
		job := db.GetJob(jobID)


		return nil
	},
}
