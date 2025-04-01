package cmd

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/completion"
	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StopCmd holds the destroy cmd flags
type StopCmd struct {
	*flags.GlobalFlags
	client2.StopOptions
}

// NewStopCmd creates a new destroy command
func NewStopCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StopCmd{
		GlobalFlags: flags,
	}
	stopCmd := &cobra.Command{
		Use:     "stop [flags] [workspace-path|workspace-name]",
		Aliases: []string{"down"},
		Short:   "Stops an existing workspace",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			ctx := cobraCmd.Context()
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			err = clientimplementation.DecodePlatformOptionsFromEnv(&cmd.StopOptions.Platform)
			if err != nil {
				return fmt.Errorf("decode platform options: %w", err)
			}

			client, err := workspace2.Get(ctx, devPodConfig, args, false, cmd.Owner, false, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, client)
		},
		ValidArgsFunction: func(rootCmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completion.GetWorkspaceSuggestions(rootCmd, cmd.Context, cmd.Provider, args, toComplete, cmd.Owner, log.Default)
		},
	}

	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(ctx context.Context, devPodConfig *config.Config, client client2.BaseWorkspaceClient) error {
	// lock workspace
	if !cmd.Platform.Enabled {
		err := client.Lock(ctx)
		if err != nil {
			return err
		}
		defer client.Unlock()
	}

	// get instance status
	instanceStatus, err := client.Status(ctx, client2.StatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus != client2.StatusRunning {
		return fmt.Errorf("cannot stop workspace because it is '%s'", instanceStatus)
	}

	// stop if single machine provider
	wasStopped, err := cmd.stopSingleMachine(ctx, client, devPodConfig)
	if err != nil {
		return err
	} else if wasStopped {
		return nil
	}

	// stop environment
	err = client.Stop(ctx, client2.StopOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (cmd *StopCmd) stopSingleMachine(ctx context.Context, client client2.BaseWorkspaceClient, devPodConfig *config.Config) (bool, error) {
	// check if single machine
	singleMachineName := workspace2.SingleMachineName(devPodConfig, client.Provider(), log.Default)
	if !devPodConfig.Current().IsSingleMachine(client.Provider()) || client.WorkspaceConfig().Machine.ID != singleMachineName {
		return false, nil
	}

	// try to find other workspace with same machine
	workspaces, err := workspace2.List(ctx, devPodConfig, false, cmd.Owner, log.Default)
	if err != nil {
		return false, errors.Wrap(err, "list workspaces")
	}

	// loop workspaces
	foundOther := false
	for _, workspace := range workspaces {
		if workspace.ID == client.Workspace() || workspace.Machine.ID != singleMachineName {
			continue
		}

		foundOther = true
		break
	}
	if foundOther {
		return false, nil
	}

	// if we haven't found another workspace on this machine, delete the whole machine
	machineClient, err := workspace2.GetMachine(devPodConfig, []string{singleMachineName}, log.Default)
	if err != nil {
		return false, errors.Wrap(err, "get machine")
	}

	// stop the machine
	err = machineClient.Stop(ctx, client2.StopOptions{})
	if err != nil {
		return false, errors.Wrap(err, "delete machine")
	}

	log.Default.Donef("Successfully stopped workspace '%s'", client.Workspace())
	return true, nil
}
