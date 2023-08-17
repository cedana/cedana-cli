package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/google/uuid"
)

// utils and types for using spot across Cedana.

func generateToken() *string {
	// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Run_Instance_Idempotency.html
	token := uuid.New().String()
	return &token
}

func MakeEC2Client(region *string) (*ec2.Client, error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(*region))
	if err != nil {
		return nil, err
	}
	// TODO: populate w/ lots of configuration
	client := ec2.NewFromConfig(cfg)

	return client, nil
}
