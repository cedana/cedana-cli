package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog"
)

func CopyTemplate(fromRegion string, toRegion string, launchTemplateId string, launchTemplateName string, logger *zerolog.Logger) (*string, error) {
	var client *ec2.Client
	ctx := context.TODO()
	client, err := MakeClient(&fromRegion)
	if err != nil {
		return nil, err
	}

	var launchTemplates []types.ResponseLaunchTemplateData
	// copies a launch template fromRegion toRegion
	describeLaunchTemplateInput := &ec2.DescribeLaunchTemplateVersionsInput{
		DryRun:           aws.Bool(false),
		LaunchTemplateId: &launchTemplateId,
		MinVersion:       aws.String("1"),
	}

	outp, err := client.DescribeLaunchTemplateVersions(ctx, describeLaunchTemplateInput)
	if err != nil {
		return nil, err
	}

	for _, ltVersions := range outp.LaunchTemplateVersions {
		launchTemplates = append(launchTemplates, *ltVersions.LaunchTemplateData)
		logger.Info().Interface("template data", *ltVersions.LaunchTemplateData)
	}

	resp := launchTemplates[0]

	// create new client
	client, err = MakeClient(&toRegion)
	if err != nil {
		return nil, err
	}

	// just pick the latest launchTemplate for now. There's some rewriting shenangians but ignore/
	// going to have to rewrite entire RequestLaunchTemplate because we can't typecast from Response -> Request
	req := &types.RequestLaunchTemplateData{
		// ignore BlockDeviceMapping
		// ignore CapacityReservationSpecification
		// ignore CpuOptions (all of these are Overriden anyway)
		EbsOptimized: resp.EbsOptimized,
		ImageId:      resp.ImageId,
		UserData:     resp.UserData, // most important
		// can add other stuff as they get filled out
	}

	out, err := client.CreateLaunchTemplate(ctx, &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: req,
		LaunchTemplateName: &launchTemplateName,
		ClientToken:        generateToken(),
		DryRun:             aws.Bool(false),
	})

	if err != nil {
		return nil, err
	}

	logger.Info().Interface("createLaunchTemplateOutput", out)

	newTemplateId := out.LaunchTemplate.LaunchTemplateId

	// verify that creation happened successfully
	// create LaunchTemplates in new region

	return newTemplateId, nil
}
