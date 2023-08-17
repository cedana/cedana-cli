package market

import (
	"context"
	"fmt"

	"github.com/cedana/cedana-cli/db"
	cedana "github.com/cedana/cedana-cli/types"
	"github.com/cedana/cedana-cli/utils"
	"github.com/rs/zerolog"
	gcp "google.golang.org/api/compute/v1"
)

type GCPSpot struct {
	Ctx    context.Context
	Cfg    *utils.CedanaConfig
	Logger *zerolog.Logger
	Client *GCPAPI
	db     *db.DB
}

// GCP segments the API, so we need a struct here (unlike the EC2 interface, we need a separate
// interface for each segmentation)
type GCPAPI struct {
	Instances GCPInstancesServiceAPI
	Zones     GCPZonesServiceAPI
}

type GCPInstancesServiceAPI interface {
	Insert(project string, zone string, instance *gcp.Instance) *gcp.InstancesInsertCall
	Delete(project string, zone string, instance string) *gcp.InstancesDeleteCall
	Get(project string, zone string, instance string) *gcp.InstancesGetCall
}

type GCPZonesServiceAPI interface {
	List(project string) *gcp.ZonesListCall
}

func GCPInsert(project string, zone string, instance *gcp.Instance, api *GCPAPI) *gcp.InstancesInsertCall {
	return api.Instances.Insert(project, zone, instance)
}

func GCPDelete(project string, zone string, instance string, api *GCPAPI) *gcp.InstancesDeleteCall {
	return api.Instances.Delete(project, zone, instance)
}

func GCPGet(project string, zone string, instance string, api *GCPAPI) *gcp.InstancesGetCall {
	return api.Instances.Get(project, zone, instance)
}

func MakeGCPClient() (*GCPAPI, error) {
	gcp, err := gcp.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	return &GCPAPI{
		Instances: gcp.Instances,
		Zones:     gcp.Zones,
	}, nil
}

func GenGCPClient() *GCPSpot {
	logger := utils.GetLogger()

	config, err := utils.InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not set up cedana config")
	}

	client, err := MakeGCPClient()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not instantiate gcp client")
	}

	return &GCPSpot{
		Cfg:    config,
		Ctx:    context.TODO(),
		Logger: &logger,
		Client: client,
		db:     db.NewDB(),
	}

}

func (s *GCPSpot) Name() string {
	return "gcp"
}

func (s *GCPSpot) CreateInstance(i *cedana.Instance) (*cedana.Instance, error) {

	// SPOT ~= preemptible
	gcpInstance := gcp.Instance{
		MachineType: i.InstanceType,
		Scheduling: &gcp.Scheduling{
			ProvisioningModel: "SPOT",
		},
	}

	// project-id, not name!
	insert := GCPInsert(s.Cfg.GCPConfig.ProjectID, i.AvailabilityZone, &gcpInstance, s.Client)

	outp, err := insert.Do()
	if err != nil {
		return nil, err
	}

	s.Logger.Debug().Interface("output", outp).Msgf("create instance output")

	// I think this is the ID? TODO - check
	i.AllocatedID = fmt.Sprint(outp.TargetId)
	i, _ = s.db.CreateInstance(i)

	return i, nil
}

func (s *GCPSpot) DestroyInstance(i *cedana.Instance) error {
	i.State = "destroyed"

	delete := GCPDelete(s.Cfg.GCPConfig.ProjectID, i.AvailabilityZone, i.AllocatedID, s.Client)

	outp, err := delete.Do()
	if err != nil {
		return err
	}

	s.Logger.Debug().Interface("output", outp).Msgf("delete instance output")

	return nil
}

func (s *GCPSpot) DescribeInstance(instances []*cedana.Instance) error {
	// Loop through the instances and perform a get request on each one
	for _, i := range instances {
		getCall := s.Client.Instances.Get(s.Cfg.GCPConfig.ProjectID, i.Region, i.AllocatedID)
		gcpInstance, err := getCall.Do()
		if err != nil {
			s.Logger.Debug().Msgf("Error describing instances: %v", err)
			return err
		}

		// Describing the instance is also an opportunity to update the instance state
		inst := ReverseLookupInstancesById(instances, i.AllocatedID)

		if inst != nil {
			// Modify in place and persist
			inst.State = gcpInstance.Status
			inst.IPAddress = gcpInstance.NetworkInterfaces[0].AccessConfigs[0].NatIP // Assuming this is the correct path to the IP address

			s.db.UpdateInstanceByID(inst, inst.Model.ID)
		}
	}

	return nil
}

func (s *GCPSpot) GetInstanceStatus(i *cedana.Instance) (*cedana.ProviderEvent, error) {
	return nil, nil
}
