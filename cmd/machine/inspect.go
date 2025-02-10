package machine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

type InspectCmd struct {
	*flags.GlobalFlags
}

func NewInspectCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &InspectCmd{
		GlobalFlags: flags,
	}
	stopCmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspects an existing machine",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return stopCmd
}

func (cmd *InspectCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	machineClient, err := workspace.GetMachine(devPodConfig, args, log.Default)
	if err != nil {
		return err
	}
	p, err := provider.LoadProviderConfig(devPodConfig.DefaultContext, machineClient.Provider())
	if err != nil {
		return err
	}

	machineConfig := machineClient.MachineConfig()
	for k := range machineConfig.Provider.Options {
		optConfig := p.Options[k]
		if optConfig.Hidden {
			delete(machineConfig.Provider.Options, k)
			continue
		}

		if optConfig.Password {
			opt := machineConfig.Provider.Options[k]
			opt.Value = "********"
		}
	}

	out, err := json.MarshalIndent(machineConfig, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))

	return nil
}
