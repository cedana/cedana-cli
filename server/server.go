package server

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/cedana/cedana-cli/types"
	"github.com/cedana/cedana-cli/utils"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
)

type CedanaOrchestrator struct {
	logger *zerolog.Logger
	config *utils.CedanaConfig
	nc     *nats.Conn
	js     jetstream.JetStream   // jetstream interface/manager
	jsc    nats.JetStreamContext // jetstream context (for object store)
	// server should be instantiated w/ the job, so all this information is already present

	id  string
	jid string // job id
	wid string // worker id

	// used to coordinate checkpoints across multiple goroutines
	CmdChannel chan types.ServerCommand
}

func (co *CedanaOrchestrator) AttachNewWorker(id string) {
	// just replaces existing worker with new one - TODO NR - multinode orchestrators
	co.wid = id
}

func (co *CedanaOrchestrator) GenClientStateIterator(ctx context.Context) (jetstream.MessagesContext, error) {
	co.logger.Info().Msgf("consuming messages on subject CEDANA.%s.%s.state", co.jid, co.wid)
	// create a consumer of client state
	cons, err := co.js.AddConsumer(ctx, "CEDANA", jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckNonePolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		FilterSubject: strings.Join([]string{"CEDANA", co.jid, co.wid, "state"}, "."),
	})

	if err != nil {
		co.logger.Info().Err(err).Msg("could not subscribe to NATS client state")
		return nil, err
	}

	iter, _ := cons.Messages()
	return iter, nil
}

// MetaState refers to provider state - instance revocations, hardware failures or provider shutdowns
// are broadcast on this iterator.
func (co *CedanaOrchestrator) GenMetaStateIterator(ctx context.Context) (jetstream.MessagesContext, error) {
	co.logger.Info().Msgf("consuming messages on subject CEDANA.%s.%s.meta", co.jid, co.wid)
	// create a consumer of meta state
	cons, err := co.js.AddConsumer(ctx, "CEDANA", jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckNonePolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		FilterSubject: strings.Join([]string{"CEDANA", co.jid, co.wid, "meta"}, "."),
	})

	if err != nil {
		co.logger.Info().Err(err).Msg("could not subscribe to NATS meta state")
		return nil, err
	}

	iter, _ := cons.Messages()
	return iter, nil
}

func (co *CedanaOrchestrator) PublishCommand(ctx context.Context, command types.ServerCommand) {
	cmd, err := json.Marshal(command)
	if err != nil {
		co.logger.Fatal().Err(err).Msg("could not marshal command")
	}

	ackF, err := co.js.PublishAsync(ctx, strings.Join([]string{"CEDANA", co.jid, co.wid, "commands"}, "."), cmd)
	if err != nil {
		co.logger.Info().Msgf("could not publish command with error %v", err)
	}

	// block and wait for ack
	select {
	case <-ackF.Ok():
		co.logger.Info().Msgf("ack received for command: %v", string(cmd))
	case err := <-ackF.Err():
		co.logger.Info().Msgf("error received: %v for command: %v", err, string(cmd))
	}
	// watch for object store changes!
}

// pulls latest applicable checkpoint name from NATS storage
func (co *CedanaOrchestrator) getLatestCheckpoint() (*string, error) {
	var checkpointPath string
	var lastModifiedTime time.Time

	store, err := co.jsc.ObjectStore(strings.Join([]string{"CEDANA", co.jid, "checkpoints"}, "_"))
	if err != nil {
		return nil, err
	}

	files, err := store.List()
	if err != nil {
		return nil, err
	}

	// get last modified checkpoint
	for _, file := range files {
		if file.ModTime.After(lastModifiedTime) {
			lastModifiedTime = file.ModTime
			checkpointPath = file.Name
		}
	}

	return &checkpointPath, nil
}

// run continuously, as a gofunction
func (co *CedanaOrchestrator) ProcessClientState(stateIter jetstream.MessagesContext) {
	for {
		var state *types.CedanaState
		var stateBufferSize int = 10
		stateBuffer := make([]*types.CedanaState, 0, stateBufferSize)

		for i := 0; i < stateBufferSize; i++ {
			// We will always wait here. The speed at which we process messages is limited by speed at which
			// clients send.
			msg, err := stateIter.Next()
			if err != nil {
				// drop error, not an issue
				co.logger.Debug().Msgf("could not get message: %v", err)
				time.Sleep(time.Second)
				continue // continue to next iter (and block until we get a message)
			}

			data := msg.Data()
			err = json.Unmarshal(data, &state)
			if err != nil {
				co.logger.Info().Msgf("could not unmarshal state: %v", err)
			}

			co.logger.Info().Msgf("got state: %v", state)

			if state != nil {
				stateBuffer = append(stateBuffer, state)
				if state.CheckpointState != "" {
					err := co.updateJobState(context.Background(), state)
					if err != nil {
						co.logger.Info().Msgf("could not update job state: %v", err)
					}
				}
			}
		}
		// buffer is full, send it and wait for processing
		co.isInstanceIdle(stateBuffer)
		time.Sleep(time.Second)
	}
}

func (co *CedanaOrchestrator) ProcessMetaState(stateIter jetstream.MessagesContext) {
	for {
		var state *types.MetaState
		var stateBufferSize int = 10
		stateBuffer := make([]*types.MetaState, 0, stateBufferSize)

		for i := 0; i < stateBufferSize; i++ {
			// We will always wait here. The speed at which we process messages is limited by speed at which
			// clients send.
			msg, err := stateIter.Next()
			if err != nil {
				// drop error, not an issue
				co.logger.Debug().Msgf("could not get message: %v", err)
				time.Sleep(time.Second)
				continue // continue to next iter (and block until we get a message)
			}
			data := msg.Data()

			co.logger.Info().Msgf("got meta state: %v", state)

			err = json.Unmarshal(data, &state)
			if err != nil {
				co.logger.Info().Msgf("could not unmarshal state: %v", err)
			}

			if state != nil {
				stateBuffer = append(stateBuffer, state)
			}
		}
		// very simply, process the latest state and check for instance revocation
		if len(stateBuffer) > 0 {
			lastState := stateBuffer[len(stateBuffer)-1]
			if lastState.Event.MarkedForTermination {
				co.logger.Info().Msgf("instance %s marked for termination ... sending checkpoint", co.wid)
				// client logics checkpointType
				co.CmdChannel <- types.ServerCommand{
					Command: "checkpoint",
				}
			}
		}
	}
}

func (co *CedanaOrchestrator) Start() error {
	defer co.nc.Drain()

	var heartbeatInterval int = 5
	if co.config.Checkpoint.HeartbeatInterval != 0 {
		heartbeatInterval = co.config.Checkpoint.HeartbeatInterval
	}
	heartbeatTicker := time.NewTicker(time.Second * time.Duration(heartbeatInterval))
	defer heartbeatTicker.Stop()

	if co.config.Checkpoint.HeartbeatEnabled {
		co.logger.Info().Msgf("heartbeat enabled with duration %v", heartbeatInterval)
		go co.HeartbeatCheckpoint(heartbeatTicker)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	co.logger.Info().Msg("starting server...")

	clientIter, err := co.GenClientStateIterator(ctx)
	if err != nil {
		return err
	}

	metaIter, err := co.GenMetaStateIterator(ctx)
	if err != nil {
		return err
	}

	go co.ProcessClientState(clientIter)
	go co.ProcessMetaState(metaIter)

	for {
		select {
		case cmd := <-co.CmdChannel:
			co.logger.Info().Msgf("publishing command: %v", cmd)
			co.PublishCommand(context.Background(), cmd)
		}
	}
}

func (co *CedanaOrchestrator) HeartbeatCheckpoint(heartbeatTicker *time.Ticker) {
	for {
		select {
		// enters this and blocks until we get a message
		case <-heartbeatTicker.C:
			co.logger.Info().Msgf("sending heartbeat to client %s...", co.wid)
			co.CmdChannel <- types.ServerCommand{
				Command:   "checkpoint",
				Heartbeat: true, // for cedana client to pre-dump? TODO NR
			}
		}
	}
}

// processes client state in batches to detect if an instance is idle
// idling triggers a checkpoint + destruction of instance - this is experimental!
func (co *CedanaOrchestrator) isInstanceIdle(stateBuffer []*types.CedanaState) {
	// fed a buffer of states
	idle := false
	if idle {
		co.logger.Info().Msgf("instance %s identified as idle... sending checkpoint", co.wid)
		// client logics checkpointType
		co.CmdChannel <- types.ServerCommand{
			Command: "checkpoint",
		}
	}
}

// TODO NR: Needs work
func (co *CedanaOrchestrator) updateJobState(ctx context.Context, state *types.CedanaState) error {
	data, err := json.Marshal(*state)
	if err != nil {
		co.logger.Info().Msgf("could not marshal state: %v", err)
	}
	_, err = co.js.Publish(ctx, strings.Join([]string{"CEDANA", co.jid, "state"}, "."), data)
	if err != nil {
		co.logger.Info().Msgf("could not push new job state: %v with error: %v", state, err)
		return err
	}

	return nil
}

func NewOrchestrator(
	orchestratorId string,
	jobId string,
	clientId string,
	nc *nats.Conn,
	logger *zerolog.Logger,
) *CedanaOrchestrator {
	config, err := utils.InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msgf("Could not initialize logger!")
	}

	// create new jetstream manager
	js, err := jetstream.New(nc)
	if err != nil {
		logger.Fatal().Err(err).Msg("Could not create JetStream interface")
	}

	// context for object store
	jsc, err := nc.JetStream()
	if err != nil {
		logger.Fatal().Err(err).Msg("Could not create JetStream context")
	}

	// command channel
	cmdChan := make(chan types.ServerCommand)

	s := &CedanaOrchestrator{
		logger: logger,
		config: config,

		nc:  nc,
		js:  js,
		jsc: jsc,
		id:  orchestratorId, // self
		jid: jobId,
		wid: clientId,

		CmdChannel: cmdChan,
	}

	return s
}
