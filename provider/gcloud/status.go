package gcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
)

type InstanceStatus struct {
	NetworkInterfaces []InstanceStatusNetworkInterface `json:"networkInterfaces,omitempty"`
	Status            string                           `json:"status,omitempty"`
}

type InstanceStatusNetworkInterface struct {
	AccessConfigs []InstanceStatusAccessConfig `json:"accessConfigs,omitempty"`
}

type InstanceStatusAccessConfig struct {
	NatIP string `json:"natIP,omitempty"`
}

func (g *gcloudProvider) Status(ctx context.Context, workspace *config.Workspace, options types.StatusOptions) (types.Status, error) {
	name := getName(workspace)
	status, err := g.getWorkspaceStatus(ctx, name)
	if err != nil {
		return "", err
	}

	return g.statusFromInstanceStatus(status)
}

func (g *gcloudProvider) statusFromInstanceStatus(status *InstanceStatus) (types.Status, error) {
	if status == nil {
		return types.StatusNotFound, nil
	}

	if status.Status == "RUNNING" {
		return types.StatusRunning, nil
	} else if status.Status == "STOPPING" || status.Status == "SUSPENDING" || status.Status == "REPAIRING" || status.Status == "PROVISIONING" || status.Status == "STAGING" {
		return types.StatusBusy, nil
	} else if status.Status == "TERMINATED" {
		return types.StatusStopped, nil
	}

	return types.StatusNotFound, fmt.Errorf("unexpected status: %v", status.Status)
}

func (g *gcloudProvider) getWorkspaceStatus(ctx context.Context, name string) (*InstanceStatus, error) {
	args := []string{
		"compute",
		"instances",
		"list",
		"--format=json",
		"--filter=name:" + name,
	}

	out, err := g.output(ctx, args...)
	if err != nil {
		return nil, err
	}

	instances := []InstanceStatus{}
	err = json.Unmarshal(out, &instances)
	if err != nil {
		return nil, err
	} else if len(instances) == 0 {
		return nil, nil
	}

	return &instances[0], nil
}
