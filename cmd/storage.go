package cmd

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	goofys "github.com/kahing/goofys/api"
	"github.com/kahing/goofys/api/common"
	"github.com/povsister/scp"
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
				st.mirrorBuckets[destDir] = sourceDir[5:]
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

func (is *InstanceSetup) setupStorage(st *Storage, user, keyPath string) error {

	//scp local directories
	for destDir, sourceDir := range st.localDirs {
		err := is.scpWorkDir(sourceDir, destDir, user, keyPath)
		if err != nil {
			return err
		}
	}

	for mountPoint, bucket := range st.mirrorBuckets {
		flags := &common.FlagStorage{
			// File system
			MountOptions:  make(map[string]string),
			MountPoint:    mountPoint,
			MountPointArg: mountPoint,
			DirMode:       0755,
			FileMode:      0644,
			Uid:           uint32(os.Getuid()),
			Gid:           uint32(os.Getgid()),

			// Tuning,
			Cheap:        false,
			ExplicitDir:  false,
			StatCacheTTL: time.Minute,
			TypeCacheTTL: time.Minute,
			HTTPTimeout:  time.Minute,

			// Common Backend Config
			Endpoint:       "",
			UseContentType: false,

			// Debugging,
			DebugFuse:  false,
			DebugS3:    false,
			Foreground: false,
		}
		_, _, err := goofys.Mount(context.Background(), bucket, flags)
		if err != nil {
			is.logger.Fatal().Err(err).Msgf("error mounting bucket %s at %s", bucket, mountPoint)
			return err
		}
	}

	return nil
}

func (is *InstanceSetup) scpWorkDir(workDirPath string, destPath string, user string, keyPath string) error {
	_, err := os.Stat(workDirPath)
	if err != nil {
		// folder doesn't exist, error out and don't continue
		return err
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
