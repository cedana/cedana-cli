package self_serve

import (
	"github.com/cedana/cedana-cli/db"
	"github.com/spf13/cobra"
)

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Commands for job management",
}

var purgeJobsCmd = &cobra.Command{
	Use:   "purge",
	Short: "Deletes non-active jobs from the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		db := db.NewDB()
		return db.PurgeJobs()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of a Cedana job",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := buildRunner()
		jobs := r.db.GetAllJobs()

		// TODO NR: job status here should query NATS to get most current state of the job (and/or infer it)
		r.prettyPrintJobs(jobs)
		return nil
	},
}

// view into checkpoints taken for a job
var checkpointsCmd = &cobra.Command{
	Use:   "checkpoints",
	Short: "show checkpoints taken for a Cedana job",
	RunE: func(cmd *cobra.Command, args []string) error {
		// get files from object store

		// pretty print checkpoints
		return nil
	},
}

func prettyPrintCheckpoints() {
}

func init() {
	runSelfServeCmd.AddCommand(jobCmd)
	jobCmd.AddCommand(purgeJobsCmd)
	jobCmd.AddCommand(statusCmd)
}
