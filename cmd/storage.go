package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/povsister/scp"

	_ "github.com/rclone/rclone/backend/sftp"
)

type StorageProvider string
type StorageMode string

const (
	ProviderS3  StorageProvider = "S3"
	ProviderGCS StorageProvider = "GCS"
)

const (
	ModeMount StorageMode = "mount"
	ModeCopy  StorageMode = "copy"
)

type SyncBucket struct {
	Bucket     string          `mapstructure:"bucket"`
	Store      StorageProvider `mapstructure:"store"`
	Persistent bool            `mapstructure:"persistent"`
	Source     string          `mapstructure:"source"`
	Mode       StorageMode     `mapstructure:"mode"`
}

type Storage struct {
	localDirs     map[string]string
	mirrorBuckets map[string]string
	syncBuckets   map[string]SyncBucket
}

func verifyBucket(is *InstanceSetup, bucket string) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return err
	}

	// Create S3 service client
	svc := s3.New(sess)

	_, err = svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil { //errors if bucket name invalid
		return err
	}
	is.logger.Info().Msgf("Verified Bucket Access: %s", bucket)

	return nil
}

func (is *InstanceSetup) initStorage() (*Storage, error) {
	st := Storage{
		localDirs:     make(map[string]string),
		mirrorBuckets: make(map[string]string),
		syncBuckets:   make(map[string]SyncBucket),
	}

	for destDir, sourceDirI := range is.jobFile.Storage {
		sourceDir, valid := sourceDirI.(string)
		if valid {
			if strings.HasPrefix(sourceDir, "s3://") || strings.HasPrefix(sourceDir, "gs://") { //TODO use set instead
				err := verifyBucket(is, sourceDir[5:])
				if err != nil {
					return nil, err
				}
				st.mirrorBuckets[destDir] = sourceDir
			} else {
				_, err := os.Stat(sourceDir)
				if err != nil {
					return nil, err // folder doesn't exist
				}
				st.localDirs[destDir] = sourceDir
			}
		} else { //TODO
			continue
			// var sync SyncBucket
			// if err := viper.UnmarshalKey("storage."+destDir, &sync); err != nil {
			// 	return nil, err
			// }
			// err := verifyBucket(is, sync.Bucket)
			// if err != nil {
			// 	return nil, err
			// }
		}
	}

	return &st, nil
}

func (is *InstanceSetup) setupStorage(st *Storage, user string) error {

	err := is.createRCloneConfig()
	if err != nil {
		return err
	}

	// f, err := fs.NewFs(context.Background(), is.instance.CedanaID+":")
	// if err != nil {
	// 	return err
	// }

	// entries, err := f.List(context.Background(), "")
	// if err != nil {
	// 	return err
	// }
	// is.logger.Info().Msgf("num of entries: %d", entries.Len())

	//scp local directories
	for destDir, sourceDir := range st.localDirs {
		err := is.scpWorkDir(sourceDir, destDir)
		if err != nil {
			return err
		}
	}

	// is.scpWorkDir("~/.aws", "~/")

	// installGoofys := []string{
	// 	fmt.Sprintf("mkdir -p /home/%s/.cedana/", user),
	// 	fmt.Sprintf("wget -nc https://github.com/kahing/goofys/releases/latest/download/goofys -O /home/%s/.cedana/goofys", user),
	// 	fmt.Sprintf("sudo chmod +x /home/%s/.cedana/goofys", user),
	// 	fmt.Sprintf("ls /home/%s/.cedana/", user),
	// }
	// b = append(b, installGoofys...)
	// for destDir, bucket := range st.mirrorBuckets {
	// 	mountCmd := []string{
	// 		fmt.Sprintf("/home/%s/.cedana/goofys %s %s", user, bucket[5:], destDir),
	// 	}
	// 	b = append(b, mountCmd...)
	// }

	return nil
}

func (is *InstanceSetup) createRCloneConfig() error {
	var sshKey string
	var user string

	switch is.instance.Provider {
	case "aws":
		sshKey = is.cfg.AWSConfig.SSHKeyPath
		user = is.cfg.AWSConfig.User
		if user == "" {
			user = "ubuntu"
		}
	case "paperspace":
		sshKey = is.cfg.PaperspaceConfig.SSHKeyPath
		user = is.cfg.PaperspaceConfig.User
		if user == "" {
			user = "paperspace"
		}
	}

	configStr := []string{
		fmt.Sprintf("[%s]", is.instance.CedanaID),
		"type = sftp",
		fmt.Sprintf("host = %s", is.instance.IPAddress),
		fmt.Sprintf("user = %s", user),
		fmt.Sprintf("key_file = %s", sshKey),
		"shell_type = unix", //hashsum needed as well?
	}

	homeDir := os.Getenv("HOME")
	confPath := filepath.Join(homeDir, ".cedana/rclone.conf")
	f, err := os.Create(confPath)
	if err != nil {
		is.logger.Fatal().Err(err).Msg("error creating rclone.conf")
		return err
	}
	err = os.Chmod(confPath, 0o644)
	if err != nil {
		is.logger.Fatal().Err(err).Msg("error setting rclone.conf permissions")
		return err
	}
	// remember to close the file
	defer f.Close()

	for _, line := range configStr {
		_, err := f.WriteString(line + "\n")
		if err != nil {
			is.logger.Fatal().Err(err).Msg("error writing rclone.conf")
			return err
		}
	}

	os.Setenv("RCLONE_CONFIG", confPath)

	return nil
}

func (is *InstanceSetup) scpWorkDir(workDirPath string, destPath string) error {
	var keyPath string
	var user string

	_, err := os.Stat(workDirPath)
	if err != nil {
		// folder doesn't exist, error out and don't continue
		return err
	}

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

	err = client.CopyDirToRemote(workDirPath, destPath, &scp.DirTransferOption{})
	if err != nil {
		is.logger.Fatal().Err(err).Msg("couldn't copy local directory to instance")
		return err
	}

	is.logger.Info().Msg("transferred work dir to remote instance.")

	return nil
}
