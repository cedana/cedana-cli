package self_serve

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cedana/cedana-cli/db"
	cedana "github.com/cedana/cedana-cli/types"
	"github.com/cedana/cedana-cli/utils"
	scp "github.com/povsister/scp"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

type InstanceSetup struct {
	logger   *zerolog.Logger
	cfg      *utils.CedanaConfig
	instance cedana.Instance
	jobFile  *cedana.JobFile
	job      *cedana.Job
}

var jobFile string
var instanceId string

var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Manually set up a launched instance with Cedana defaults and user-provided scripts",
	Long:  "Provide commands to run on the remote instance in user_commands.yaml in the ~/.cedana config folder",
	RunE: func(cmd *cobra.Command, args []string) error {
		// ClientSetup takes a SpotInstance as input - match against the state file
		db := db.NewDB()
		l := utils.GetLogger()
		jobFile, err := cedana.InitJobFile(jobFile)
		if err != nil {
			l.Fatal().Err(err).Msg("could not set up cedana job")
		}

		instance := db.GetInstanceByCedanaID(instanceId)
		if instance.IPAddress == "" {
			return fmt.Errorf("could not find instance with id %s", instanceId)
		}
		cfg, err := utils.InitCedanaConfig()
		if err != nil {
			return fmt.Errorf("could not load config %v", err)
		}

		is := InstanceSetup{
			logger:   &l,
			cfg:      cfg,
			instance: instance,
			jobFile:  jobFile,
		}

		is.ClientSetup()
		return nil
	},
}

func BuildInstanceSetup(i cedana.Instance, job cedana.Job) *InstanceSetup {
	l := utils.GetLogger()

	cfg, err := utils.InitCedanaConfig()
	if err != nil {
		l.Fatal().Err(err).Msg("could not load spot config")
	}

	jobFile, err := cedana.InitJobFile(job.JobFilePath)
	if err != nil {
		l.Fatal().Err(err).Msg("could not parse cedana job file")
	}

	return &InstanceSetup{
		logger:   &l,
		cfg:      cfg,
		instance: i,
		jobFile:  jobFile,
		job:      &job,
	}
}

func (is *InstanceSetup) CreateConn() (*ssh.Client, error) {
	var keyPath string
	var user string

	if is.instance.Provider == "aws" {
		user = is.cfg.AWSConfig.User
		if user == "" {
			user = "ubuntu"
		}
		keyPath = is.cfg.AWSConfig.SSHKeyPath
	}

	if is.instance.Provider == "paperspace" {
		user = is.cfg.PaperspaceConfig.User
		if user == "" {
			user = "paperspace"
		}
		keyPath = is.cfg.PaperspaceConfig.SSHKeyPath
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		is.logger.Fatal().Err(err).Msg("error loading keyfile to ssh into instance")
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		is.logger.Fatal().Err(err).Msg("error parsing key")
		return nil, err
	}
	config := &ssh.ClientConfig{
		User:              user,
		Auth:              []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyAlgorithms: []string{"ssh-ed25519"},
		// not sure how much of a mistake this is, considering the setup is only done once
		// and everything happens as soon as AWS gives us the IP address.
		// TODO NR: make more secure!
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var conn *ssh.Client
	for i := 0; i < 5; i++ {
		conn, err = ssh.Dial("tcp", fmt.Sprintf("%s:22", is.instance.IPAddress), config)
		if err == nil {
			break
		}
		is.logger.Warn().Msgf("instance setup failed (attempt %d/%d) with error: %v. Retrying...", i+1, 5, err)
		time.Sleep(40 * time.Second)

	}

	if err != nil {
		is.logger.WithLevel(zerolog.FatalLevel).Err(err).Msg("dial failed")
		return nil, err
	}

	return conn, nil
}

// Runs cedana-specific and user-specified instantiation scripts for a client instance in an SSH session.
func (is *InstanceSetup) ClientSetup() error {

	//create connection first (and retry) beforehand
	conn, err := is.CreateConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	// copy workdir if specified and exists
	var workDir string
	if is.jobFile.WorkDir != "" {
		_, err := os.Stat(is.jobFile.WorkDir)
		if err != nil {
			// folder doesn't exist, error out and don't continue
			return err
		} else {
			workDir = is.jobFile.WorkDir
			err = is.scpWorkDir(workDir)
			if err != nil {
				return err
			}
		}
	}

	var user string

	if is.instance.Provider == "aws" {
		user = is.cfg.AWSConfig.User
		if user == "" {
			user = "ubuntu"
		}
	}

	if is.instance.Provider == "paperspace" {
		user = is.cfg.PaperspaceConfig.User
		if user == "" {
			user = "paperspace"
		}
	}

	var cmds []string
	is.buildUserSetupCommands(&cmds)
	err = is.execCommands(cmds, conn)
	if err != nil {
		is.logger.Fatal().Err(err).Msg("error executing commands")
		return err
	}

	return nil
}

// Take a list of commands and execute them, creating a new session each time
// avoids any issues with only being able to run session.Run once, and is more elegant
// than timing.

// TODO: Add noisy flag or move to debug logger
func (is *InstanceSetup) execCommands(cmds []string, conn *ssh.Client) error {
	for _, cmd := range cmds {
		// TODO: this is pretty inefficient, we don't need a new session for each command!
		session, err := conn.NewSession()
		if err != nil {
			is.logger.Fatal().Err(err).Msg("session failed")
			return err
		}
		defer session.Close()

		stdout, err := session.StdoutPipe()
		if err != nil {
			is.logger.Fatal().Err(err).Msg("error getting stdout pipe")
			return err
		}
		stderr, err := session.StderrPipe()
		if err != nil {
			is.logger.Fatal().Err(err).Msg("error getting stderr pipe")
			return err
		}

		// closing the session closes the stdout/errPipes, which functionally terminates
		// these goroutines.
		// TODO: ugly logging from this, but whatever
		go func() {
			_, err := io.Copy(is.logger, stdout)
			if err != nil {
				is.logger.Fatal().Err(err).Msg("could not copy stdout")
			}
		}()

		go func() {
			_, err := io.Copy(is.logger, stderr)
			if err != nil {
				is.logger.Fatal().Err(err).Msg("could not copy stderr")
			}
		}()

		is.logger.Debug().Msgf("running: `%s`", cmd)
		err = session.Run(cmd)
		if err != nil {
			is.logger.Fatal().Err(err).Msg("error running command on remote host")
			return err
		}
	}
	return nil
}

// exec work is different because it fires off the work in a goroutine - preventing hangups
// TODO - should consolidate this with execCommand
func (is *InstanceSetup) execCommandAsync(cmds []string, conn *ssh.Client) error {
	for _, cmd := range cmds {
		session, err := conn.NewSession()
		if err != nil {
			is.logger.Fatal().Err(err).Msg("session failed")
			return err
		}

		stdout, err := session.StdoutPipe()
		if err != nil {
			is.logger.Fatal().Err(err).Msg("error getting stdout pipe")
			return err
		}

		stderr, err := session.StderrPipe()
		if err != nil {
			is.logger.Fatal().Err(err).Msg("error getting stderr pipe")
			return err
		}

		go func() {
			_, err := io.Copy(is.logger, stdout)
			if err != nil {
				is.logger.Fatal().Err(err).Msg("could not copy stdout")
			}
		}()

		go func() {
			_, err := io.Copy(is.logger, stderr)
			if err != nil {
				is.logger.Fatal().Err(err).Msg("could not copy stderr")
			}
		}()

		is.logger.Debug().Msgf("running: `%s`", cmd)
		go func() {
			err = session.Run(cmd)
			if err != nil {
				is.logger.Fatal().Err(err).Msg("error running command on remote host")
			}
			session.Close()
		}()
	}

	// Wait for all commands to complete
	for range cmds {
		<-time.After(500 * time.Millisecond) // Add a small delay between commands
	}

	return nil
}

// Attaches user specified comments (specified as yaml) to a
// list of startup commands. Have to happen post cedana-setup
func (is *InstanceSetup) buildUserSetupCommands(b *[]string) {
	cmds := is.jobFile.SetupCommands
	// if we _did_ manage to populate it
	if len(cmds.C) > 0 {
		*b = append(*b, cmds.C...)
	}
}

func (is *InstanceSetup) scpWorkDir(workDirPath string) error {
	var keyPath string
	var user string

	if is.instance.Provider == "aws" {
		user = is.cfg.AWSConfig.User
		if user == "" {
			user = "ubuntu"
		}
		keyPath = is.cfg.AWSConfig.SSHKeyPath
	}

	if is.instance.Provider == "paperspace" {
		user = is.cfg.PaperspaceConfig.User
		if user == "" {
			user = "paperspace"
		}
		keyPath = is.cfg.PaperspaceConfig.SSHKeyPath
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		is.logger.Fatal().Err(err).Msg("error loading keyfile to scp data to instance")
		return err
	}
	config, err := scp.NewSSHConfigFromPrivateKey(user, keyBytes)
	if err != nil {
		is.logger.Fatal().Err(err).Msg("error creating config from key file")
		return err
	}

	client, err := scp.NewClient(is.instance.IPAddress, config, &scp.ClientOption{})
	if err != nil {
		is.logger.Fatal().Err(err).Msg("couldn't establish a connection to the remote server")
		return err
	}
	defer client.Close()

	err = client.CopyDirToRemote(workDirPath, ".", &scp.DirTransferOption{})
	if err != nil {
		is.logger.Fatal().Err(err).Msg("couldn't copy local directory to instance")
		return err
	}

	is.logger.Info().Msg("transferred work dir to remote instance.")

	return nil
}

func init() {
	runSelfServeCmd.AddCommand(SetupCmd)
	SetupCmd.Flags().StringVarP(&jobFile, "job", "j", "", "job file to use for setup")
	SetupCmd.Flags().StringVarP(&instanceId, "instance", "i", "", "provider instance id to setup")
	cobra.MarkFlagRequired(SetupCmd.Flags(), "job")

}
