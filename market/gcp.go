package market

import (
	"context"

	"github.com/cedana/cedana-cli/db"
	cedana "github.com/cedana/cedana-cli/types"
	"github.com/cedana/cedana-cli/utils"
	"github.com/rs/zerolog"
)

type GCPSpot struct {
	Ctx    context.Context
	Cfg    *utils.CedanaConfig
	Logger *zerolog.Logger
	Client GCPAPI
	db     *db.DB
}

type GCPAPI interface {
}

func GenGCPClient() {

}

func (s *GCPSpot) Name() string {
	return "gcp"
}

func (s *GCPSpot) CreateInstance(i *cedana.Instance) (*cedana.Instance, error) {
	return nil, nil
}

func (s *GCPSpot) DeleteInstance(i *cedana.Instance) error {
	return nil
}

func (s *GCPSpot) DescribeInstance(Instances []*cedana.Instance, filter string) error {
	return nil
}

func (s *GCPSpot) GetInstanceStatus(i *cedana.Instance) (*cedana.ProviderEvent, error) {
	return nil, nil
}
