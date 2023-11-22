package self_serve

import (
	"github.com/cedana/cedana-cli/aws"
	"github.com/cedana/cedana-cli/utils"
	"github.com/spf13/cobra"
)

var copyRegion = &cobra.Command{
	Use:    "template",
	Short:  "Migrate a launch template from region A to region B",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO fix args
		logger := utils.GetLogger()
		_, err := aws.CopyTemplate(args[0], args[1], args[2], args[3], &logger)
		if err != nil {
			return err
		}
		return nil
	},
}

// stupid AWS returns everything as a pointer
func StringPtrToString(p *string) string {
	if p != nil {
		return *p
	}
	return "(nil)"
}

func init() {
	runSelfServeCmd.AddCommand(copyRegion)
}
