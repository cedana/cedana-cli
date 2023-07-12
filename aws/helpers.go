package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// utils and types for using spot across Cedana.

func generateToken() *string {
	// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Run_Instance_Idempotency.html
	token := uuid.New().String()
	return &token
}

func MakeClient(region *string) (*ec2.Client, error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(*region))
	if err != nil {
		return nil, err
	}
	// TODO: populate w/ lots of configuration
	client := ec2.NewFromConfig(cfg)

	return client, nil
}

func BuildInstanceTypes(s []string) []types.InstanceType {
	it := make([]types.InstanceType, len(s))

	for i, v := range s {
		it[i] = types.InstanceType(v)
	}

	return it
}

// GetInstanceStatus should be run as a goroutine. It routinely checks for
// termination events or statuses, and fires a message into the appropriate channel.
func GetInstanceStatus(ctx context.Context, c *ec2.Client, logger *zerolog.Logger, sc chan string) {
	// want to check for revocations/terminations/etc
	params := &ec2.DescribeSpotInstanceRequestsInput{}

	for {
		result, err := c.DescribeSpotInstanceRequests(ctx, params)
		if err != nil {
			logger.Fatal().Err(err).Msg("could not request describe spot instance from ec2 api")
		}

		for _, r := range result.SpotInstanceRequests {
			if *r.Status.Code == "instance-terminated-by-price" {
				logger.Info().Msgf("instance % is about to be terminated with code %s", *r.InstanceId, *r.Status.Code)
				sc <- *r.Status.Code
			}
		}

		time.Sleep(30 * time.Second)
	}
}

// credentials
