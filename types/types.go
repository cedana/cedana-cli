package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type UserInstanceSpecs struct {
	InstanceType string  `mapstructure:"instance_type"`
	Memory       int     `mapstructure:"memory_gb"`
	VCPUs        int     `mapstructure:"cpu_cores"`
	VRAM         int     `mapstructure:"vram_gb"`
	GPU          string  `mapstructure:"gpu"`
	MaxPrice     float64 `mapstructure:"max_price_usd_hour"`
}

// due to key-value nature of yaml, need a nested commands struct
type UserCommands struct {
	SetupCommands     Commands `mapstructure:"setup"`
	PostSetupCommands Commands `mapstructure:"post_setup"`
	PreCheckpoint     Commands `mapstructure:"pre_checkpoint"`
	PostCheckpoint    Commands `mapstructure:"post_checkpoint"`
	PreRestore        Commands `mapstructure:"pre_restore"`
	PostRestore       Commands `mapstructure:"post_restore"`
}

type Commands struct {
	C []string `mapstructure:"run"`
}

// Job type to be used to run on an instance, user-defined
// should be yaml spec
type JobFile struct {
	JobFilePath       string            `mapstructure:"job_file_path"`
	WorkDir           string            `mapstructure:"work_dir"` // TODO NR - data storage abstractions
	Containerized     bool              `mapstructure:"containerized"`
	UserInstanceSpecs UserInstanceSpecs `mapstructure:"instance_specs"`
	SetupCommands     Commands          `mapstructure:"setup"`
	Task              Commands          `mapstructure:"task"`
	RestoredTask      Commands          `mapstructure:"restored_task"`
}

// foreign keys are weird in GORM, just attach InstanceIDs for now
type Job struct {
	gorm.Model
	JobID              string    `json:"job_id"`        // ignore json unmarshal
	JobFilePath        string    `json:"job_file_path"` // absolute path of job file
	Instances          string    `json:"instances"`     // serialized instances.TODO: need to figure out associations!!
	State              JobState  `json:"state"`
	Checkpointed       bool      `json:"checkpointed"`
	LastCheckpointedAt time.Time `json:"last_checkpointed_at"` // latest checkpoint
	Bucket             string    `json:"bucket"`
}

type JobState string

const (
	JobStatePending     JobState = "PENDING"
	JobStateRunning     JobState = "RUNNING"
	JobStateFailed      JobState = "FAILED"
	JobStateDone        JobState = "DONE"
	JobStateSetupFailed JobState = "SETUP_FAILED"
)

// only serialize instanceID, can reverse lookup for instance using id
type SerializedInstance struct {
	InstanceID string `json:"instance_id"`
}

func (j *Job) GetInstanceIds() ([]SerializedInstance, error) {
	// deserialize j.instances and return list
	var instances []SerializedInstance
	err := json.Unmarshal([]byte(j.Instances), &instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// these should ideally be called from the db - keeps things consistent
func (j *Job) AppendInstance(id string) error {
	// deserialize if j.instances exists, otherwise create
	if j.Instances != "" {
		var instances []SerializedInstance
		err := json.Unmarshal([]byte(j.Instances), &instances)
		if err != nil {
			return err
		}
		instances = append(instances, SerializedInstance{InstanceID: id})
		// marshal and update
		marshalledInstances, err := json.Marshal(instances)
		if err != nil {
			return err
		}
		j.Instances = string(marshalledInstances)
	} else {
		// j.Instances is empty, just marshal and update
		marshalledInstances, err := json.Marshal([]SerializedInstance{{InstanceID: id}})
		if err != nil {
			return err
		}
		j.Instances = string(marshalledInstances)
	}

	return nil
}

type MetaState struct {
	Event            ProviderEvent    `json:"provider_event" mapstructure:"provider_event"`
	CheckpointReason CheckpointReason `json:"checkpoint_reason" mapstructure:"checkpoint_reason"`
}

type CheckpointReason string

const (
	CheckpointReasonInstanceTermination CheckpointReason = "instance_termination"
	CheckpointReasonJobTermination      CheckpointReason = "job_termination"
	CheckpointReasonHeartbeat           CheckpointReason = "heartbeat"
)

type CheckpointState string

const (
	CheckpointSuccess CheckpointState = "CHECKPOINTED"
	CheckpointFailed  CheckpointState = "CHECKPOINT_FAILED"
	RestoreSuccess    CheckpointState = "RESTORED"
	RestoreFailed     CheckpointState = "RESTORE_FAILED"
)

func InitJobFile(filepath string) (*JobFile, error) {
	var job JobFile
	viper.SetConfigFile(filepath)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&job); err != nil {
		fmt.Println(err)
		return nil, err
	}

	job.JobFilePath = filepath
	return &job, nil
}
