package market

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cedana/cedana-cli/db"
	"github.com/cedana/cedana-cli/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cedana "github.com/cedana/cedana-cli/types"
)

var lastUsedRegion string

type Spot struct {
	Ctx          context.Context
	Cfg          *utils.CedanaConfig
	Logger       *zerolog.Logger
	Client       EC2CreateInstanceAPI
	LaunchParams *ec2.CreateFleetInput
	db           *db.DB
}

type EC2CreateInstanceAPI interface {
	CreateFleet(ctx context.Context,
		params *ec2.CreateFleetInput,
		optFns ...func(*ec2.Options)) (*ec2.CreateFleetOutput, error)

	CreateTags(ctx context.Context,
		params *ec2.CreateTagsInput,
		optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)

	GetSpotPlacementScores(ctx context.Context,
		params *ec2.GetSpotPlacementScoresInput,
		optFns ...func(*ec2.Options)) (*ec2.GetSpotPlacementScoresOutput, error)

	DescribeLaunchTemplateVersions(ctx context.Context,
		params *ec2.DescribeLaunchTemplateVersionsInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeLaunchTemplateVersionsOutput, error)

	CreateLaunchTemplate(ctx context.Context,
		params *ec2.CreateLaunchTemplateInput,
		optFns ...func(*ec2.Options)) (*ec2.CreateLaunchTemplateOutput, error)

	DescribeInstances(ctx context.Context,
		params *ec2.DescribeInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)

	TerminateInstances(ctx context.Context,
		params *ec2.TerminateInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)

	DescribeAvailabilityZones(ctx context.Context,
		params *ec2.DescribeAvailabilityZonesInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeAvailabilityZonesOutput, error)

	DescribeSubnets(ctx context.Context,
		params *ec2.DescribeSubnetsInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)

	DescribeVpcs(ctx context.Context,
		params *ec2.DescribeVpcsInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)

	CreateSubnet(ctx context.Context,
		params *ec2.CreateSubnetInput,
		optFns ...func(*ec2.Options)) (*ec2.CreateSubnetOutput, error)

	DescribeSpotInstanceRequests(ctx context.Context,
		params *ec2.DescribeSpotInstanceRequestsInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeSpotInstanceRequestsOutput, error)
}

func AWSCreateFleet(c context.Context, api EC2CreateInstanceAPI, input *ec2.CreateFleetInput) (*ec2.CreateFleetOutput, error) {
	return api.CreateFleet(c, input)
}

func AWSGetScores(c context.Context, api EC2CreateInstanceAPI, input *ec2.GetSpotPlacementScoresInput) (*ec2.GetSpotPlacementScoresOutput, error) {
	return api.GetSpotPlacementScores(c, input)
}

func AWSGetLaunchTemplate(c context.Context, api EC2CreateInstanceAPI, input *ec2.DescribeLaunchTemplateVersionsInput) (*ec2.DescribeLaunchTemplateVersionsOutput, error) {
	return api.DescribeLaunchTemplateVersions(c, input)
}

func AWSCreateLaunchTemplate(c context.Context, api EC2CreateInstanceAPI, input *ec2.CreateLaunchTemplateInput) (*ec2.CreateLaunchTemplateOutput, error) {
	return api.CreateLaunchTemplate(c, input)
}

func AWSDescribeInstances(c context.Context, api EC2CreateInstanceAPI, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return api.DescribeInstances(c, input)
}

func AWSTerminateInstances(c context.Context, api EC2CreateInstanceAPI, input *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	return api.TerminateInstances(c, input)
}

func AWSDescribeAvailabilityZones(c context.Context, api EC2CreateInstanceAPI, input *ec2.DescribeAvailabilityZonesInput) (*ec2.DescribeAvailabilityZonesOutput, error) {
	return api.DescribeAvailabilityZones(c, input)
}

func AWSDescribeSubnets(c context.Context, api EC2CreateInstanceAPI, input *ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error) {
	return api.DescribeSubnets(c, input)
}

func AWSDescribeVPCs(c context.Context, api EC2CreateInstanceAPI, input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
	return api.DescribeVpcs(c, input)
}

func AWSCreateSubnet(c context.Context, api EC2CreateInstanceAPI, input *ec2.CreateSubnetInput) (*ec2.CreateSubnetOutput, error) {
	return api.CreateSubnet(c, input)
}

func AWSDescribeSpotInstanceRequests(c context.Context, api EC2CreateInstanceAPI, input *ec2.DescribeSpotInstanceRequestsInput) (*ec2.DescribeSpotInstanceRequestsOutput, error) {
	return api.DescribeSpotInstanceRequests(c, input)
}

func MakeClient(region *string, ctx context.Context) (*ec2.Client, error) {

	if region == nil {
		// grab the last used region. TODO:Urgent: This is not ideal for cross-region stuff!
		region = &lastUsedRegion
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(*region))
	if err != nil {
		return nil, err
	}
	// TODO: populate w/ lots of configuration
	client := ec2.NewFromConfig(cfg)

	return client, nil
}

func GenSpotClient() *Spot {

	logger := utils.GetLogger()

	config, err := utils.InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not set up cedana config")
	}

	// start with the first (0th) element in the config - it's reset anyway if something better is found
	client, err := MakeClient(&config.AWSConfig.EnabledRegions[0], context.Background())
	if err != nil {
		logger.Fatal().Err(err).Msg("could not instantiate AWS client")
	}

	s := &Spot{
		Cfg:    config,
		Ctx:    context.TODO(),
		Logger: &logger,
		Client: client,
		db:     db.NewDB(),
	}

	return s

}

func genBool(b bool) *bool {
	return &b
}

func (s *Spot) Name() string {
	return "aws"
}

func (s *Spot) spotSetup(i *cedana.Instance) error {
	// launch template check
	valid := s.isValidLaunchTemplateName(s.Cfg.AWSConfig.LaunchTemplateName)
	// TODO NR: Nothing is happening here w/ launch templates
	if !valid {
		s.Logger.Info().Msg("launch template does not exist in set region, creating...")
	}
	s.LaunchParams = &ec2.CreateFleetInput{
		// launch template config is stupid and annoying, but the overriding the override param
		// gives us the flexibility we need. There's too many lists though!

		LaunchTemplateConfigs: []types.FleetLaunchTemplateConfigRequest{
			{
				LaunchTemplateSpecification: genFleetLaunchTemplateSpec(s.Cfg.AWSConfig.LaunchTemplateName),
				Overrides: []types.FleetLaunchTemplateOverridesRequest{
					s.genFleetOverrides(i),
				},
			},
		},
		ClientToken:                      GenerateToken(),
		DryRun:                           isDryRun(),
		ReplaceUnhealthyInstances:        genBool(false),
		TerminateInstancesWithExpiration: genBool(false),
		Type:                             types.FleetTypeInstant,
		TargetCapacitySpecification: &types.TargetCapacitySpecificationRequest{
			TotalTargetCapacity:       aws.Int32(1),
			DefaultTargetCapacityType: types.DefaultTargetCapacityTypeSpot,
			SpotTargetCapacity:        aws.Int32(1),
			// There is a specification for unit type here, where we the default is a instance. Can change
			// unit type to vCPU or memory
		},
	}

	return nil
}

func (s *Spot) CreateInstance(i *cedana.Instance) (*cedana.Instance, error) {
	// HACK! We reset the client here with a correct region
	cfg, err := config.LoadDefaultConfig(s.Ctx, config.WithRegion(i.Region))
	if err != nil {
		s.Logger.Fatal().Err(err).Msgf("error instantiating aws config")
	}

	s.Logger.Info().Msgf("trying to create instance in AWS region: %s", i.Region)
	// reset client
	s.Client = ec2.NewFromConfig(cfg)

	s.Logger.Info().Msg("creating fleet...")

	err = s.spotSetup(i)
	if err != nil {
		return nil, err
	}

	// TODO: this needs a retrier
	outp, err := AWSCreateFleet(s.Ctx, s.Client, s.LaunchParams)
	if err != nil {
		s.Logger.Fatal().Err(err).Msgf("could not create instance")
		return nil, err
	}

	s.Logger.Debug().Interface("output", outp).Msgf("create fleet output")

	// check for any hidden errors
	if len(outp.Errors) != 0 {
		for _, err := range outp.Errors {
			if strings.Contains(*err.ErrorCode, "InsufficientInstanceCapacity") {
				// systems upstream should catch this error and try launching other instances
				s.Logger.Debug().Msg("capacity error, retrying...")
				return nil, &cedana.CapacityError{
					Code:    "capacity",
					Message: *err.ErrorMessage,
				}
			} else if strings.Contains(*err.ErrorCode, "MaxSpotInstanceCountExceeded") {
				// max instance count exceeded for the region. hack around this by retrying and disabling region
				s.Logger.Debug().Msg("max instance count exceeded, retrying...")
				return nil, &cedana.CapacityError{
					Code:    "capacity",
					Message: *err.ErrorMessage,
					Region:  i.Region,
				}
			} else {
				// stupid aws err type doesn't implement error
				s.Logger.Fatal().Interface("err", err).Msg("error during instantiation of fleet")
			}
		}
	}
	// assume we've successfully launched by getting here, so persist & modify
	// we also only launch a single instance ever using this API, so safe to assume we want 0th elements
	i.AllocatedID = outp.Instances[0].InstanceIds[0]
	i, _ = s.db.CreateInstance(i)

	return i, nil
}

func isDryRun() *bool {
	var b = false
	return &b
}

func genFleetLaunchTemplateSpec(launchTemplateName string) *types.FleetLaunchTemplateSpecificationRequest {
	version := "$Latest"
	return &types.FleetLaunchTemplateSpecificationRequest{
		LaunchTemplateName: &launchTemplateName,
		Version:            &version,
	}
}

// Generates overrides for the fleet launch templates. Ideally we want everything overriden, so it
// can be set programatically.
func (s *Spot) genFleetOverrides(i *cedana.Instance) types.FleetLaunchTemplateOverridesRequest {
	// meaty function, overrides a lot of the AWS decisions on spinning up spot instances
	// we overload the AWS request to force our instance size recommendations to go through

	var image *string

	subnet := s.getSubnets(i)
	if subnet == nil {
		s.Logger.Fatal().Msg("could not find or create a subnet for the instance")
	}

	// default to cedana AMIs
	if s.Cfg.AWSConfig.ImageId == "" {
		image = s.AMIByRegion(i.Region)
	} else {
		image = &s.Cfg.AWSConfig.ImageId
	}

	overrideRequest := types.FleetLaunchTemplateOverridesRequest{
		ImageId:          image,
		InstanceType:     types.InstanceType(i.InstanceType),
		AvailabilityZone: aws.String(s.azIDtoAvZone(i.AvailabilityZone)),
		SubnetId:         subnet,
	}
	return overrideRequest
}

func GenerateToken() *string {
	// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Run_Instance_Idempotency.html
	token := uuid.New().String()
	return &token
}

// Checks to see if a launch template name is valid. Used to create one if it doesn't exist.
// We don't search on the id because the name is the same across regions.
func (s *Spot) isValidLaunchTemplateName(name string) bool {
	params := &ec2.DescribeLaunchTemplateVersionsInput{
		DryRun:             aws.Bool(false),
		LaunchTemplateName: &name,
	}
	_, err := s.Client.DescribeLaunchTemplateVersions(s.Ctx, params)
	if err != nil {
		return false
	} else {
		return true
	}
}

func (s *Spot) DescribeInstance(instances []*cedana.Instance, filter string) error {
	var f []types.Filter
	if filter != "" {
		f = append(f, types.Filter{
			Name:   aws.String("instance-state-name"),
			Values: []string{filter},
		})
	}

	var ids []string
	for _, i := range instances {
		ids = append(ids, i.AllocatedID)
	}

	out, err := AWSDescribeInstances(s.Ctx, s.Client, &ec2.DescribeInstancesInput{
		InstanceIds: ids,
		Filters:     f,
	})

	if err != nil {
		s.Logger.Debug().Msgf("Error describing instances: %v", err)
		return err
	}

	// Describing the instance is also an opportunity to update the instance state
	for _, res := range out.Reservations {
		for _, i := range res.Instances {
			// Describing the instance is also an opportunity to update the instance state
			inst := ReverseLookupInstancesById(instances, StringPtrToString(i.InstanceId))

			if inst != nil {
				// modify in place and persist
				inst.State = string(i.State.Name)
				inst.IPAddress = StringPtrToString(i.PublicDnsName)

				s.db.UpdateInstanceByID(inst, inst.Model.ID)

			}
		}
	}
	return nil
}

func ReverseLookupInstancesById(instances []*cedana.Instance, id string) *cedana.Instance {
	for _, i := range instances {
		if i.AllocatedID == id {
			return i
		}
	}
	return nil
}

func StringPtrToString(p *string) string {
	if p != nil {
		return *p
	}
	return "(nil)"
}

// TODO: should only take one instance at a time!
func (s *Spot) DestroyInstance(instance cedana.Instance) error {
	cfg, err := config.LoadDefaultConfig(s.Ctx, config.WithRegion(instance.Region))
	if err != nil {
		s.Logger.Fatal().Err(err).Msgf("error instantiating aws config")
	}

	s.Client = ec2.NewFromConfig(cfg)

	var instanceIds []string
	instanceIds = append(instanceIds, instance.AllocatedID)
	out, err := AWSTerminateInstances(s.Ctx, s.Client, &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
		DryRun:      aws.Bool(false),
	})

	if err != nil {
		if strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
			// clean up instance, but let user know
			s.Logger.Info().Msg("Instance not found on AWS - removing from local database...")
			s.db.DeleteInstance(&instance)
			s.Logger.Info().Msg("done!")
			return nil
		}
		s.Logger.Fatal().Err(err).Msg("Error terminating instances")
	}

	for _, stateChange := range out.TerminatingInstances {
		s.Logger.Debug().Interface("instance state change: ", stateChange).Msg("terminate instance output - ")
	}

	s.db.DeleteInstance(&instance)

	return nil
}

func (s *Spot) azIDtoAvZone(azID string) string {
	r, err := AWSDescribeAvailabilityZones(s.Ctx, s.Client, &ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		s.Logger.Fatal().Err(err).Msg("error describing availability zones")
	}

	azMap := make(map[string]string)

	for _, az := range r.AvailabilityZones {
		azMap[*az.ZoneName] = *az.ZoneId
	}

	var z string
	for zone, id := range azMap {
		if id == azID {
			z = zone
		}
	}

	if z != "" {
		return z
	} else {
		s.Logger.Fatal().Msg("could not find availability zone")
		return ""
	}
}

// get the default VPC Id for the aws account.
// used to create a subnet if there isn't one present
func (s *Spot) getDefaultVPCId() *string {
	out, err := AWSDescribeVPCs(s.Ctx, s.Client, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("isDefault"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		s.Logger.Fatal().Err(err).Msg("error describing vpcs")
		return nil
	}

	if len(out.Vpcs) > 0 {
		return out.Vpcs[0].VpcId
	} else {
		return nil
	}
}

// check if subnet exists in availability zone for aws account
// if it doesn't exist, create one using the default VPC id for the account
func (s *Spot) getSubnets(i *cedana.Instance) *string {
	var subnet *string
	// we want to check if there's even a subnet in the region
	// if there isn't, we need to create one
	defaultVpc := s.getDefaultVPCId()
	if defaultVpc == nil {
		s.Logger.Fatal().Msg("no default VPC found for AWS account")
		return nil
	}

	out, err := AWSDescribeSubnets(s.Ctx, s.Client, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("availability-zone"),
				Values: []string{s.azIDtoAvZone(i.AvailabilityZone)},
			},
			{
				Name:   aws.String("vpc-id"),
				Values: []string{*defaultVpc},
			},
		},
	})
	if err != nil {
		s.Logger.Fatal().Err(err).Msg("error describing subnets")
	}

	// if subnet already exists, just return the first one
	if len(out.Subnets) > 0 {
		subnet = out.Subnets[0].SubnetId
	} else {
		// create a subnet
		out, err := AWSCreateSubnet(s.Ctx, s.Client, &ec2.CreateSubnetInput{
			AvailabilityZone: aws.String(s.azIDtoAvZone(i.AvailabilityZone)),
			VpcId:            defaultVpc,
			CidrBlock:        aws.String("172.16.0.0/12"),
		})
		if err != nil {
			s.Logger.Fatal().Err(err).Msg("error creating subnet")
		}

		subnet = out.Subnet.SubnetId
	}

	return subnet
}

func (s *Spot) GetInstanceStatus(instance cedana.Instance) (*cedana.ProviderEvent, error) {
	e := &cedana.ProviderEvent{}
	out, err := AWSDescribeSpotInstanceRequests(s.Ctx, s.Client, &ec2.DescribeSpotInstanceRequestsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: []string{instance.AllocatedID},
			},
		},
	})

	if err != nil {
		s.Logger.Info().Msgf("error describing spot instance requests: %v", err)
		return nil, err
	}

	if len(out.SpotInstanceRequests) == 0 {
		return nil, nil
	}

	// we only want 0th value
	req := out.SpotInstanceRequests[0]

	spotStatusCode := req.Status.Code
	e.FaultCode = *spotStatusCode

	if *spotStatusCode == "marked-for-stop" ||
		*spotStatusCode == "marked-for-termination" ||
		*spotStatusCode == "instance-terminated-no-capacity" {
		e.MarkedForTermination = true
		// this is fake, need a better way to get the termination time
		e.TerminationTime = time.Now().Add(time.Minute * 2).Unix()

	}

	return e, nil
}

func (s *Spot) AMIByRegion(region string) *string {
	// return the default cedana AMI for region we're in
	amiToRegion := map[string]string{
		"us-east-1":      "ami-0fd08154bf663bc22",
		"us-east-2":      "ami-02b07400abd6304af",
		"us-west-1":      "ami-0f546131043d71e89",
		"us-west-2":      "ami-02a4b30abd8e85a59",
		"ap-southeast-1": "ami-007c15aa36db08000",
		"ap-southeast-2": "ami-0a60559019a825c9f",
	}

	if _, ok := amiToRegion[region]; !ok {
		s.Logger.Fatal().Msgf("No Default Cedana AMI exists for region %s, recommend adding an AMI to config", region)
		return nil
	} else {
		return aws.String(amiToRegion[region])
	}
}
