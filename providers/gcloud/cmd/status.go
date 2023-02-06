package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
	"os"
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

// StatusCmd holds the cmd flags
type StatusCmd struct{}

// NewStatusCmd defines a command
func NewStatusCmd() *cobra.Command {
	cmd := &StatusCmd{}
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Status an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			gcloudProvider, err := newProvider(log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), gcloudProvider, provider.FromEnvironment(), log.Default)
		},
	}

	return statusCmd
}

// Run runs the command logic
func (cmd *StatusCmd) Run(ctx context.Context, provider *gcloudProvider, workspace *provider.Workspace, log log.Logger) error {
	name := getName(workspace)
	status, err := getWorkspaceStatus(ctx, name, provider)
	if err != nil {
		return err
	}

	instanceStatus, err := cmd.statusFromInstanceStatus(status)
	if err != nil {
		return err
	}

	_, _ = os.Stdout.Write([]byte(instanceStatus))
	return nil
}

func (cmd *StatusCmd) statusFromInstanceStatus(status *InstanceStatus) (provider.Status, error) {
	if status == nil {
		return provider.StatusNotFound, nil
	}

	if status.Status == "RUNNING" {
		return provider.StatusRunning, nil
	} else if status.Status == "STOPPING" || status.Status == "SUSPENDING" || status.Status == "REPAIRING" || status.Status == "PROVISIONING" || status.Status == "STAGING" {
		return provider.StatusBusy, nil
	} else if status.Status == "TERMINATED" {
		return provider.StatusStopped, nil
	}

	return provider.StatusNotFound, fmt.Errorf("unexpected status: %v", status.Status)
}

func getWorkspaceStatus(ctx context.Context, name string, provider *gcloudProvider) (*InstanceStatus, error) {
	args := []string{
		"compute",
		"instances",
		"list",
		"--format=json",
		"--filter=name:" + name,
	}

	out, err := provider.output(ctx, args...)
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
