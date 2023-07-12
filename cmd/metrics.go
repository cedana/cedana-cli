package cmd

import (
	"os"
	"strconv"
	"time"

	"github.com/cedana/cedana-cli/db"
	"github.com/cedana/cedana-cli/types"
	"github.com/cedana/cedana-cli/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

/**
Develops pricing metrics for Cedana users. Right now, we support just
metrics based on historic Cedana usage. We eventually want to show cost savings as well though
so ideally we're pulling in information about reserved instance pricing and comparing it to show
total cost savings.
*/

var showYtdMetrics bool

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Pricing metrics on Cedana usage",
	Run: func(cmd *cobra.Command, args []string) {
		logger := utils.GetLogger()
		db := db.NewDB()
		m := &Metrics{
			db:     db,
			logger: &logger,
		}

		if showYtdMetrics {
			m.prettyPrintYTDMetrics()
		}

		m.prettyPrintRunningMetrics()
	},
}

type Metrics struct {
	db     *db.DB
	logger *zerolog.Logger
}

// pricing by running instances
func (m *Metrics) prettyPrintRunningMetrics() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Instance ID", "Provider", "Region", "State", "Price/hr ($)", "Total Running Cost ($)"})

	instances := m.db.GetAllRunningInstances()
	for _, instance := range instances {
		hoursSinceInstantiation := time.Since(instance.CreatedAt).Hours()
		totalCost := instance.Price * hoursSinceInstantiation

		table.Append([]string{
			instance.AllocatedID,
			instance.Provider,
			instance.Region,
			instance.State,
			strconv.FormatFloat(instance.Price, 'f', 3, 64),
			strconv.FormatFloat(totalCost, 'f', 3, 64),
		})
	}

	table.Render()
}

// pricing by provider
func (m *Metrics) prettyPrintYTDMetrics() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Provider", "Total Cost YTD ($)"})

	for _, p := range types.ProviderNames {
		instances := m.db.GetAllInstancesByProvider(p)

		totalCost := 0.0

		for _, instance := range instances {
			if instance.State == "destroyed" {
				totalHours := instance.DeletedAt.Time.Hour() - instance.CreatedAt.Hour()
				totalCost += instance.Price * float64(totalHours)
			} else {
				// assume instance is running
				hoursRun := time.Since(instance.CreatedAt).Hours()
				totalCost += instance.Price * float64(hoursRun)
			}
		}

		table.Append([]string{
			p,
			strconv.FormatFloat(totalCost, 'f', 2, 64),
		})
	}

	table.Render()
}

func init() {
	metricsCmd.Flags().BoolVarP(&showYtdMetrics, "ytd", "y", false, "Show YTD metrics")
	rootCmd.AddCommand(metricsCmd)
}
