package pro

import (
	"context"
	"fmt"
	"os"
	"sync"

	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	providercmd "github.com/loft-sh/devpod/cmd/provider"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	platformdaemon "github.com/loft-sh/devpod/pkg/daemon/platform"
	"github.com/loft-sh/devpod/pkg/provider"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*proflags.GlobalFlags

	IgnoreNotFound bool
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *proflags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete or logout from a DevPod Pro Instance",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	deleteCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "Treat \"pro instance not found\" as a successful delete")
	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please specify an pro instance to delete")
	}

	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	// load pro instance config
	proInstanceName := args[0]
	proInstanceConfig, err := provider2.LoadProInstanceConfig(devPodConfig.DefaultContext, proInstanceName)
	if err != nil {
		if os.IsNotExist(err) && cmd.IgnoreNotFound {
			return nil
		}

		return fmt.Errorf("load pro instance %s: %w", proInstanceName, err)
	}

	providerConfig, err := provider.LoadProviderConfig(devPodConfig.DefaultContext, proInstanceConfig.Provider)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}

	// stop daemon and clean up local workspaces
	if providerConfig.IsDaemonProvider() {
		// clean up local workspaces
		daemonDir, err := provider.GetDaemonDir(devPodConfig.DefaultContext, proInstanceConfig.Provider)
		if err != nil {
			return err
		}

		workspaces, err := workspace.List(ctx, devPodConfig, true, log.Default)
		if err != nil {
			log.Default.Warnf("Failed to list workspaces: %v", err)
		} else {
			cleanupLocalWorkspaces(ctx, devPodConfig, workspaces, providerConfig.Name, log.Default)
		}

		daemonClient := platformdaemon.NewLocalClient(daemonDir, proInstanceConfig.Provider)
		err = daemonClient.Shutdown(ctx)
		if err != nil {
			log.Default.Warnf("Failed to shut down daemon: %v", err)
		}
	}

	// delete the provider config
	err = providercmd.DeleteProviderConfig(devPodConfig, proInstanceConfig.Provider, true)
	if err != nil {
		return err
	}

	// delete the pro instance dir itself
	proInstanceDir, err := provider2.GetProInstanceDir(devPodConfig.DefaultContext, proInstanceConfig.Host)
	if err != nil {
		return err
	}

	err = os.RemoveAll(proInstanceDir)
	if err != nil {
		return errors.Wrap(err, "delete pro instance dir")
	}

	log.Default.Donef("Successfully deleted pro instance '%s'", proInstanceName)
	return nil
}

func cleanupLocalWorkspaces(ctx context.Context, devPodConfig *config.Config, workspaces []*provider2.Workspace, providerName string, log log.Logger) {
	usedWorkspaces := []*provider2.Workspace{}

	for _, workspace := range workspaces {
		if workspace.Provider.Name == providerName {
			usedWorkspaces = append(usedWorkspaces, workspace)
		}
	}

	if len(usedWorkspaces) > 0 {
		wg := sync.WaitGroup{}
		// try to force delete all workspaces in the background
		for _, w := range usedWorkspaces {
			wg.Add(1)
			go func(w provider2.Workspace) {
				defer wg.Done()
				client, err := workspace.Get(ctx, devPodConfig, []string{w.ID}, false, log)
				if err != nil {
					log.Errorf("Failed to get workspace %s: %v", w.ID, err)
					return
				}
				// delete workspace folder
				err = clientimplementation.DeleteWorkspaceFolder(devPodConfig.DefaultContext, client.Workspace(), client.WorkspaceConfig().SSHConfigPath, log)
				if err != nil {
					log.Errorf("Failed to delete workspace %s: %v", w.ID, err)
					return
				}
				log.Donef("Successfully deleted workspace %s", w.ID)
			}(*w)
		}

		log.Infof("Waiting for %d workspaces to be deleted", len(usedWorkspaces))
		wg.Wait()
	}
}
