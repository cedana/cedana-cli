package market

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cedana/cedana-client/db"
	cedana "github.com/cedana/cedana-client/types"
	"github.com/cedana/cedana-client/utils"
	"github.com/rs/zerolog"
)

// a local provider for self-serve cedana usage

type LocalProvider struct {
	db     *db.DB
	logger *zerolog.Logger
}

func GenLocalClient() *LocalProvider {
	logger := utils.GetLogger()
	return &LocalProvider{
		db:     db.NewDB(),
		logger: &logger,
	}
}

func (l *LocalProvider) CreateInstance(Candidate *cedana.Instance) (*cedana.Instance, error) {
	return nil, nil
}

func (l *LocalProvider) DestroyInstance(i cedana.Instance) error {
	i.State = "destroyed"
	// see note in db.DeleteLocalInstance
	err := l.db.DeleteInstance(&i)
	if err != nil {
		return err
	}

	if i.Tag == "orchestrator" {
		// instance is running locally as orchestrator/daemon pair, find pid & kill
		var jobID string
		jobs := l.db.GetAllJobs()
		for _, job := range jobs {
			instanceIDs, err := job.GetInstanceIds()
			if err != nil {
				return err
			}
			for _, instanceID := range instanceIDs {
				if instanceID.InstanceID == i.CedanaID {
					jobID = job.JobID
					break
				}
			}
		}

		// kill pid and remove file
		homeDir := os.Getenv("HOME")
		pidPath := fmt.Sprintf("%s/.cedana/cedana-orchestrate.%s.pid", homeDir, jobID)
		_, err := os.Stat(pidPath)
		if err != nil {
			return nil
		}

		// read pid and kill
		pidbytes, err := os.ReadFile(pidPath)
		if err != nil {
			return err
		}

		pid, err := strconv.Atoi(string(pidbytes))
		if err != nil {
			return err
		}

		l.logger.Info().Msgf("Killing pid %d...", pid)
		process, err := os.FindProcess(pid)
		if err != nil {
			l.logger.Warn().Msgf("Could not find pid %d: %v", pid, err)
		}

		err = process.Kill()
		if err != nil {
			l.logger.Warn().Msgf("Could not kill pid %d: %v", pid, err)
			// potentially dangerous, but do nothing if this fails
		}

		err = os.Remove(pidPath)
		if err != nil {
			return err
		}

		logPath := fmt.Sprintf("%s/.cedana/cedana-orchestrate.%s.log", homeDir, jobID)
		_, err = os.Stat(logPath)
		if err != nil {
			// we don't care, just exit
			return nil
		}
		// delete logFile
		err = os.Remove(logPath)
		if err != nil {
			return err
		}

	}
	return nil
}

func (l *LocalProvider) DescribeInstance(Instances []*cedana.Instance, filter string) error {
	return nil
}

func (l *LocalProvider) GetInstanceStatus(i cedana.Instance) (*cedana.ProviderEvent, error) {
	return nil, nil
}

func (l *LocalProvider) Name() string {
	return "local"
}
