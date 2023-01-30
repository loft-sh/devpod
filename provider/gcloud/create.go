package gcloud

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/loft-sh/devpod/pkg/template"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/devpod/scripts"
	"github.com/pkg/errors"
)

//go:embed cloud-config.yaml.tpl
var CloudConfig string

func (g *gcloudProvider) Create(ctx context.Context, workspace *config.Workspace, options types.CreateOptions) error {
	name := getName(workspace)
	args := []string{
		"compute",
		"instances",
		"create",
		name,
		"--project=" + g.Config.Project,
		"--zone=" + g.Config.Zone,
		"--no-shielded-secure-boot",
	}

	// get token
	t, err := token.GenerateWorkspaceToken(workspace.ID)
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
	args = append(args, "--machine-type="+withDefault(g.Config.MachineType, "e2-standard-2"))

	// image & size
	image := withDefault(g.Config.DiskImage, "projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20230112")
	size := withDefault(g.Config.DiskSizeGB, 30)
	args = append(args, "--create-disk")
	args = append(args, fmt.Sprintf("auto-delete=yes,boot=yes,device-name=%s,image=%s,mode=rw,size=%d,type=pd-ssd", name, image, size))

	// network
	args = append(args, "--network-interface=network-tier=PREMIUM,subnet=default")

	// extra args
	args = append(args, g.Config.CreateExtraArgs...)

	g.Log.Infof("Creating VM Instance %s...", name)
	_, err = g.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "create vm")
	}

	g.Log.Infof("Successfully created VM instance %s", name)
	return nil
}

func getName(workspace *config.Workspace) string {
	return "devpod-" + workspace.ID
}

func withDefault[V int | string](val V, other V) V {
	var t V
	if val == t {
		return other
	}
	return val
}
