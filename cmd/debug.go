package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cedana/cedana-client/market/catalog"
	"github.com/cedana/cedana-client/types"
	"github.com/cedana/cedana-client/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var debugCmd = &cobra.Command{
	Use:    "debug",
	Short:  "Functions/tools for debugging instances or testing new components",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("run debug with one of its subcommands")
	},
}

var cfgCmd = &cobra.Command{
	Use: "config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.InitCedanaConfig()
		if err != nil {
			return err
		}
		fmt.Sprintf("config file used: %s", viper.GetViper().ConfigFileUsed())
		// pretty print config for debugging to make sure it's been loaded correctly
		prettyCfg, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(prettyCfg))
		return nil
	},
}

var envCmd = &cobra.Command{
	Use: "env",
	RunE: func(cmd *cobra.Command, args []string) error {
		// we get env vars from the os - sometimes useful to know what they're assigned to
		fmt.Println(os.Getenv("CEDANA_ORCH_ID"))
		fmt.Println(os.Getenv("CEDANA_JOB_ID"))
		return nil
	},
}

var buildConfigCmd = &cobra.Command{
	Use:   "build client config",
	Short: "Command for directly building server overrides",
	RunE: func(cmd *cobra.Command, args []string) error {

		client_config := utils.BuildClientConfig(nil)
		cc_marshaled, _ := json.Marshal(client_config)

		err := os.WriteFile("/home/nravichandra/.cedana/server_overrides.json", cc_marshaled, 0o644)
		if err != nil {
			return fmt.Errorf("could not marshal overrides %v", err)
		}

		return nil
	},
}

var parseAndUploadToR2Cmd = &cobra.Command{
	Use:   "upload",
	Short: "Workaround for directly uploading provider catalogs to R2",
	RunE: func(cmd *cobra.Command, args []string) error {
		filepath := args[0]
		// just push
		catalog.UploadToR2(filepath)
		return nil
	},
}

var generateCatalogCmd = &cobra.Command{
	Use:   "gen-catalog",
	Short: "Workaround for directly generating provider catalogs",
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := args[0]
		switch provider {
		case "aws":
			catalog.ParseAWSCatalog()
		case "paperspace":
			catalog.ParsePaperspaceCatalog()
		}
		return nil
	},
}

var downloadCatalogCmd = &cobra.Command{
	Use:   "download",
	Short: "Workaround for directly downloading provider catalogs",
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := args[0]
		instances := catalog.DownloadFromR2(provider)

		// marshal it for pretty-print
		b, _ := json.MarshalIndent(instances, "", "  ")

		fmt.Printf("%s\n", string(b))
		return nil
	},
}

var setupTestCmd = &cobra.Command{
	Use:   "setup_test",
	Short: "setup nats for a test with jobId ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jid := args[0]
		r := buildRunner()

		// create fake job
		r.job = &types.Job{
			JobID: jid,
		}
		r.SetupNATSForJob()
		return nil
	},
}

var unzipCmd = &cobra.Command{
	Use:  "unzip",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		folder := args[1]
		err := utils.Unzip(file, folder)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)
	debugCmd.AddCommand(buildConfigCmd)
	debugCmd.AddCommand(parseAndUploadToR2Cmd)
	debugCmd.AddCommand(generateCatalogCmd)
	debugCmd.AddCommand(downloadCatalogCmd)
	debugCmd.AddCommand(cfgCmd)
	debugCmd.AddCommand(envCmd)
	debugCmd.AddCommand(setupTestCmd)
	debugCmd.AddCommand(unzipCmd)
}
