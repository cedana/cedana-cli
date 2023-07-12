package pricing

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	aws_helpers "github.com/nravic/cedana-orch/aws"
	cedana "github.com/nravic/cedana-orch/types"
	"github.com/nravic/cedana-orch/utils"
	"github.com/rs/zerolog"
)

// gather pricing data for use in other components

type AWSPricingModel struct {
	client *ec2.Client
	ctx    context.Context
	logger *zerolog.Logger
	cfg    *utils.CedanaConfig
}

func GenAWSPricingModel() *AWSPricingModel {
	logger := utils.GetLogger()

	cfg, err := utils.InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("error initializing spot config")
	}

	// should have at least one enabled region
	if len(cfg.AWSConfig.EnabledRegions) == 0 {
		logger.Fatal().Msg("no enabled regions found, run bootstrap first or add an AWS region to the config.")
	}

	// TODO - this might be a bug.
	// Is it possible to get spot prices in other regions if the client is set to one here?
	client, err := aws_helpers.MakeClient(&cfg.AWSConfig.EnabledRegions[0])
	if err != nil {
		logger.Fatal().Err(err).Msg("could not create aws client")
	}
	return &AWSPricingModel{
		client: client,
		ctx:    context.Background(),
		logger: &logger,
		cfg:    cfg,
	}
}

// loops through and gets instantaneous spot instance price
// TODO: this is an unoptimized query! we could potentially be hitting this 100s of times per call
// we should use pointers!
func (apm *AWSPricingModel) GetPrices(instances []cedana.Instance) []cedana.Instance {
	var pricedInstances []cedana.Instance
	for _, i := range instances {
		input := &ec2.DescribeSpotPriceHistoryInput{
			MaxResults:    aws.Int32(1),
			InstanceTypes: []types.InstanceType{types.InstanceType(i.InstanceType)},
			EndTime:       aws.Time(time.Now()),
		}

		r, err := apm.client.DescribeSpotPriceHistory(apm.ctx, input)
		if err != nil {
			apm.logger.Fatal().Err(err).Msg("error fetching spot price history")
		}

		p, err := strconv.ParseFloat(*r.SpotPriceHistory[0].SpotPrice, 64)
		if err != nil {
			apm.logger.Info().Err(err).Msg("could not parse spot price")
		}
		i.Price = p
		pricedInstances = append(pricedInstances, i)
	}
	return pricedInstances
}

func (apm *AWSPricingModel) GetPrice(i *cedana.Instance) {
	input := &ec2.DescribeSpotPriceHistoryInput{
		MaxResults:    aws.Int32(1),
		InstanceTypes: []types.InstanceType{types.InstanceType(i.InstanceType)},
		EndTime:       aws.Time(time.Now()),
	}

	r, err := apm.client.DescribeSpotPriceHistory(apm.ctx, input)
	if err != nil {
		apm.logger.Fatal().Err(err).Msg("error fetching spot price history")
	}

	if r.SpotPriceHistory != nil && len(r.SpotPriceHistory) > 0 {
		p, err := strconv.ParseFloat(*r.SpotPriceHistory[0].SpotPrice, 64)
		if err != nil {
			apm.logger.Info().Err(err).Msg("could not parse spot price")
		}
		i.Price = p
	} else {
		// set price to arbitarily high so it gets filtered out
		i.Price = 100000000.00
	}
}

func (apm *AWSPricingModel) GetCapacityScores(instances []cedana.Instance) {
	// var region string
	// var avZone string

	// type PlacementScore struct {
	// score  int32
	// avZone string
	// region string
	// }

	// // extract unique instances from instances
	// instanceTypes := []string{}
	// for _, i := range instances {
	// if slices.Contains(instanceTypes, i.InstanceType) {
	// instanceTypes = append(instanceTypes, i.InstanceType)
	// }

	// for _, it := range o.input {
	// params := &ec2.GetSpotPlacementScoresInput{
	// TargetCapacity:         aws.Int32(1),
	// DryRun:                 aws.Bool(false),
	// InstanceTypes:          instanceTypes,
	// MaxResults:             aws.Int32(1),
	// SingleAvailabilityZone: aws.Bool(true),
	// RegionNames:            apm.cfg.AWSConfig.EnabledRegions, // temporary - managing complexity of launch templates and s3 buckets across regions
	// // by limiting to a set region.
	// }

	// outp, err := apm.client.GetSpotPlacementScores(apm.ctx, params)
	// if err != nil {
	// apm.logger.Fatal().Err(err).Msgf("error requesting spot placement scores")
	// }

	// apm.logger.Info().Interface("output", outp).Msg("spot placement scores")

	// // we take the last best score we got and append it.
	// for _, score := range outp.SpotPlacementScores {
	// var scoreThreshold int32 = 0
	// avZone = "n/a"
	// region = "n/a"
	// if score.AvailabilityZoneId != nil || score.Region != nil {
	// if score.AvailabilityZoneId != nil {
	// avZone = *score.AvailabilityZoneId
	// }
	// if score.Region != nil {
	// region = *score.Region
	// }
	// s.Logger.Debug().Msgf("evaluating spot placement score: %d, for zone: %s, region: %s, and instanceType %s",
	// *score.Score, avZone, region, it)
	// if *score.Score > scoreThreshold {
	// // slap best score into map
	// instanceTypeToScore[it.InstanceType] = PlacementScore{*score.Score, avZone, region}
	// scoreThreshold = *score.Score
	// s.Logger.Debug().Msgf(
	// "appending score to instanceType: %s with score %d, region %s and avZone %s",
	// it,
	// *score.Score,
	// region,
	// avZone,
	// )
	// } else {
	// s.Logger.Debug().Msgf("spot placement score does not meet threshold > %s", scoreThreshold)
	// }
	// }
	// }
	// }
	// s.Logger.Debug().Msgf("list of scored instance types: %v", instanceTypeToScore)
	// bestScore := PlacementScore{score: 0, avZone: "", region: ""}
	// bestIt := "" // dummy
	// // evaluate map and grab best score ?
	// for it, score := range instanceTypeToScore {
	// if score.score > bestScore.score {
	// bestScore = score
	// bestIt = it
	// }
	// }
	// }
}
