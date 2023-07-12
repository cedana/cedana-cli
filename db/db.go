package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/nravic/cedana-orch/types"
	"github.com/nravic/cedana-orch/utils"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type DB struct {
	logger *zerolog.Logger
	config *utils.CedanaConfig
	orm    *gorm.DB
}

func NewDB() *DB {
	logger := utils.GetLogger()

	config, err := utils.InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not set up config")
	}

	// this is the same code that's sitting in bootstrap right now
	// safe to assume that bootstrap has been run (but also safe to assume it hasn't?)
	homeDir := os.Getenv("HOME")
	configFolderPath := filepath.Join(homeDir, ".cedana")
	// check that $HOME/.cedana folder exists - create if it doesn't
	_, err = os.Stat(configFolderPath)
	if err != nil {
		logger.Info().Msg("config folder doesn't exist, creating...")
		err = os.Mkdir(configFolderPath, 0o755)
		if err != nil {
			logger.Fatal().Err(err).Msg("could not create config folder")
		}
	}

	dbPath := filepath.Join(homeDir, ".cedana", "instances.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		FullSaveAssociations: true,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to open database")
	}
	db.AutoMigrate(&types.Job{})
	db.AutoMigrate(&types.Instance{})
	return &DB{
		logger: &logger,
		config: config,
		orm:    db,
	}
}

func (db *DB) GetAllInstances() []types.Instance {
	var instances []types.Instance

	// we want to include deleted instances in this call
	db.orm.Unscoped().Find(&instances)

	return instances
}

func (db *DB) GetAllInstancesByProvider(provider string) []types.Instance {
	var instances []types.Instance
	// include deleted instances
	db.orm.Unscoped().Where("provider = ?", provider).Find(&instances)

	return instances
}

func (db *DB) GetAllRunningInstances() []types.Instance {
	var instances []types.Instance
	db.orm.Where("state = ?", "running").Find(&instances)

	return instances
}

func (db *DB) GetInstanceByCedanaID(cid string) types.Instance {
	var instance types.Instance
	result := db.orm.Unscoped().Where(&types.Instance{CedanaID: cid}).First(&instance)
	if result.Error != nil {
		db.logger.Fatal().Err(result.Error).Msg("could not find instance")
		return types.Instance{}
	}
	return instance
}

func (db *DB) GetInstanceByProviderId(id string) *types.Instance {
	var instance types.Instance
	db.orm.Where("allocated_id = ?", id).Find(&instance)

	return &instance
}

func (db *DB) GetInstancesByProvider(provider string) []types.Instance {
	var instances []types.Instance
	db.orm.Where("provider = ?", provider).Find(&instances)

	return instances
}

// get conditionally
func (db *DB) GetInstancesByState(state string) []types.Instance {
	var instances []types.Instance
	db.orm.Where("state = ?", state).Find(&instances)

	return instances
}

func (db *DB) GetInstancesByCondition(field string, query string) []types.Instance {
	var instances []types.Instance
	db.orm.Where(fmt.Sprintf("%s = ?", field), query).Find(&instances) // dangerous lol

	return instances
}

func (db *DB) CreateInstance(instance *types.Instance) (*types.Instance, error) {
	id := xid.New()
	instance.CedanaID = id.String()
	db.logger.Info().Msgf("creating instance with id %s", instance.CedanaID)
	db.orm.Create(&instance)

	return instance, nil
}

func (db *DB) UpdateInstanceByID(instance *types.Instance, id uint) error {
	db.orm.Model(&instance).Where("id = ?", id).Updates(instance)

	return nil
}

func (db *DB) UpdateInstance(instance *types.Instance) error {
	if instance != nil {
		db.orm.Model(&instance).Updates(instance)
	}
	return nil
}

// we implement gorm.Model in the instance struct, so these are soft deletes!
func (db *DB) DeleteInstance(instance *types.Instance) error {
	instance.State = "destroyed"

	db.UpdateInstance(instance)
	db.orm.Model(&instance).Delete(&instance).Where("id = ?", instance.ID)

	return nil
}

func (db *DB) DeleteInstanceByProviderID(id string) error {
	db.orm.Where("allocated_id = ?", id).Delete(&types.Instance{})

	return nil
}

func (db *DB) CreateJob(jobFile *types.JobFile) *types.Job {
	id := xid.New()
	cj := types.Job{
		JobID:       id.String(),
		JobFilePath: jobFile.JobFilePath,
	}
	db.orm.Create(&cj)

	return &cj
}

func (db *DB) GetJob(id string) *types.Job {
	var job types.Job
	result := db.orm.Model(&job).Where("job_id = ?", id).Find(&job)
	if result.Error != nil {
		db.logger.Fatal().Err(result.Error).Msg("could not find job")
	}
	return &job
}

func (db *DB) GetJobByFileName(name string) *types.Job {
	var job types.Job
	db.orm.Model(&job).Where("job_file = ?", name).Find(&job)
	return &job
}

func (db *DB) UpdateJob(job *types.Job) error {
	if job != nil {
		db.orm.Model(&job).Updates(job)
	}

	return nil
}

func (db *DB) AttachInstanceToJob(job *types.Job, instance types.Instance) {
	job.AppendInstance(instance.CedanaID)
	db.UpdateJob(job)
}

func (db *DB) UpdateJobState(job *types.Job, state types.JobState) error {
	cj := db.GetJob(job.JobID)
	if cj != nil {
		cj.State = state
		db.orm.Model(&cj).Where("job_id = ?", job.JobID).Updates(cj)
	}
	return nil
}

func (db *DB) PurgeJobs() error {
	db.orm.Model(&types.Job{}).Where("1 = 1").Delete(&types.Job{})
	return nil
}
func (db *DB) GetAllJobs() []types.Job {
	var jobs []types.Job
	db.orm.Find(&jobs)

	return jobs
}
