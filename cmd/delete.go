package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"time"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	GracePeriod string
	Force       bool
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes an existing workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, args)
		},
	}

	deleteCmd.Flags().StringVar(&cmd.GracePeriod, "grace-period", "", "The amount of time to give the command to delete the workspace")
	deleteCmd.Flags().BoolVar(&cmd.Force, "force", false, "Delete workspace even if it is not found remotely anymore")
	return deleteCmd
}

// Run runs the command logic
func (cmd *DeleteCmd) Run(ctx context.Context, devPodConfig *config.Config, args []string) error {
	// try to load workspace
	client, err := workspace2.GetWorkspace(devPodConfig, nil, args, false, log.Default)
	if err != nil {
		if !cmd.Force {
			log.Default.Errorf("cannot delete workspace because there was an error loading the workspace. Run with --force to ignore this error")
			return err
		} else if len(args) == 0 {
			return fmt.Errorf("cannot delete workspace because there was an error loading the workspace: %v. Please specify the id of the workspace you want to delete. E.g. 'devpod delete my-workspace --force'", err)
		}

		workspaceID := workspace2.Exists(devPodConfig, args)
		if workspaceID == "" {
			return fmt.Errorf("couldn't find workspace %s", args[0])
		}

		// print error
		log.Default.Errorf("Error retrieving workspace: %v", err)

		// delete workspace folder
		err = clientimplementation.DeleteWorkspaceFolder(devPodConfig.DefaultContext, workspaceID)
		if err != nil {
			return err
		}

		log.Default.Donef("Successfully deleted workspace '%s'", workspaceID)
		return nil
	}

	// get instance status
	if !cmd.Force {
		instanceStatus, err := client.Status(ctx, client2.StatusOptions{})
		if err != nil {
			return err
		} else if instanceStatus == client2.StatusNotFound {
			return fmt.Errorf("cannot delete workspace because it couldn't be found. Run with --force to ignore this error")
		}
	}

	var duration *time.Duration
	if cmd.GracePeriod != "" {
		gracePeriod, err := time.ParseDuration(cmd.GracePeriod)
		if err != nil {
			return errors.Wrap(err, "parse grace-period")
		}

		duration = &gracePeriod
	}

	// destroy environment
	err = client.Delete(ctx, client2.DeleteOptions{
		Force:       cmd.Force,
		GracePeriod: duration,
	})
	if err != nil {
		return err
	}

	log.Default.Donef("Successfully deleted workspace '%s'", client.Workspace())
	return nil
}
