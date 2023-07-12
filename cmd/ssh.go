package cmd

import (
	"errors"
	"fmt"

	"github.com/cedana/cedana-client/db"
	"github.com/cedana/cedana-client/utils"
	"github.com/spf13/cobra"
)

var sshCommand = &cobra.Command{
	Use:   "ssh",
	Short: "SSH into instance",
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

		inst := db.GetInstanceByProviderId(id)
		if inst == nil {
			return errors.New("could not find instance with provided id")
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

		fmt.Printf("`ssh -o \"IdentitiesOnly=yes\" -i %s %s@%s`\n",
			sshKey,
			user,
			ipaddr,
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(sshCommand)
}
