package self_serve

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cedana/cedana-cli/market"
	"github.com/cedana/cedana-cli/utils"
	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Bootstrap cedana-cli
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Setup host for cedana usage",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		logger := utils.GetLogger()
		b := &Bootstrap{
			l:   &logger,
			ctx: ctx,
		}

		// TODO: should take cues for other bootstraps
		err := b.createConfig()
		if err != nil {
			logger.Fatal().Err(err).Msg("could not create config")
		}
		err = b.getProviders()
		if err != nil {
			logger.Fatal().Err(err).Msg("could not assign providers in config")
		}

		return nil
	},
}

type Bootstrap struct {
	l   *zerolog.Logger
	ctx context.Context
}

func (b *Bootstrap) createConfig() error {
	homeDir := os.Getenv("HOME")
	configFolderPath := filepath.Join(homeDir, ".cedana")
	// check that $HOME/.cedana folder exists - create if it doesn't
	_, err := os.Stat(configFolderPath)
	if err != nil {
		b.l.Info().Msg("config folder doesn't exist, creating...")
		err = os.Mkdir(configFolderPath, 0o755)
		if err != nil {
			b.l.Fatal().Err(err).Msg("could not create config folder")
		}
	}

	b.l.Info().Msg("checking for config...")
	_, err = os.OpenFile(filepath.Join(homeDir, "/.cedana/cedana_config.json"), 0, 0o644)
	if errors.Is(err, os.ErrNotExist) {
		b.l.Info().Msg("cedana_config.json does not exist, creating template")
		// copy template, use viper to set programatically
		err = utils.CreateCedanaConfig(filepath.Join(configFolderPath, "cedana_config.json"))
		if err != nil {
			b.l.Fatal().Err(err).Msg("could not create cedana_config")
		}
	}
	return nil
}

// sets enabled providers
func (b *Bootstrap) getProviders() error {
	available := []*item{
		{
			ID: "aws",
		},
		{
			ID: "paperspace",
		},
	}
	selected, err := selectItems(0, available)
	if err != nil {
		b.l.Fatal().Err(err).Msg("no providers selected")
	}
	var providers []string
	for _, i := range selected {
		providers = append(providers, i.ID)
	}

	_, err = utils.InitCedanaConfig()
	if err != nil {
		b.l.Fatal().Err(err).Msg("error initializing config")
	}

	viper.Set("enabled_providers", providers)
	err = viper.WriteConfig()
	if err != nil {
		b.l.Fatal().Err(err).Msg("error writing config")
		return err
	}

	for _, provider := range providers {
		if provider == "aws" {
			b.AWSBootstrap()
		}
	}

	return nil
}

// we should really be setting by just overwriting the config?
func (b *Bootstrap) AWSBootstrap() {
	c, err := utils.InitCedanaConfig()
	if err != nil {
		b.l.Fatal().Err(err).Msg("error initializing config")
	}

	// check that the regions are set
	if len(c.AWSConfig.EnabledRegions) == 0 {
		b.l.Info().Msg("No regions declared in config!")
		prompt := promptui.Prompt{
			Label: "Enter comma-separated aws regions you would like cedana to operate with",
		}
		result, err := prompt.Run()
		if err != nil {
			b.l.Fatal().Err(err).Msg("error reading prompt input")
		}
		regions := strings.Split(result, ",")
		viper.Set("aws.enabled_regions", regions)
		err = viper.WriteConfig()
		if err != nil {
			b.l.Fatal().Err(err).Msg("error writing config")
		}
	}

	// .aws/credentials check
	// can't proceed w/ invalid creds
	b.l.Info().Msg("checking for local aws credentials in env and ~/.aws/credentials..")
	_, err = market.MakeClient(aws.String("us-east-1"), b.ctx)
	if err != nil {
		b.l.Fatal().Err(err).Msg("Could not find credentials in env vars or shared configuration folder. Follow instructions here to set them up for your AWS account: https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/setup-credentials.html ")
	}
	b.l.Info().Msg("aws credentials found!")

	// check and set key file for ssh access.
	b.l.Info().Msg("checking for .pem key file for ssh access to instances...")
	// keep going if aws key is set in config
	var keyName string
	if c.AWSConfig.SSHKeyPath == "" {
		b.l.Info().Msg("no key file found in config!")
		keyName = b.promptAWSKey()
	}

	// check for launch template
	b.l.Info().Msg("checking for valid launch template...")
	if c.AWSConfig.LaunchTemplateName == "" {
		b.l.Info().Msg("Launch template not found, creating...")
		b.promptLaunchTemplateCreation(keyName)
	}
}

func (b *Bootstrap) promptLaunchTemplateCreation(keyName string) {
	c, err := utils.InitCedanaConfig()
	if err != nil {
		b.l.Fatal().Err(err).Msg("error initializing config")
	}
	prompt := promptui.Select{
		Label: "Do you want to create Cedana launch templates for all configured regions? [Y/n]",
		Items: []string{"Y", "n"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		b.l.Fatal().Err(err).Msg("error reading prompt")
	}

	if result == "Y" {
		// get regions
		if keyName != "" {
			regions := c.AWSConfig.EnabledRegions
			b.createLaunchTemplates(regions, keyName)
		} else {
			b.l.Fatal().Msg("invalid keyname found in config")
		}
	}
}

func (b *Bootstrap) promptAWSKey() string {
	var keyName string
	c, err := utils.InitCedanaConfig()
	if err != nil {
		b.l.Fatal().Err(err).Msg("error initializing config")
	}

	prompt := promptui.Select{
		Label: "Do you have a valid key file for ec2 instance ssh access? [Y/n]",
		Items: []string{"Y", "n"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		b.l.Fatal().Err(err).Msg("error reading prompt")
	}

	if result == "Y" {
		// ask for location and set.
		// TODO: validation function here
		prompt := promptui.Prompt{
			Label: "Enter path for key file",
		}
		r, err := prompt.Run()
		if err != nil {
			b.l.Fatal().Err(err).Msg("error reading prompt")
		}
		viper.Set("aws.ssh_key_path", r)
		err = viper.WriteConfig()
		if err != nil {
			b.l.Fatal().Err(err).Msg("could not write cedana config to file")
		}
		b.l.Info().Msg("wrote key path to config")

		// get keyName from key path
		base := filepath.Base(r)
		keyName = base[0 : len(base)-len(filepath.Ext(r))]
		return keyName
	}
	if result == "n" {
		prompt := promptui.Select{
			Label: "create one from credentials? [Y/n]",
			Items: []string{"Y", "n"},
		}
		_, result, err := prompt.Run()
		if err != nil {
			b.l.Fatal().Err(err).Msg("error reading prompt")
		}
		if result == "Y" {
			b.l.Info().Msgf("creating keys for all regions specified in config")
			for _, region := range c.AWSConfig.EnabledRegions {
				b.l.Info().Msgf("creating key for region %s", region)
				keyName = b.CreateAWSKeyFile(region)
				return keyName
			}
		}
		if result == "n" {
			b.l.Info().Msg("follow these instructions to create your own keyfile: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html")
		}
	}

	return ""
}

func (b *Bootstrap) GCPBootstrap() {

}

func (b *Bootstrap) AzureBootstrap() {

}

func (b *Bootstrap) CreateAWSKeyFile(region string) string {
	keyName := "cedana-ssh"
	client, err := market.MakeClient(aws.String(region), b.ctx)
	if err != nil {
		b.l.Fatal().Err(err).Msg("error creating aws client")
	}
	// TODO: should check for existing key w/ this string and bubble up an error if it exists.
	// Also add a delete key function
	out, err := client.CreateKeyPair(b.ctx, &ec2.CreateKeyPairInput{
		KeyName: aws.String(keyName),
	})
	if err != nil {
		b.l.Fatal().Err(err).Msg("error creating key file")
	}

	// save key to .cedana
	keyPath := filepath.Join(os.Getenv("HOME"), ".cedana", strings.Join([]string{keyName, "pem"}, "."))
	err = os.WriteFile(keyPath, []byte(*out.KeyMaterial), 0600)
	if err != nil {
		b.l.Fatal().Err(err).Msg("error writing keyfile to disk")
	}

	// write to config file
	viper.Set("aws.ssh_key_path", keyPath)
	err = viper.WriteConfig()
	if err != nil {
		b.l.Fatal().Err(err).Msg("could not write keyfile path to config")
	}

	return keyName
}

type item struct {
	ID         string
	IsSelected bool
}

// selectItems() prompts user to select one or more items in the given slice
func selectItems(selectedPos int, allItems []*item) ([]*item, error) {
	// Always prepend a "Done" item to the slice if it doesn't
	// already exist.
	const doneID = "Done"
	if len(allItems) > 0 && allItems[0].ID != doneID {
		var items = []*item{
			{
				ID: doneID,
			},
		}
		allItems = append(items, allItems...)
	}

	// Define promptui template
	templates := &promptui.SelectTemplates{
		Label: `{{if .IsSelected}}
                    ✔
                {{end}} {{ .ID }} - label`,
		Active:   "→ {{if .IsSelected}}✔ {{end}}{{ .ID | cyan }}",
		Inactive: "{{if .IsSelected}}✔ {{end}}{{ .ID | cyan }}",
	}

	prompt := promptui.Select{
		Label:     "Item",
		Items:     allItems,
		Templates: templates,
		Size:      5,
		// Start the cursor at the currently selected index
		CursorPos:    selectedPos,
		HideSelected: true,
	}

	selectionIdx, _, err := prompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	chosenItem := allItems[selectionIdx]

	if chosenItem.ID != doneID {
		// If the user selected something other than "Done",
		// toggle selection on this item and run the function again.
		chosenItem.IsSelected = !chosenItem.IsSelected
		return selectItems(selectionIdx, allItems)
	}

	// If the user selected the "Done" item, return
	// all selected items.
	var selectedItems []*item
	for _, i := range allItems {
		if i.IsSelected {
			selectedItems = append(selectedItems, i)
		}
	}
	return selectedItems, nil
}

// creates a launch template in the user account in set of given regions
// assume key is passed around (for now!)
// with authentication and user support, we should be able to centralize some of this logic
func (b *Bootstrap) createLaunchTemplates(regions []string, keyName string) error {
	for _, region := range regions {
		client, err := market.MakeClient(aws.String(region), b.ctx)
		if err != nil {
			return err
		}

		// check if a security group w/ name "cedana" exists
		describeSecurityGroupOut, err := client.DescribeSecurityGroups(b.ctx, &ec2.DescribeSecurityGroupsInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("group-name"),
					Values: []string{"cedana"},
				},
			},
		})

		if err != nil {
			b.l.Fatal().Err(err).Msg("error describing security group")
			return err
		}

		var sgID string
		// create security group if we couldn't find a cedana one
		if len(describeSecurityGroupOut.SecurityGroups) == 0 {
			// create a security group indiscriminately for ssh access
			out, err := client.CreateSecurityGroup(b.ctx, &ec2.CreateSecurityGroupInput{
				Description: aws.String("cedana security group for SSH access"),
				GroupName:   aws.String("cedana"),
			})
			if err != nil {
				b.l.Fatal().Err(err).Msg("error creating security group")
				return err
			}
			b.l.Info().Msgf("created security group %s", *out.GroupId)
			sgID := *out.GroupId

			// authorize ingress
			_, err = client.AuthorizeSecurityGroupIngress(b.ctx, &ec2.AuthorizeSecurityGroupIngressInput{
				GroupId:    aws.String(sgID),
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				CidrIp:     aws.String("0.0.0.0/0"),
			})

			if err != nil {
				b.l.Fatal().Err(err).Msg("error creating ssh rules for security group")
				return err
			}
		} else {
			sgID = *describeSecurityGroupOut.SecurityGroups[0].GroupId
		}

		// create launch template
		output, err := client.CreateLaunchTemplate(b.ctx, &ec2.CreateLaunchTemplateInput{
			LaunchTemplateName: aws.String("cedana-base"),
			LaunchTemplateData: &types.RequestLaunchTemplateData{
				KeyName: aws.String(keyName),
				SecurityGroupIds: []string{
					sgID,
				},
			},
		})

		if err != nil {
			b.l.Fatal().Err(err).Msg("error creating launch template")
			return err
		}
		b.l.Info().Msgf("created launch template %s", *output.LaunchTemplate.LaunchTemplateId)
	}

	// write launch template name to config
	_, err := utils.InitCedanaConfig()
	if err != nil {
		return err
	}

	viper.Set("aws.launch_template_name", "cedana-base")
	err = viper.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

func init() {
	runSelfServeCmd.AddCommand(bootstrapCmd)
}
