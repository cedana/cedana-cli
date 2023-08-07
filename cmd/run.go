package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cedana/cedana-cli/db"
	"github.com/cedana/cedana-cli/market"
	"github.com/cedana/cedana-cli/types"
	"github.com/cedana/cedana-cli/utils"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"

	cedana "github.com/cedana/cedana-cli/types"
)

type Runner struct {
	ctx       context.Context
	cfg       *utils.CedanaConfig
	logger    *zerolog.Logger
	providers map[string]cedana.Provider
	jobFile   *cedana.JobFile
	job       *cedana.Job
	db        *db.DB
	nc        *nats.Conn
}

var showOnlyRunning bool

func buildRunner() *Runner {
	logger := utils.GetLogger()

	config, err := utils.InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not set up spot config")
	}

	r := &Runner{
		ctx:       context.Background(),
		cfg:       config,
		logger:    &logger,
		providers: make(map[string]cedana.Provider),
		db:        db.NewDB(),
	}

	// create nats connections.
	// placing this here acts almost as a proxy for an authentication server
	// TODO NR: weak though!!
	opts := []nats.Option{nats.Name("Cedana CLI")}
	opts = append(opts, nats.Token(r.cfg.Connection.AuthToken))

	nc, err := nats.Connect(r.cfg.Connection.NATSUrl, opts...)
	if err != nil {
		r.logger.Fatal().Err(err).Msg("Could not connect to NATS")
	}

	r.nc = nc
	r.buildProviders()

	return r
}

func (r *Runner) cleanRunner() {
	r.nc.Close()
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run your workload on the most optimal instance, anywhere",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r := buildRunner()
		defer r.cleanRunner()

		jobFile, err := cedana.InitJobFile(args[0])
		if err != nil {
			r.logger.Fatal().Err(err).Msg("could not set up cedana job")
		}
		r.jobFile = jobFile

		r.job = r.db.CreateJob(r.jobFile)

		// TODO NR - expand later to bring in managed service
		if r.cfg.SelfServe {
			err = r.runJobSelfServe()
			if err != nil {
				return err
			}
			return nil
		}

		return nil
	},
}

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Retry a failed setup from jobID [job-id]",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r := buildRunner()
		defer r.cleanRunner()

		jobID := args[0]

		// check db for existing job
		job := r.db.GetJob(jobID)
		if job == nil {
			return fmt.Errorf("could not find job with id %s", jobID)
		}

		r.job = job
		// pull worker ID out

		attachedInstanceIDs, err := job.GetInstanceIds()
		if err != nil {
			return err
		}

		var worker cedana.Instance

		// this should be way better. ideally the spun up instances themselves
		// should have
		for _, i := range attachedInstanceIDs {
			instance := r.db.GetInstanceByCedanaID(i.InstanceID)
			if instance.Tag == "worker" {
				worker = instance
			}
		}

		err = r.retryJob(worker)
		if err != nil {
			return err
		}

		return nil
	},
}

var showInstancesCmd = &cobra.Command{
	Use:   "show",
	Short: "Show instances launched with Cedana",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := buildRunner()
		defer r.cleanRunner()

		// update state, by calling the correct DescribeInstances function for each set of instances
		// we don't want to call update functions individually, would ideally do it in batch (like w/ AWS)
		for provider := range r.providers {
			if provider == "aws" {
				aws := r.providers["aws"]
				awsInstances := r.db.GetInstancesByProvider("aws")
				// TODO: NR - slice of pointers, ugh
				instances := make([]*cedana.Instance, len(awsInstances))
				for i, v := range awsInstances {
					instances[i] = &v
				}
				// this function updates in place
				aws.DescribeInstance(instances, "")
			}
			if provider == "paperspace" {
				paperspace := r.providers["paperspace"]
				paperspaceInstances := r.db.GetInstancesByProvider("paperspace")
				instances := make([]*cedana.Instance, len(paperspaceInstances))
				for i, v := range paperspaceInstances {
					instances[i] = &v
				}

				paperspace.DescribeInstance(instances, "")

			}
		}

		// updated in prior step
		var instances []cedana.Instance
		if showOnlyRunning {
			instances = r.db.GetAllRunningInstances()
		} else {
			instances = r.db.GetAllInstances()
		}
		if len(instances) > 0 {
			prettyPrintInstances(instances)
		} else {
			r.logger.Warn().Msg("could not find any instances in account.")
		}
		return nil
	},
}

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy instance or cluster of instances launched with Cedana",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r := buildRunner()
		defer r.cleanRunner()
		// instance destruction is handled by the provider (both db and at provider levels)
		// TODO NR: assuming for now user just wants to destroy one instance, have to expand
		id := args[0]

		instance := r.db.GetInstanceByCedanaID(id)
		if instance.ID == 0 {
			return fmt.Errorf("could not find instance with id %s", id)
		}

		switch instance.Provider {
		case "aws":
			aws := r.providers["aws"]
			err := aws.DestroyInstance(instance)
			if err != nil {
				return err
			}
		case "paperspace":
			paperspace := r.providers["paperspace"]
			err := paperspace.DestroyInstance(instance)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

var destroyAllCmd = &cobra.Command{
	Use:   "destroy-all",
	Short: "Destroy all running instances launched with Cedana",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := buildRunner()
		defer r.cleanRunner()

		runningInstances := r.db.GetAllRunningInstances()

		r.logger.Info().Msgf("destroying %d instances...", len(runningInstances))
		for _, instance := range runningInstances {
			provider := r.providers[instance.Provider]
			err := provider.DestroyInstance(instance)
			if err != nil {
				return err
			}
		}
		r.logger.Info().Msg("done!")

		r.logger.Info().Msgf("purging jobs...")
		err := db.NewDB().PurgeJobs()
		if err != nil {
			return err
		}
		r.logger.Info().Msg("done!")

		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Manually restore a previously checkpoint onto a new instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r := buildRunner()
		defer r.cleanRunner()

		jobID := args[0]

		err := r.restoreJob(jobID)
		if err != nil {
			return err
		}

		r.logger.Info().Msgf("restored job %s onto new instance at time %s!", jobID, time.Now().Format(time.RFC3339))

		return nil
	},
}

// very basic retry of a failed setup, allows users to play with the yaml without
// needing to tear down and redeploy
func (r *Runner) retryJob(worker cedana.Instance) error {
	is := BuildInstanceSetup(worker, *r.job)

	err := is.ClientSetup(true)

	if err != nil {
		r.logger.Info().Msgf("could not set up client, retry using `./cedana-cli setup -i %s -j %s`", worker.CedanaID, "yourjob.yml")
		r.db.UpdateJobState(r.job, types.JobStateSetupFailed)
		return err
	}

	r.db.UpdateJobState(r.job, types.JobStateRunning)

	// job is orchestrated by a local worker
	orch, err := r.db.CreateInstance(&cedana.Instance{
		Provider:    "local",
		IPAddress:   "0.0.0.0",
		Tag:         "orchestrator",
		State:       "running",
		AllocatedID: "local",
	})

	r.db.AttachInstanceToJob(r.job, *orch)

	if err != nil {
		return err
	}

	cd := NewCLIDaemon()
	cd.Start(orch.CedanaID, r.job.JobID, worker.CedanaID)

	return nil

}

// restoreJob manually restores the most recent checkpoint onto a new instance
func (r *Runner) restoreJob(jobID string) error {
	// validate that job exists
	job := r.db.GetJob(jobID)
	if job == nil {
		r.logger.Fatal().Err(fmt.Errorf("could not find job with id %s", jobID))
	}
	// validate that attached instance is destroyed
	attachedInstanceIDs, err := job.GetInstanceIds()

	if err != nil || len(attachedInstanceIDs) == 0 {
		r.logger.Fatal().Msgf("Could not get attached instances for job: %s", job.JobID)
	}

	var oldWorker cedana.Instance
	for _, i := range attachedInstanceIDs {
		instance := r.db.GetInstanceByCedanaID(i.InstanceID)
		if instance.Tag == "worker" {
			oldWorker = instance
		}
		if oldWorker.State != "destroyed" {
			r.logger.Fatal().Msgf("Instance %s is not destroyed", i.InstanceID)
		}
	}
	r.logger.Debug().Msgf("old worker: %+v", oldWorker)

	// spin up new instance and attach to job
	r.job = job
	// TODO NR - this needs to be somewhere else!
	jobFile, err := cedana.InitJobFile(r.job.JobFilePath)
	if err != nil {
		r.logger.Fatal().Err(err).Msg("could not set up cedana job")
	}
	r.jobFile = jobFile

	candidates := market.Optimize(r.jobFile)
	worker, err := r.deployWorker(candidates, false)
	if err != nil {
		r.logger.Fatal().Err(err).Msg("could not deploy worker")
		return err
	}

	w, err := NewWhisperer(job.JobID)
	if err != nil {
		return err
	}
	defer w.cleanup()
	w.orch.AttachNewWorker(worker.CedanaID)
	w.sendRestoreCommand(r.job.JobID, true)

	return nil

}

/*
Runs a job for the self serve model of Cedana.
We don't deploy an orchestrator to the cloud (instead we run it locally in a daemon) and pass through our NATS.
*/
func (r *Runner) runJobSelfServe() error {
	candidates := market.Optimize(r.jobFile)
	err := r.SetupNATSForJob()
	if err != nil {
		r.logger.Fatal().Err(err).Msg("could not set up inter-cloud broker architecture")
		return err
	}

	r.logger.Info().Msg("setting up job...")
	r.db.UpdateJobState(r.job, types.JobStatePending)

	worker, err := r.deployWorker(candidates, true)
	if err != nil {
		r.logger.Fatal().Err(err).Msg("could not deploy worker")
		return err
	}

	// job is orchestrated by a local worker
	orch, err := r.db.CreateInstance(&cedana.Instance{
		Provider:    "local",
		IPAddress:   "0.0.0.0",
		Tag:         "orchestrator",
		State:       "running",
		AllocatedID: "local",
	})

	r.db.AttachInstanceToJob(r.job, *orch)

	if err != nil {
		return err
	}

	// daemon has a separate instance of the runner - should we be passing
	// self to it? TODO NR, probably some perf gains here
	cd := NewCLIDaemon()
	cd.Start(orch.CedanaID, r.job.JobID, worker.CedanaID)

	return nil
}

func (r *Runner) deployWorker(candidates []cedana.Instance, runTask bool) (*cedana.Instance, error) {
	// a for loop here that breaks when we have an instance
	// as the list is sorted + filtered, most optimal is the first
	var optimalInstance *cedana.Instance
	var unavailableRegions []string
	for _, candidate := range candidates {
		// if region is in unavailableRegions, skip
		if slices.Contains(unavailableRegions, candidate.Region) {
			continue
		}
		provider := r.providers[candidate.Provider]
		i, err := provider.CreateInstance(&candidate)

		if err == nil {
			// we have an instance, break out of this loop
			optimalInstance = i
			break
		} else {
			r.logger.Warn().Msgf("capacity error returned from provider: %v!", err)
			// if we have a capacity related error - return and keep trying
			if ce, ok := err.(*cedana.CapacityError); ok {
				if ce.Code == "capacity" {
					if ce.Region == candidate.Region {
						// region is now on our shitlist, try next instance
						r.logger.Info().Msgf("capacity error in region %s, trying other regions...", candidate.Region)
						unavailableRegions = append(unavailableRegions, candidate.Region)
						continue
					} else {
						r.logger.Warn().Msg("capacity error during instance creation - trying the next optimal instance")
						continue
					}
				}
			} else {
				// other error - break
				return nil, err
			}
		}
	}
	if optimalInstance == nil {
		return nil, errors.New("something went wrong during instance creation - nil instance returned from provider")
	}

	optimalInstance.Tag = "worker"

	r.logger.Info().Msg("waiting for instance to be ready...")
	for {
		if optimalInstance.State == "running" || optimalInstance.State == "ready" {
			break
		}
		switch p := optimalInstance.Provider; p {
		case "aws":
			aws := r.providers["aws"]
			err := aws.DescribeInstance([]*cedana.Instance{optimalInstance}, "")
			if err != nil {
				// do nothing - describe could fail for any number of stupid reasons
				continue
			}
			time.Sleep(5 * time.Second)
		case "paperspace":
			paperspace := r.providers["paperspace"]
			err := paperspace.DescribeInstance([]*cedana.Instance{optimalInstance}, "")
			if err != nil {
				continue
			}
			time.Sleep(5 * time.Second)
		}
	}

	r.db.AttachInstanceToJob(r.job, *optimalInstance)

	r.logger.Info().Msg("running setup scripts...")

	is := BuildInstanceSetup(*optimalInstance, *r.job)

	err := is.ClientSetup(runTask)

	if err != nil {
		r.logger.Info().Msgf("could not set up client, retry using `./cedana-cli setup -i %s -j %s`", optimalInstance.AllocatedID, "yourjob.yml")
		r.db.UpdateJobState(r.job, types.JobStateSetupFailed)
		return nil, err
	}

	r.db.UpdateJobState(r.job, types.JobStateRunning)
	return optimalInstance, nil
}

func (r *Runner) SetupNATSForJob() error {
	err := r.CreateNATSStream()
	if err != nil {
		r.logger.Fatal().Err(err).Msg("Could not create NATS stream")
		return err
	}

	err = r.CreateObjectStores()
	if err != nil {
		r.logger.Fatal().Err(err).Msg("Could not create object stores")
		return err
	}

	err = r.PublishJob()
	if err != nil {
		r.logger.Fatal().Err(err).Msg("Could not publish initial job state")
		return err
	}

	return nil
}

func (r *Runner) CreateNATSStream() error {
	js, err := jetstream.New(r.nc)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "CEDANA",
		Subjects: []string{"CEDANA.>"},
	})

	if err != nil {
		if strings.Contains(err.Error(), "stream already exists") {
			// stream already exists. We're possibly retrying an extant job in this case - drop error
			return nil
		}
		return err
	}
	return nil
}

// creates object stores for checkpoints & for workdirs
func (r *Runner) CreateObjectStores() error {
	js, err := r.nc.JetStream()
	if err != nil {
		return err
	}

	// create checkpoint bucket (TODO NR: should this be elsewhere?)
	_, err = js.CreateObjectStore(&nats.ObjectStoreConfig{
		Bucket: strings.Join([]string{"CEDANA", r.job.JobID, "checkpoints"}, "_"),
	})

	if err != nil {
		// if the bucket already exists, just drop the error
		if strings.Contains(err.Error(), "exists") {
			r.logger.Info().Msg("checkpoint bucket already exists, skipping creation...")
			return nil
		} else {
			r.logger.Fatal().Err(err).Msg("Could not create checkpoint bucket")
			return err
		}
	}

	return nil
}

func (r *Runner) PublishJob() error {
	// serialize and publish job to NATS for initial ingestion by server
	js, err := jetstream.New(r.nc)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	marshaledJob, err := json.Marshal(r.job)
	if err != nil {
		return err
	}
	_, err = js.Publish(ctx, fmt.Sprintf("CEDANA.%s", r.job.JobID), marshaledJob)
	if err != nil {
		return err
	}

	return nil
}

func (r *Runner) buildProviders() {
	// get list of enabled enabledProviders from config
	enabledProviders := r.cfg.EnabledProviders
	for _, p := range enabledProviders {
		if p == "aws" {
			spot := buildAWSProvider()
			r.providers["aws"] = spot
		}
		if p == "paperspace" {
			paperspace := buildPaperspaceProvider()
			r.providers["paperspace"] = paperspace
		}
		// TODO: add others
		r.providers["local"] = buildLocalProvider()
	}
}

func buildAWSProvider() *market.Spot {
	return market.GenSpotClient()
}

func buildPaperspaceProvider() *market.Paperspace {
	return market.GenPaperspaceClient()
}

func buildLocalProvider() *market.LocalProvider {
	return market.GenLocalClient()
}

func (r *Runner) prettyPrintJobs(jobs []cedana.Job) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Job", "Created At", "Running For", "Attached Instances", "Orchestrated By", "State", "Checkpointed", "Last Checkpointed At"})
	for _, j := range jobs {
		var attachedInstanceIDs []cedana.SerializedInstance
		var IDs []string
		var orchestrator string

		attachedInstanceIDs, err := j.GetInstanceIds()

		if err != nil || len(attachedInstanceIDs) == 0 {
			r.logger.Info().Msgf("Could not get attached instances for job: %s", j.JobID)
			continue
		}

		for _, i := range attachedInstanceIDs {
			instance := r.db.GetInstanceByCedanaID(i.InstanceID)
			if instance.Tag == "orchestrator" {
				orchestrator = i.InstanceID
			} else {
				IDs = append(IDs, i.InstanceID)
			}
		}

		var lastCheckpointedAt string
		duration := time.Duration(time.Since(j.LastCheckpointedAt))
		// HACK! TODO NR: figure out the max duration stuff
		if duration >= 922337203685 {
			lastCheckpointedAt = "Never"
		} else {
			lastCheckpointedAt = duration.String()
		}

		table.Append([]string{
			j.JobID,
			j.JobFilePath,
			j.CreatedAt.Format("2006-01-02 15:04:05"),
			time.Since(j.CreatedAt).Abs().String(),
			strings.Join(IDs, ", "),
			orchestrator,
			string(j.State),
			strconv.FormatBool(j.Checkpointed),
			lastCheckpointedAt,
		})
	}
	table.SetRowLine(true)

	table.Render()
}

func prettyPrintInstances(instances []cedana.Instance) {
	table := tablewriter.NewWriter(os.Stdout)
	for _, it := range instances {
		var manufacturer string
		var gpuName string
		gpuinfo := it.GetGPUs()
		if gpuinfo.Gpus != nil {
			gpuName = gpuinfo.Gpus[0].Name
			manufacturer = gpuinfo.Gpus[0].Manufacturer
		}
		// TODO: assumption here is that GPUs are all the same (if there are multiple attached to the machine)
		table.SetHeader([]string{
			"Cedana ID",
			"Provider",
			"VCPUs",
			"RAM (GB)",
			"GPU",
			"Total VRAM (GB)",
			"Instance Type",
			"Region",
			"Price ($/hr)",
			"Tag",
			"Created At",
			"Status",
		})
		table.Append([]string{
			fmt.Sprint(it.CedanaID),
			it.Provider,
			strconv.FormatFloat(it.VCPUs, 'f', -1, 64),
			strconv.FormatFloat(it.MemoryGiB, 'f', -1, 64),
			fmt.Sprintf("%s %s", manufacturer, gpuName),
			strconv.Itoa(gpuinfo.TotalGpuMemoryInMiB / 1000),
			it.InstanceType,
			it.Region,
			strconv.FormatFloat(it.Price, 'f', 5, 64),
			it.Tag,
			it.CreatedAt.Format("2006-01-02 15:04:05"),
			it.State,
		})
	}

	table.Render()

}

func init() {
	showInstancesCmd.Flags().BoolVarP(&showOnlyRunning, "running", "r", false, "Show only running instances")
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(showInstancesCmd)
	rootCmd.AddCommand(destroyAllCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(retryCmd)
}
