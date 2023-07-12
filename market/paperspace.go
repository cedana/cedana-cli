package market

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Paperspace/paperspace-go"
	"github.com/cedana/cedana-client/db"
	cedana "github.com/cedana/cedana-client/types"
	"github.com/cedana/cedana-client/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// implementation of Provider for paperspace
type Paperspace struct {
	cfg    *utils.CedanaConfig
	logger *zerolog.Logger
	db     *db.DB
	client *paperspace.Client
}

var templates = map[string]string{
	"gpu_enabled": "tftao63e",
	"cpu_only":    "tr3mn1ip",
}

// response and curl function created w/ GPT
// the golang paperspace api is.... lacking
type CreateMachinePaperspaceResponse struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	OS                     string `json:"os"`
	RAM                    string `json:"ram"`
	CPUs                   int    `json:"cpus"`
	GPU                    string `json:"gpu"`
	StorageTotal           string `json:"storageTotal"`
	StorageUsed            string `json:"storageUsed"`
	MachineType            string `json:"machineType"`
	UsageRate              string `json:"usageRate"`
	ShutdownTimeoutInHours int    `json:"shutdownTimeoutInHours"`
	ShutdownTimeoutForces  bool   `json:"shutdownTimeoutForces"`
	PerformAutoSnapshot    bool   `json:"performAutoSnapshot"`
	AutoSnapshotFrequency  string `json:"autoSnapshotFrequency"`
	AutoSnapshotSaveCount  int    `json:"autoSnapshotSaveCount"`
	DynamicPublicIP        bool   `json:"dynamicPublicIp"`
	AgentType              string `json:"agentType"`
	DTCreated              string `json:"dtCreated"`
	State                  string `json:"state"`
	UpdatesPending         bool   `json:"updatesPending"`
	NetworkID              string `json:"networkId"`
	PrivateIPAddress       string `json:"privateIpAddress"`
	PublicIPAddress        string `json:"publicIpAddress"`
	Region                 string `json:"region"`
	ScriptID               string `json:"scriptId"`
	DTLastRun              string `json:"dtLastRun"`
	RestorePointSnapshotID string `json:"restorePointSnapshotId"`
	RestorePointFrequency  string `json:"restorePointFrequency"`
}

func (p *Paperspace) createSinglePaperspaceMachine(i *cedana.Instance) (*CreateMachinePaperspaceResponse, error) {
	url := "https://api.paperspace.io/machines/createSingleMachinePublic"
	apiKey := p.cfg.PaperspaceConfig.APIKey

	var template string

	if p.cfg.PaperspaceConfig.TemplateId != "" {
		template = p.cfg.PaperspaceConfig.TemplateId
	} else {
		if i.GetGPUs().TotalGpuMemoryInMiB > 0 {
			template = templates["gpu_enabled"]
		} else {
			template = templates["cpu_only"]
		}
	}

	name := "cedana-" + uuid.New().String()

	data := map[string]interface{}{
		"region":          i.Region,
		"machineType":     i.InstanceType,
		"size":            1000,
		"billingType":     "hourly",
		"startOnCreate":   true,
		"dynamicPublicIp": true,
		"templateId":      template,
		"name":            name,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var machine CreateMachinePaperspaceResponse
	err = json.Unmarshal(body, &machine)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return &machine, nil
}

func GenPaperspaceClient() *Paperspace {
	logger := utils.GetLogger()
	config, err := utils.InitCedanaConfig()

	if err != nil {
		logger.Fatal().Err(err).Msg("could not setup cedana config")
	}

	client := paperspace.NewClient()
	client.APIKey = config.PaperspaceConfig.APIKey

	return &Paperspace{
		cfg:    config,
		logger: &logger,
		db:     db.NewDB(),
		client: client,
	}

}

func (p *Paperspace) CreateInstance(i *cedana.Instance) (*cedana.Instance, error) {
	// have to resort to using curl instead of the golang library
	m, err := p.createSinglePaperspaceMachine(i)

	if err != nil {
		if strings.Contains(err.Error(), "capacity") {
			// we don't know what a paperspace capacity issue looks like (yet), placeholder for now
			return nil, cedana.CapacityError{
				Message: "capacity",
			}
		}
		return nil, err
	}

	machjson, _ := json.Marshal(m)
	p.logger.Debug().Msgf("created machine %v", string(machjson))

	i.AllocatedID = m.ID
	i.IPAddress = m.PublicIPAddress

	i, err = p.db.CreateInstance(i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (p *Paperspace) DestroyInstance(i cedana.Instance) error {
	err := p.client.DeleteMachine(i.AllocatedID, paperspace.MachineDeleteParams{})
	if err != nil {
		return err
	}
	i.State = "destroyed"
	p.db.DeleteInstance(&i)

	return nil
}

func (p *Paperspace) DescribeInstance(instances []*cedana.Instance, filter string) error {
	for _, i := range instances {
		m, err := p.client.GetMachine(i.AllocatedID, paperspace.MachineGetParams{})
		if err != nil {
			p.logger.Fatal().Err(err)
		}

		machjson, _ := json.Marshal(m)
		p.logger.Debug().Msgf("updating/describing machine %v", string(machjson))

		// modify in place and persist
		i.State = string(m.State)
		i.IPAddress = StringPtrToString(&m.PublicIpAddress)

		// override ready for cross-compat
		if i.State == "ready" {
			i.State = "running"
		}

		p.db.UpdateInstanceByID(i, i.Model.ID)

	}
	return nil
}

func (p *Paperspace) Name() string {
	return "paperspace"
}

// TODO: Unimplemented
func (p *Paperspace) GetInstanceStatus(i cedana.Instance) (*cedana.ProviderEvent, error) {
	return nil, nil
}
