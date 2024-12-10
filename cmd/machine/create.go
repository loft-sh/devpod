package machine

import (
	"context"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// CreateCmd holds the configuration
type CreateCmd struct {
	*flags.GlobalFlags

	ProviderOptions []string
}

// NewCreateCmd creates a new create command
func NewCreateCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: flags,
	}
	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Creates a new machine",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}
	createCmd.Flags().StringSliceVar(&cmd.ProviderOptions, "provider-option", []string{}, "Provider option in the form KEY=VALUE")
	return createCmd
}

// Run runs the command logic
func (cmd *CreateCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	machineClient, err := workspace.ResolveMachine(devPodConfig, args, cmd.ProviderOptions, log.Default)
	if err != nil {
		return err
	}

	err = machineClient.Create(ctx, client.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}
