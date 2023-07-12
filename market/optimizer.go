package market

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/cedana/cedana-cli/market/catalog"
	"github.com/cedana/cedana-cli/market/pricing"
	cedana "github.com/cedana/cedana-cli/types"
	"github.com/cedana/cedana-cli/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog"
	"k8s.io/utils/strings/slices"
)

// functions to optimally pick an instance
// these functions should be "functional" - i.e you should
// be able to chain them to add filter on filter.
// eg. instances = priceOptimizer().regionOptimizer().capacityOptimizer()...
// these should have some way of generating a proof as well for maximum transparency
// everything here right now is super hacky though and written to move as fast as possible.
// justifying because there's some questionable decisionmaking :)

type Optimizer struct {
	input   []cedana.Instance
	cfg     *utils.CedanaConfig
	logger  *zerolog.Logger
	output  []cedana.Instance
	ctx     context.Context
	jobFile *cedana.JobFile
}

var optimalOrchestratorSpecs = cedana.UserInstanceSpecs{
	Memory:   1,
	VCPUs:    1,
	MaxPrice: 0.2,
}

// TODO: NR. This is a hack for now, and we eventually want to move
// the orchestrator somewhere. Need to think about it though.
var defaultOrchestratorProvider = "aws"

// maximum number of instances to consider for a given provider
// when launching to circumvent capacity issues.
// TODO NR
var maxInstancesToConsider = 10

func (o *Optimizer) LoadCatalogsFromR2() {
	// pull catalogs into memory for each enabled provider
	for _, provider := range o.cfg.EnabledProviders {
		dl := catalog.DownloadFromR2(provider)
		o.input = append(o.input, dl...)
	}
}

func OptimizeOrchestrator() []cedana.Instance {
	// returns an optimal instance (read - cheap!) for an orchestrator
	// ideally we don't care which provider the worker instance is also launched in,
	// this is just a hack for simplicity
	// job can't be nillable - so create an empty one for specs

	// don't just leave this in here!
	o := Build(&cedana.JobFile{
		UserInstanceSpecs: optimalOrchestratorSpecs,
	})
	o.input = catalog.DownloadFromR2(defaultOrchestratorProvider)

	// we just want the cheapest one. Prices aren't loaded if the provider is aws.
	o.FilterUsingConfigOrch().FilterUsingMaxPrice()
	o.output = o.input
	return o.output
}

// Every optimization function should be adding instances to o.output at every stage.
func Optimize(jobFile *cedana.JobFile) []cedana.Instance {
	o := Build(jobFile)

	// pull catalogs from cloud storage
	o.logger.Info().Msg("loading catalogs for enabled providers: " + strings.Join(o.cfg.EnabledProviders, ","))
	o.LoadCatalogsFromR2()

	o.logger.Info().Msg("searching for optimal instance...")
	// for each provider, run provider-specific optimizers, finally filter by max price
	o.FilterUsingConfig()

	if o.jobFile.UserInstanceSpecs.VRAM != 0 || o.jobFile.UserInstanceSpecs.GPU != "" {
		o.FilterForGPUs()
	}

	o.FilterUsingMaxPrice().FilterByRegions(maxInstancesToConsider)

	if len(o.input) == 0 {
		o.logger.Fatal().Msg("No instances found that match your specs. Recommend increasing max price or reducing capacity requirements.")
	}

	displayOptimizerOutput(o.input[:len(o.input)/2])

	o.logger.Info().Msgf("found optimal instance...")
	// size of input is limited by region filter, no need to slice
	o.output = o.input
	return o.output
}

func Build(jobFile *cedana.JobFile) *Optimizer {
	logger := utils.GetLogger()
	cfg, err := utils.InitCedanaConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not initialize cedana config")
	}

	// match config specs to all available instance types, and then match to regions (?)
	// get subset of catalog and attach to optimizer

	return &Optimizer{
		input:   make([]cedana.Instance, 0),
		cfg:     cfg,
		logger:  &logger,
		ctx:     context.Background(),
		jobFile: jobFile,
	}
}

func (o *Optimizer) FilterUsingConfigOrch() *Optimizer {
	// we don't want to search the entire catalog for an optimizer,
	// so this is a separate function that limits the region filter

	// massive code-smell because it just repeats the below code lol
	var filtered []cedana.Instance
	for i := range o.input {
		switch o.input[i].Provider {
		case "aws":
			if o.jobFile.UserInstanceSpecs.Memory < int(o.input[i].MemoryGiB) &&
				o.jobFile.UserInstanceSpecs.VCPUs < int(o.input[i].VCPUs) &&
				isInstanceInRegion(o.input[i].Region, []string{o.cfg.AWSConfig.EnabledRegions[0]}) &&
				isInstancePartOfFamily(o.input[i].InstanceType, o.cfg.AWSConfig.EnabledInstanceFamilies) {
				filtered = append(filtered, o.input[i])
			}
		case "paperspace":
			if o.jobFile.UserInstanceSpecs.Memory < int(o.input[i].MemoryGiB) &&
				o.jobFile.UserInstanceSpecs.VCPUs < int(o.input[i].VCPUs) &&
				isInstanceInRegion(o.input[i].Region, o.cfg.PaperspaceConfig.EnabledRegions) {
				filtered = append(filtered, o.input[i])
			}
		}
	}

	o.input = filtered
	o.logger.Info().Msgf("filtered using config: %d instances", len(o.input))
	return o
}

func (o *Optimizer) FilterUsingConfig() *Optimizer {
	// TODO: This can absolutely be refactored into provider-specific optimizers.
	// Leave as is for now, will become unwieldly the more providers we add however.
	var filtered []cedana.Instance
	for i := range o.input {
		switch o.input[i].Provider {
		case "aws":
			if o.jobFile.UserInstanceSpecs.Memory < int(o.input[i].MemoryGiB) &&
				o.jobFile.UserInstanceSpecs.VCPUs < int(o.input[i].VCPUs) &&
				isInstanceInRegion(o.input[i].Region, o.cfg.AWSConfig.EnabledRegions) &&
				isInstancePartOfFamily(o.input[i].InstanceType, o.cfg.AWSConfig.EnabledInstanceFamilies) {
				filtered = append(filtered, o.input[i])
			}
		case "paperspace":
			if o.jobFile.UserInstanceSpecs.Memory < int(o.input[i].MemoryGiB) &&
				o.jobFile.UserInstanceSpecs.VCPUs < int(o.input[i].VCPUs) &&
				isInstanceInRegion(o.input[i].Region, o.cfg.PaperspaceConfig.EnabledRegions) {
				filtered = append(filtered, o.input[i])
			}
		}
	}

	o.input = filtered
	o.logger.Info().Msgf("filtered using config: %d instances", len(o.input))
	return o
}

// Only call if GPU is asked for, as this will return empty otherwise
func (o *Optimizer) FilterForGPUs() *Optimizer {
	// there's no provider-specific elements w/ this filter, so we can just blast through everything
	var filtered []cedana.Instance
	for i := range o.input {
		gpuinfo := o.input[i].GetGPUs()
		for _, g := range gpuinfo.Gpus {
			if o.jobFile.UserInstanceSpecs.GPU != "" {
				// if GPU name is set in the config, user expects exact match
				if o.jobFile.UserInstanceSpecs.GPU == g.Name {
					filtered = append(filtered, o.input[i])
				}
			}
		}
		// if GPU name isn't set, user only cares about total VRAM on instance
		if o.jobFile.UserInstanceSpecs.VRAM < int(gpuinfo.TotalGpuMemoryInMiB/1000) {
			filtered = append(filtered, o.input[i])
		}
	}

	if len(filtered) != 0 {
		o.input = filtered
		o.logger.Info().Msgf("filtered for GPUs: %d instances", len(o.input))
	}
	return o
}

func (o *Optimizer) FilterUsingMaxPrice() *Optimizer {
	// need to fetch realtime prices for aws
	// safe to assume we have aws instances kicking around the optimizer if it's in the provider list
	if slices.Contains(o.cfg.EnabledProviders, "aws") {
		// ensure aws instances have a price attached
		apm := pricing.GenAWSPricingModel()
		for i := range o.input {
			if o.input[i].Provider == "aws" {
				apm.GetPrice(&o.input[i])
			}
		}
	}

	// at this point, assume all prices are valid/set
	o.logger.Info().Msgf("filtering on configured max price of %f", o.jobFile.UserInstanceSpecs.MaxPrice)
	filtered := make([]cedana.Instance, 0)
	for i := range o.input {
		if o.input[i].Price <= o.jobFile.UserInstanceSpecs.MaxPrice {
			filtered = append(filtered, o.input[i])
		}
	}

	// sort
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Price < filtered[j].Price
	})

	o.input = filtered
	o.logger.Info().Msgf("filtered using max price: %d instances", len(o.input))

	return o
}

func (o *Optimizer) FilterByRegions(maxInstancesToConsider int) *Optimizer {
	filtered := make([]cedana.Instance, 0)
	// TODO - these are manual lol
	regions := len(o.cfg.AWSConfig.EnabledRegions) + len(o.cfg.PaperspaceConfig.EnabledRegions)
	maxPerRegion := maxInstancesToConsider / regions
	instancesByRegion := make(map[string]int)
	for i := range o.input {
		instancesByRegion[o.input[i].Region]++
		// we want to cap the number of instances per region
		if instancesByRegion[o.input[i].Region] <= maxPerRegion {
			filtered = append(filtered, o.input[i])
		}
		if len(filtered) == maxInstancesToConsider {
			break
		}
	}

	o.input = filtered
	return o
}

func (o *Optimizer) FilterAWSCapacity() *Optimizer {
	return o
}

func (o *Optimizer) CedanaBasicPriceOptimizer() *Optimizer {
	return o
}

// ignore display for now - variable instance since some fields can be empty
func displayOptimizerOutput(instances []cedana.Instance) {
	table := tablewriter.NewWriter(os.Stdout)
	for _, it := range instances {
		var manufacturer string
		var gpuName string
		var gpuCount int
		gpuinfo := it.GetGPUs()
		if gpuinfo.Gpus != nil {
			gpuName = gpuinfo.Gpus[0].Name
			gpuCount = gpuinfo.Gpus[0].Count
			manufacturer = gpuinfo.Gpus[0].Manufacturer
		}
		// TODO: assumption here is that GPUs are all the same (if there are multiple attached to the machine)
		table.SetHeader([]string{"Provider", "Accelerator Name", "VCPUs", "RAM (GB)", "GPU", "GPU Count", "Total VRAM (GB)", "Instance Type", "Region", "Price"})
		table.Append([]string{
			it.Provider,
			it.AcceleratorName,
			strconv.FormatFloat(it.VCPUs, 'f', -1, 64),
			strconv.FormatFloat(it.MemoryGiB, 'f', -1, 64),
			fmt.Sprintf("%s %s", manufacturer, gpuName),
			strconv.Itoa(gpuCount),
			strconv.Itoa(gpuinfo.TotalGpuMemoryInMiB / 1000),
			it.InstanceType,
			it.Region,
			strconv.FormatFloat(it.Price, 'f', 5, 64),
		})
	}

	table.Render()

}

func isInstancePartOfFamily(instanceType string, families []string) bool {
	for _, f := range families {
		if strings.Contains(instanceType, f) {
			return true
		}
	}
	return false
}

func isInstanceInRegion(instanceRegion string, regions []string) bool {
	for _, r := range regions {
		if instanceRegion == r {
			return true
		}
	}
	return false
}
