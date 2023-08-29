package cmd

import (
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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

	is.logger.Info().Msgf("Bucket: %s", bucket)
	_, err = svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil { //errors if bucket name invalid
		return err
	}

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

func (st *Storage) MountStorage() error {
	return nil
}
