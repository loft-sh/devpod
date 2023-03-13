package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// InitCmd holds the cmd flags
type InitCmd struct {
	*flags.GlobalFlags
}

// NewInitCmd creates a new command
func NewInitCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &AddCmd{
		GlobalFlags: flags,
	}
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes a provider in DevPod",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, args)
		},
	}

	return initCmd
}

func (cmd *InitCmd) Run(ctx context.Context, devPodConfig *config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please specify a provider name. E.g. devpod provider init local")
	}

	providerConfig, err := workspace.FindProvider(devPodConfig, args[0], log.Default)
	if err != nil {
		return err
	}

	writer := log.Default.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// init provider
	err = initProvider(ctx, devPodConfig, providerConfig.Config, writer, writer)
	if err != nil {
		return err
	}

	// save provider config
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	log.Default.Donef("Successfully initialized provider '%s'", providerConfig.Config.Name)
	return nil
}
