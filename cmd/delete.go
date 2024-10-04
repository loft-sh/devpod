package cmd

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags
	client2.DeleteOptions
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete [flags] [workspace-path|workspace-name]",
		Short: "Deletes an existing workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			_, err := clientimplementation.DecodeOptionsFromEnv(clientimplementation.DevPodFlagsDelete, &cmd.DeleteOptions)
			if err != nil {
				return fmt.Errorf("decode up options: %w", err)
			}

			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, args)
		},
	}

	deleteCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "Treat \"workspace not found\" as a successful delete")
	deleteCmd.Flags().StringVar(&cmd.GracePeriod, "grace-period", "", "The amount of time to give the command to delete the workspace")
	deleteCmd.Flags().BoolVar(&cmd.Force, "force", false, "Delete workspace even if it is not found remotely anymore")
	return deleteCmd
}

// Run runs the command logic
func (cmd *DeleteCmd) Run(ctx context.Context, devPodConfig *config.Config, args []string) error {
	// try to load workspace
	client, err := workspace2.GetWorkspace(devPodConfig, args, false, log.Default)
	if err != nil {
		if len(args) == 0 {
			return fmt.Errorf("cannot delete workspace because there was an error loading the workspace: %w. Please specify the id of the workspace you want to delete. E.g. 'devpod delete my-workspace --force'", err)
		}

		workspaceID := workspace2.Exists(devPodConfig, args)
		if workspaceID == "" {
			if cmd.IgnoreNotFound {
				return nil
			}

			return fmt.Errorf("couldn't find workspace %s", args[0])
		} else if !cmd.Force {
			log.Default.Errorf("cannot delete workspace because there was an error loading the workspace. Run with --force to ignore this error")
			return err
		}

		// print error
		log.Default.Errorf("Error retrieving workspace: %v", err)

		// delete workspace folder
		err = clientimplementation.DeleteWorkspaceFolder(devPodConfig.DefaultContext, workspaceID, "", log.Default)
		if err != nil {
			return err
		}

		log.Default.Donef("Successfully deleted workspace '%s'", workspaceID)
		return nil
	}

	// skip deletion if imported
	workspaceConfig := client.WorkspaceConfig()
	if !cmd.Force && workspaceConfig.Imported {
		// delete workspace folder
		err = clientimplementation.DeleteWorkspaceFolder(devPodConfig.DefaultContext, client.Workspace(), workspaceConfig.SSHConfigPath, log.Default)
		if err != nil {
			return err
		}

		log.Default.Donef("Skip remote deletion of workspace %s as it is imported, if you really want to delete this workspace also remotely, run with --force", client.Workspace())
		return nil
	}

	// get instance status
	if !cmd.Force {
		// lock workspace only if we don't force deletion
		err := client.Lock(ctx)
		if err != nil {
			return err
		}
		defer client.Unlock()

		// retrieve instance status
		instanceStatus, err := client.Status(ctx, client2.StatusOptions{})
		if err != nil {
			return err
		} else if instanceStatus == client2.StatusNotFound {
			return fmt.Errorf("cannot delete workspace because it couldn't be found. Run with --force to ignore this error")
		}
	}

	// delete if single machine provider
	wasDeleted, err := cmd.deleteSingleMachine(ctx, client, devPodConfig)
	if err != nil {
		return err
	} else if wasDeleted {
		return nil
	}

	// destroy environment
	err = client.Delete(ctx, cmd.DeleteOptions)
	if err != nil {
		return err
	}

	log.Default.Donef("Successfully deleted workspace '%s'", client.Workspace())
	return nil
}

func (cmd *DeleteCmd) deleteSingleMachine(ctx context.Context, client client2.BaseWorkspaceClient, devPodConfig *config.Config) (bool, error) {
	// check if single machine
	singleMachineName := workspace2.SingleMachineName(devPodConfig, client.Provider(), log.Default)
	if !devPodConfig.Current().IsSingleMachine(client.Provider()) || client.WorkspaceConfig().Machine.ID != singleMachineName {
		return false, nil
	}

	// try to find other workspace with same machine
	workspaces, err := workspace2.ListWorkspaces(devPodConfig, log.Default)
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

	// delete the machine
	err = machineClient.Delete(ctx, cmd.DeleteOptions)
	if err != nil {
		return false, errors.Wrap(err, "delete machine")
	}

	// delete workspace folder
	err = clientimplementation.DeleteWorkspaceFolder(client.Context(), client.Workspace(), client.WorkspaceConfig().SSHConfigPath, log.Default)
	if err != nil {
		return false, err
	}

	log.Default.Donef("Successfully deleted workspace '%s'", client.Workspace())
	return true, nil
}
