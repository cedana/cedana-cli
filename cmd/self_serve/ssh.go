package self_serve

import (
	"fmt"

	"github.com/cedana/cedana-cli/db"
	"github.com/cedana/cedana-cli/utils"
	"github.com/spf13/cobra"
)

var tunnel string

var sshCommand = &cobra.Command{
	Use:   "ssh",
	Short: "SSH into instance using [cedana-id]",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		// retrieve ip from db
		db := db.NewDB()
		logger := utils.GetLogger()
		cfg, err := utils.InitCedanaConfig()
		if err != nil {
			logger.Fatal().Err(err).Msgf("could not set up config!")
		}

		inst := db.GetInstanceByCedanaID(id)
		if inst.CedanaID == "" {
			logger.Fatal().Msgf("could not find instance with id %s", id)
		}
		ipaddr := inst.IPAddress
		var sshKey string
		var user string

		switch inst.Provider {
		case "aws":
			sshKey = cfg.AWSConfig.SSHKeyPath
			user = cfg.AWSConfig.User
			if user == "" {
				user = "ubuntu"
			}
		case "paperspace":
			sshKey = cfg.PaperspaceConfig.SSHKeyPath
			user = cfg.PaperspaceConfig.User
			if user == "" {
				user = "paperspace"
			}
		}

		if tunnel != "" {
			fmt.Printf("ssh -o \"IdentitiesOnly=yes\" -i %s -L %s %s@%s\n",
				sshKey,
				tunnel,
				user,
				ipaddr,
			)
		} else {
			fmt.Printf("`ssh -o \"IdentitiesOnly=yes\" -i %s %s@%s`\n",
				sshKey,
				user,
				ipaddr,
			)
		}
		return nil
	},
}

func init() {
	runSelfServeCmd.AddCommand(sshCommand)
	sshCommand.Flags().StringVarP(&tunnel, "tunnel", "t", "", "tunnel FROM:TO (e.g 8080:localhost:4999)")
}
