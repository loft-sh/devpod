package cmd

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/template"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/devpod/scripts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

//go:embed cloud-config.yaml.tpl
var CloudConfig string

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

	// get token
	t, err := token.GenerateWorkspaceToken(workspace.Context, workspace.ID)
	if err != nil {
		return err
	}

	// fill init script
	initScript, err := template.FillTemplate(scripts.InstallDevPodTemplate, map[string]string{
		"BaseUrl": agent.DefaultAgentDownloadURL,
		"Token":   t,
	})
	if err != nil {
		return err
	}

	// add cloud config
	cloudConfig, err := template.FillTemplate(CloudConfig, map[string]string{
		"InitScript": base64.StdEncoding.EncodeToString([]byte(initScript)),
	})
	if err != nil {
		return err
	}
	args = append(args, "--metadata", "user-data="+cloudConfig)

	// add machine type
	args = append(args, "--machine-type="+withDefault(provider.Config.MachineType, "e2-standard-2"))

	// image & size
	image := withDefault(provider.Config.DiskImage, "projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20230112")
	size := withDefault(provider.Config.DiskSizeGB, 30)
	args = append(args, "--create-disk")
	args = append(args, fmt.Sprintf("auto-delete=yes,boot=yes,device-name=%s,image=%s,mode=rw,size=%d,type=pd-ssd", name, image, size))

	// network
	args = append(args, "--network-interface=network-tier=PREMIUM,subnet=default")

	log.Infof("Creating VM Instance %s...", name)
	_, err = provider.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "create vm")
	}

	provider.Log.Infof("Successfully created VM instance %s", name)
	return nil
}
