package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nravic/cedana-orch/db"
	"github.com/nravic/cedana-orch/server"
	"github.com/nravic/cedana-orch/types"
	"github.com/nravic/cedana-orch/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var jobID string
var debug bool
var restoreFromLatest bool
var path string
var checkpointType string

var communeCmd = &cobra.Command{
	Use:   "commune",
	Short: "Commune with the orchestrator or client by listening or publishing over NATS",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("please specify a subcommand")
	},
}

type Commune struct {
	nc   *nats.Conn
	orch *server.CedanaOrchestrator
}

func NewCommune(jobID string) (*Commune, error) {
	logger := utils.GetLogger()
	cfg, err := utils.InitCedanaConfig()
	if err != nil {
		return nil, err
	}

	nc, err := createNatsConn(&logger, cfg)
	if err != nil {
		return nil, err
	}

	// get client id
	var workerID string
	if !debug {
		db := db.NewDB()
		job := db.GetJob(jobID)

		if job.JobID == "" && !debug {
			return nil, fmt.Errorf("job %s not found", jobID)
		}

		instanceIDs, err := job.GetInstanceIds()
		if err != nil {
			return nil, fmt.Errorf("could not get instances attached to job %s: %v", jobID, err)
		}

		for _, i := range instanceIDs {
			instance := db.GetInstanceByCedanaID(i.InstanceID)
			if instance.Tag == "worker" {
				workerID = instance.CedanaID
			}
		}
	} else {
		workerID = "client123"
	}

	hostname := os.Getenv("HOSTNAME")
	orch := server.NewOrchestrator(
		fmt.Sprintf("cli-commune-%s", hostname),
		jobID,
		workerID,
		nc,
		&logger,
	)

	return &Commune{
		nc:   nc,
		orch: orch,
	}, nil

}

func (c *Commune) cleanup() {
	c.nc.Close()
}

var checkpointCmd = &cobra.Command{
	Use:   "checkpoint",
	Short: "Publishes a checkpoint command to the orchestrator for job with [job-id]",
	RunE: func(cmd *cobra.Command, args []string) error {
		if jobID == "" {
			return fmt.Errorf("job-id is required")
		}

		c, err := NewCommune(jobID)
		if err != nil {
			return err
		}

		c.sendCheckpointCommand(jobID)

		return nil
	},
}

var communeRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restores the latest checkpoint by default for a given job with [job-id] on a fresh instance",
	Long:  "Restore by default applies the latest checkpoint, downloading the latest synced workdir and running the restore command (if specified).",
	RunE: func(cmd *cobra.Command, args []string) error {
		if jobID == "" {
			return fmt.Errorf("job-id is required")
		}
		c, err := NewCommune(jobID)
		if err != nil {
			return err
		}

		err = c.sendRestoreCommand(jobID, true)
		if err != nil {
			return err
		}
		return nil
	},
}

var listCheckpointsCmd = &cobra.Command{
	Use:   "list-checkpoints",
	Short: "Lists all checkpoints for a given job with [job-id]",
	RunE: func(cmd *cobra.Command, args []string) error {
		if jobID == "" {
			return fmt.Errorf("job-id is required")
		}
		c, err := NewCommune(jobID)
		if err != nil {
			return err
		}

		c.prettyPrintCheckpoints(jobID)
		return nil
	},
}

func (c *Commune) sendCheckpointCommand(jobID string) {
	serverCommand := types.ServerCommand{
		Command: "checkpoint",
	}
	publishCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	c.orch.PublishCommand(publishCtx, serverCommand)
	fmt.Printf("Successfully checkpointed job %s at %s\n", jobID, time.Now().Format("2006-01-02 15:04:05"))
}

func (c *Commune) sendRestoreCommand(jobID string, restoreFromLatest bool) error {
	jsc, err := c.nc.JetStream()
	if err != nil {
		return err
	}

	store, err := jsc.ObjectStore(strings.Join([]string{"CEDANA", jobID, "checkpoints"}, "_"))
	if err != nil {
		return err
	}

	files, err := store.List()
	if err != nil {
		return err
	}

	// either restore from latest (by pulling from latest) or take a defined path to restore from
	if !restoreFromLatest {
		var exists bool
		for _, file := range files {
			// list out all files for debugging purposes
			fmt.Println(file.Name)
			if file.Name == path {
				exists = true
			}
		}
		if !exists {
			return fmt.Errorf("checkpoint %s does not exist", path)
		}
	} else {
		var lastModifiedTime time.Time

		// get last modified checkpoint
		for _, file := range files {
			if file.ModTime.After(lastModifiedTime) {
				lastModifiedTime = file.ModTime
				path = file.Name
			}
		}
	}

	if path == "" {
		return fmt.Errorf("checkpoint %s does not exist", path)
	}

	if checkpointType == "" {
		// assume checkpoint type is criu
		checkpointType = string(types.CheckpointTypeCRIU)
	}

	if types.CheckpointType(checkpointType) == "" {
		return fmt.Errorf("could not parse provided checkpoint type %s", checkpointType)
	}

	publishCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	c.orch.PublishCommand(publishCtx, types.ServerCommand{
		Command: "restore",
		CedanaState: types.CedanaState{
			CheckpointPath: path,
			CheckpointType: types.CheckpointType(checkpointType),
		},
	})
	return nil
}

func (c *Commune) prettyPrintCheckpoints(jobID string) error {
	jsc, err := c.nc.JetStream()
	if err != nil {
		return err
	}

	store, err := jsc.ObjectStore(strings.Join([]string{"CEDANA", jobID, "checkpoints"}, "_"))
	if err != nil {
		return err
	}

	files, err := store.List()
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	for _, f := range files {
		table.SetHeader([]string{
			"Bucket",
			"Path",
			"Checkpoint Type",
			"Created At",
			"Size (MB)",
		})

		sizeMb := bytesToMB(f.Size)

		table.Append([]string{
			f.Bucket,
			f.Name,
			"Heartbeat",
			f.ModTime.Format("2006-01-02 15:04:05"),
			strconv.FormatFloat(sizeMb, 'f', 2, 64),
		})
	}

	table.SetRowLine(true)
	table.Render()

	return nil
}

func bytesToMB(bytes uint64) float64 {
	mb := float64(bytes) / (1024 * 1024)
	return mb
}

func createNatsConn(logger *zerolog.Logger, config *utils.CedanaConfig) (*nats.Conn, error) {

	opts := []nats.Option{nats.Name(fmt.Sprintf("Cedana orchestrator %s", "cedana_orchestrator"))}
	opts = setupConnOptions(opts, logger)

	opts = append(opts, nats.Token(config.Connection.AuthToken))

	nc, err := nats.Connect(config.Connection.NATSUrl, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not connect to NATS: %v", err)
	}

	return nc, nil
}

func init() {
	checkpointCmd.Flags().StringVarP(&jobID, "job-id", "j", "", "job id")
	communeRestoreCmd.Flags().StringVarP(&jobID, "job-id", "j", "", "job id")
	checkpointCmd.Flags().BoolVarP(&debug, "debug", "d", false, "debug mode")
	communeRestoreCmd.Flags().BoolVarP(&debug, "debug", "d", false, "debug mode")
	communeRestoreCmd.Flags().BoolVarP(&restoreFromLatest, "latest", "l", false, "restore from latest checkpoint")
	communeRestoreCmd.Flags().StringVarP(&path, "path", "p", "", "path to restore from")
	communeRestoreCmd.Flags().StringVarP(&checkpointType, "type", "t", "", "type of checkpoint")
	communeRestoreCmd.MarkFlagsMutuallyExclusive("latest", "path")
	communeRestoreCmd.MarkFlagsRequiredTogether("type", "path")
	listCheckpointsCmd.Flags().StringVarP(&jobID, "job-id", "j", "", "job id")
	rootCmd.AddCommand(communeCmd)
	communeCmd.AddCommand(checkpointCmd)
	communeCmd.AddCommand(communeRestoreCmd)
	communeCmd.AddCommand(listCheckpointsCmd)
}
