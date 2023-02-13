package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// CreateCmd holds the cmd flags
type CreateCmd struct{}

// NewCreateCmd defines a command
func NewCreateCmd() *cobra.Command {
	cmd := &CreateCmd{}
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			gcloudProvider, err := newProvider(log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), gcloudProvider, provider.FromEnvironment(), log.Default)
		},
	}

	return createCmd
}

// Run runs the command logic
func (cmd *CreateCmd) Run(ctx context.Context, provider *gcloudProvider, workspace *provider.Workspace, log log.Logger) error {
	name := getName(workspace)
	args := []string{
		"compute",
		"instances",
		"create",
		name,
		"--project=" + provider.Config.Project,
		"--zone=" + provider.Config.Zone,
		"--no-shielded-secure-boot",
	}

	// add machine type
	args = append(args, "--machine-type="+provider.Config.MachineType)

	// image & size
	args = append(args, "--create-disk")
	args = append(args, fmt.Sprintf("auto-delete=yes,boot=yes,device-name=%s,image=%s,mode=rw,size=%d,type=pd-ssd", name, provider.Config.DiskImage, provider.Config.DiskSizeGB))

	_, err := provider.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "create vm")
	}
	return nil
}
