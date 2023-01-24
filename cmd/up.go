package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/gcp"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	Snapshot bool
}

// NewUpCmd creates a new up command
func NewUpCmd() *cobra.Command {
	cmd := &UpCmd{}
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts a new workspace",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	upCmd.Flags().BoolVar(&cmd.Snapshot, "snapshot", false, "If true will create a snapshot for the environment")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context, _ []string) error {
	// TODO: remove hardcode
	provider := gcp.NewGCPProvider(log.Default)
	workspace := &types.Workspace{
		ID:         "test",
		Repository: "https://github.com/microsoft/vscode-course-sample",
	}

	// create environment
	err := provider.Apply(context.Background(), workspace, types.ApplyOptions{})
	if err != nil {
		return err
	}

	// start ssh
	handler, err := provider.RemoteCommandHost(ctx, workspace, types.RemoteCommandOptions{})
	if err != nil {
		return err
	}
	defer handler.Close()

	// install devpod
	err = installDevPod(handler)
	if err != nil {
		return err
	}

	// run devpod agent up
	log.Default.Infof("Creating devcontainer...")
	err = devPodAgentUp(handler, workspace)
	if err != nil {
		return err
	}

	// configure container ssh
	err = configureSSH(workspace.ID, "vscode")
	if err != nil {
		return err
	}
	log.Default.Infof("Run 'ssh %s.devpod' to ssh into the devcontainer", workspace.ID)

	// start VSCode
	log.Default.Infof("Starting VSCode...")
	err = exec.Command("code", "--folder-uri", fmt.Sprintf("vscode-remote://ssh-remote+%s.devpod/workspaces/%s", workspace.ID, workspace.ID)).Run()
	if err != nil {
		return err
	}

	// create snapshot
	if cmd.Snapshot {
		err = provider.ApplySnapshot(ctx, workspace, types.ApplySnapshotOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func devPodAgentUp(handler types.RemoteCommandHandler, workspace *types.Workspace) error {
	err := handler.Run(context.TODO(), fmt.Sprintf("%s agent up --id %s --repository %s", agent.RemoteDevPodHelperLocation, workspace.ID, workspace.Repository), nil, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	return nil
}
