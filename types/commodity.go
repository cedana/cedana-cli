package types

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var InstanceStates = map[string]int{
	"pending":       0,
	"running":       1,
	"shutting-down": 2,
	"terminated":    3,
	"stopping":      4,
}

var ProviderNames = []string{
	"aws",
	"gcp",
	"azure",
	"paperspace",
}

// Types for commodity providers (e.g. AWS, GCP, etc)
// generic interface for a commodity provider, that actors we broker between (AWS, GCP, etc)
// will each implement.
type Provider interface {
	// CreateInstance takes a list of "optimal" instances as input and creates them.
	// We take multiple to circumvent any capacity issues.
	Name() string
	CreateInstance(Candidate *Instance) (*Instance, error)
	DestroyInstance(i Instance) error
	// Anywhere describeInstance is called, the entry in the db should be updated with the latest information
	DescribeInstance(Instances []*Instance, filter string) error
	// should encapsulate all events or state changes on the instance. Function that is used for state polling
	// regularly, so keep efficiency in mind when designing for a provider.
	GetInstanceStatus(i Instance) (*ProviderEvent, error)
}

type CapacityError struct {
	Code    string
	Message string
	Region  string
}

func (e CapacityError) Error() string {
	return e.Message
}

// PricingModel populates Instance.Price
type PricingModel interface {
	GetPrices() []Instance
}

type ProviderEvent struct {
	InstanceID string `json:"instance_id"`
	FaultCode  string `json:"fault_code"`
	// the below fields are deriviatives of the above, we keep the fault code for any downstream processing
	MarkedForTermination bool  `json:"marked_for_termination"`
	TerminationTime      int64 `json:"termination_time"`
}

type CedanaCluster struct {
	gorm.Model
	ClusterID uuid.UUID  `json:"cluster_id" gorm:"type:uuid"`
	Workers   []Instance `json:"workers" gorm:"foreignKey:CedanaID"`
}

type Instance struct {
	gorm.Model
	CedanaID         string  `json:"-"`            // ignore json unmarshal. Cedana ID used for NATS messages
	AllocatedID      string  `json:"allocated_id"` // id allocated by the provider, not to be used as a key
	Provider         string  `json:"provider"`
	InstanceType     string  `json:"InstanceType"`
	AcceleratorName  string  `json:"AcceleratorName"`
	AcceleratorCount int     `json:"AcceleratorCount"`
	VCPUs            float64 `json:"vCPUs"`
	MemoryGiB        float64 `json:"MemoryGiB"`
	GPUs             string  `json:"GPU"`
	Region           string  `json:"Region"`
	AvailabilityZone string  `json:"AvailabilityZone"`
	Price            float64 `json:"Price"`
	IPAddress        string  `json:"ip_addr"`
	State            string  `json:"state"`
	Tag              string  `json:"-"` // tag instance as orch or client
}

func (i *Instance) GetGPUs() GpuInfo {
	var gpu GpuInfo
	json.Unmarshal([]byte(i.GPUs), &gpu)
	return gpu
}

func (i *Instance) SerializeSelf() ([]byte, error) {
	return json.Marshal(i)
}

func (i *Instance) DeserializeSelf(data []byte) (Instance, error) {
	var inst Instance
	err := json.Unmarshal(data, &inst)
	return inst, err
}

type GPU struct {
	Name         string `json:"Name"`
	Manufacturer string `json:"Manufacturer"`
	Count        int    `json:"Count"`
	MemoryInfo   struct {
		SizeInMiB int `json:"SizeInMiB"`
	} `json:"MemoryInfo"`
}

type GpuInfo struct {
	Gpus                []GPU `json:"Gpus"`
	TotalGpuMemoryInMiB int   `json:"TotalGpuMemoryInMiB"`
}
