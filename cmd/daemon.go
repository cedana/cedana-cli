package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nravic/cedana-orch/db"
	"github.com/nravic/cedana-orch/server"
	"github.com/nravic/cedana-orch/types"
	"github.com/nravic/cedana-orch/utils"
	"github.com/rs/zerolog"
	gd "github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
)

/*
*
A daemon for monitoring client state to detect resource idling.
Client side daemon only gets run if cedana is running in self-serve mode.
The daemon does polling and interacts with the db, in a fascimilie of the larger market.
*/

type CLIDaemon struct {
	logger *zerolog.Logger
	db     *db.DB
	cfg    *utils.CedanaConfig

	nc *nats.Conn
	js jetstream.JetStream

	runner *Runner // daemon has it's own runner to spin up instances on failure
}

var (
	orchestratorID string
	clientID       string
	// jobID defined elsewhere
)

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Local daemon for self-serve cedana",
	Hidden: true, // run only when using self-serve
	RunE: func(cmd *cobra.Command, args []string) error {
		cd := NewCLIDaemon()

		// if o,c,j flags weren't passed, check env
		if orchestratorID == "" {
			oid, exists := os.LookupEnv("CEDANA_ORCH_ID")
			if !exists {
				return fmt.Errorf("CEDANA_ORCH_ID not set")
			}
			orchestratorID = oid
		}

		if jobID == "" {
			jid, exists := os.LookupEnv("CEDANA_JOB_ID")
			if !exists {
				return fmt.Errorf("CEDANA_JOB_ID not set")
			}
			jobID = jid
		}

		if clientID == "" {
			cid, exists := os.LookupEnv("CEDANA_CLIENT_ID")
			if !exists {
				return fmt.Errorf("CEDANA_CLIENT_ID not set")
			}
			clientID = cid
		}

		cd.Start(orchestratorID, jobID, clientID)
		return nil
	},
}

func NewCLIDaemon() *CLIDaemon {
	logger := utils.GetLogger()
	config, err := utils.InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not set up spot config")
	}

	runner := buildRunner()

	return &CLIDaemon{
		logger: &logger,
		db:     db.NewDB(),
		cfg:    config,
		runner: runner,
	}
}

func (cd *CLIDaemon) Start(orchestratorID string, jobID string, clientID string) {
	opts := []nats.Option{nats.Name(fmt.Sprintf("Cedana orchestrator %s", "cedana_orchestrator"))}
	opts = setupConnOptions(opts, cd.logger)

	opts = append(opts, nats.Token(cd.cfg.Connection.AuthToken))

	// TODO NR: retrier
	nc, err := nats.Connect(cd.cfg.Connection.NATSUrl, opts...)
	if err != nil {
		cd.logger.Fatal().Err(err).Msg("could not connect to NATS")
	}

	// create new jetstream manager
	js, err := jetstream.New(nc)
	if err != nil {
		cd.logger.Fatal().Err(err).Msg("Could not create JetStream interface")
	}

	cd.nc = nc
	cd.js = js

	cedanaDir := strings.Join([]string{os.Getenv("HOME"), ".cedana"}, "/")

	ctx := &gd.Context{
		PidFileName: strings.Join(
			[]string{
				cedanaDir,
				"cedana-orchestrate." + jobID + ".pid",
			},
			"/"),
		PidFilePerm: 0644,
		// if we've come this far, safe to assume .cedana exists in $HOME
		LogFileName: strings.Join(
			[]string{
				cedanaDir,
				"cedana-orchestrate." + jobID + ".log",
			},
			"/"),
		LogFilePerm: 0755,
		WorkDir:     "./",
		// always pass ids as flags to avoid checking env on local machine
		Args: []string{
			"cedana-orch", "daemon",
			"-o", orchestratorID,
			"-j", jobID,
			"-c", clientID,
		},
	}

	d, err := ctx.Reborn()
	if err != nil {
		cd.logger.Err(err).Msg("could not start daemon")
	}

	if d != nil {
		return
	}

	defer ctx.Release()

	cd.logger.Info().Msgf("daemon started at %s", time.Now().Local())

	co := server.NewOrchestrator(
		orchestratorID,
		jobID,
		clientID,
		cd.nc,
		cd.logger,
	)
	go cd.PollProvider(clientID, jobID)
	go cd.UpdateJobStatus(jobID)
	co.Start()

	cd.logger.Info().Msgf("daemon stopped at %s", time.Now().Local())
}

// PollingDaemon checks for provider events and sends a checkpoint status over NATS. Needs to be async (so can't use the command channel)
// because it might have to be run on the user's machine.
func (cd *CLIDaemon) PollProvider(clientID, jobID string) {
	// get instance from clientID
	i := cd.db.GetInstanceByCedanaID(clientID)
	ticker := time.NewTicker(1 * time.Minute)
	provider := cd.runner.providers[i.Provider]

	for {
		select {
		case <-ticker.C:
			event, err := provider.GetInstanceStatus(i)
			cd.logger.Info().Msgf("got event %v", event)
			if err != nil {
				cd.logger.Info().Msgf("could not poll provider with error %v", err)
			}
			if event != nil {
				if event.MarkedForTermination {
					cd.logger.Info().Msgf("provider marked client %s for termination", clientID)

					cd.sendStatus(types.MetaState{
						Event:            *event,
						CheckpointReason: types.CheckpointReasonInstanceTermination,
					}, clientID, jobID)

					// this should probably live in a separate function or something
					// for a self-serve version this is enough I think
					if cd.cfg.KeepRunning {
						err := cd.runner.restoreJob(jobID)
						if err != nil {
							cd.logger.Error().Err(err).Msg("could not restore job onto a new instance")
						}
					}
				}
			}

		default:
			time.Sleep(time.Second)
			// do nothing
		}
	}
}

func (cd *CLIDaemon) PollProviderOnce(clientID, jobID string) error {
	i := cd.db.GetInstanceByCedanaID(clientID)
	provider := cd.runner.providers[i.Provider]
	event, err := provider.GetInstanceStatus(i)
	if err != nil {
		return err
	}
	if event.MarkedForTermination {
		cd.sendStatus(types.MetaState{
			Event:            *event,
			CheckpointReason: types.CheckpointReasonInstanceTermination,
		}, clientID, jobID)
	}
	return nil
}

// UpdateJobStatus receives on NATS messages from the orchestrator about
// successful checkpoints/restores and on the state of the running task.
func (cd *CLIDaemon) UpdateJobStatus(jobID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	cons, err := cd.js.AddConsumer(ctx, "CEDANA", jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckNonePolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		FilterSubject: strings.Join([]string{"CEDANA", jobID, "state"}, "."),
	})

	if err != nil {
		return err
	}

	iter, err := cons.Messages()
	if err != nil {
		return err
	}

	for {
		msg, err := iter.Next()
		if err != nil {
			cd.logger.Debug().Msgf("could not get message with error %v", err)
			time.Sleep(time.Second)
			continue // continue to next iter (and block until we get a message)
		}

		// inefficient (TODO NR) we're just passing along cedanaState
		data := msg.Data()
		var cedanaState types.CedanaState
		err = json.Unmarshal(data, &cedanaState)
		if err != nil {
			// skip error
			cd.logger.Debug().Msgf("could not unmarshal message with error %v", err)
			continue
		}

		cd.logger.Info().Msgf("got job status: %v", cedanaState)

		cd.logger.Info().Msgf("updating status in db for job %s", jobID)
		// get latest job
		j := cd.db.GetJob(jobID)

		if cedanaState.CheckpointState == types.CheckpointSuccess {
			j.Checkpointed = true
			j.LastCheckpointedAt = time.Now()
			j.Bucket = cedanaState.CheckpointPath
		}

		cd.db.UpdateJob(j)
		time.Sleep(time.Second)
	}
}

func (cd *CLIDaemon) sendStatus(state types.MetaState, clientID string, jobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	stateMarshalled, err := json.Marshal(state)
	if err != nil {
		cd.logger.Fatal().Err(err).Msg("could not marshal command")
	}

	ackF, err := cd.js.PublishAsync(ctx, strings.Join([]string{"CEDANA", jobID, clientID, "meta"}, "."), stateMarshalled)
	if err != nil {
		cd.logger.Info().Msgf("could not publish command with error %v", err)
	}

	// block and wait for ack
	select {
	case <-ackF.Ok():
		cd.logger.Info().Msgf("ack received for command: %v", string(stateMarshalled))
	case err := <-ackF.Err():
		cd.logger.Info().Msgf("error received: %v for command: %v", err, string(stateMarshalled))
	}

}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.Flags().StringVarP(&orchestratorID, "orchestrator-id", "o", "", "orchestrator id")
	daemonCmd.Flags().StringVarP(&clientID, "client-id", "c", "", "client id")
	daemonCmd.Flags().StringVarP(&jobID, "job-id", "j", "", "job id")
}
