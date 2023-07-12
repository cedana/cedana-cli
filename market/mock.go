package market

import cedana "github.com/cedana/cedana-client/types"

// a mock provider that implements the Provider interface

type MockProvider struct {
	MockTermination bool // helps fiddle with the termination logic
}

func (m *MockProvider) CreateInstance(Candidate *cedana.Instance) (*cedana.Instance, error) {
	return nil, nil
}

func (m *MockProvider) DestroyInstance(i cedana.Instance) error {
	return nil
}

func (m *MockProvider) DescribeInstance(Instances []*cedana.Instance, filter string) error {
	return nil
}

func (m *MockProvider) GetInstanceStatus(i cedana.Instance) (*cedana.ProviderEvent, error) {
	if m.MockTermination {
		return &cedana.ProviderEvent{
			InstanceID:           i.AllocatedID,
			FaultCode:            "terminated",
			MarkedForTermination: true,
		}, nil
	}
	return nil, nil
}

func (m *MockProvider) Name() string {
	return "mock"
}
