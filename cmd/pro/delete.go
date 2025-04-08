package pro

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	providercmd "github.com/loft-sh/devpod/cmd/provider"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	daemon "github.com/loft-sh/devpod/pkg/daemon/platform"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
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
	proInstanceConfig, err := provider.LoadProInstanceConfig(devPodConfig.DefaultContext, proInstanceName)
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
		workspaces, err := workspace.ListLocalWorkspaces(devPodConfig.DefaultContext, false, log.Default)
		if err != nil {
			log.Default.Warnf("Failed to list workspaces: %v", err)
		} else {
			cleanupLocalWorkspaces(ctx, devPodConfig, workspaces, providerConfig.Name, cmd.Owner, log.Default)
		}

		daemonClient := daemon.NewLocalClient(proInstanceConfig.Provider)
		err = daemonClient.Shutdown(ctx)
		if err != nil {
			log.Default.Warnf("Failed to shut down daemon: %v", err)
		}
		log.Default.Debug("Waiting for daemon to shut down")
		err = waitDaemonStopped(ctx, providerConfig.Name)
		if err != nil {
			log.Default.Warnf("Failed to wait for daemon to be stopped: %v", err)
		}
	}

	// delete the provider config
	err = providercmd.DeleteProviderConfig(devPodConfig, proInstanceConfig.Provider, true)
	if err != nil {
		return err
	}

	// delete the pro instance dir itself
	proInstanceDir, err := provider.GetProInstanceDir(devPodConfig.DefaultContext, proInstanceConfig.Host)
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

func cleanupLocalWorkspaces(ctx context.Context, devPodConfig *config.Config, workspaces []*provider.Workspace, providerName string, owner platform.OwnerFilter, log log.Logger) {
	usedWorkspaces := []*provider.Workspace{}

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
			go func(w provider.Workspace) {
				defer wg.Done()
				client, err := workspace.Get(ctx, devPodConfig, []string{w.ID}, true, owner, true, log)
				if err != nil {
					log.Errorf("Failed to get workspace %s: %v", w.ID, err)
					return
				}
				// delete workspace folder
				err = clientimplementation.DeleteWorkspaceFolder(devPodConfig.DefaultContext, client.Workspace(), client.WorkspaceConfig().SSHConfigPath, log)
				if err != nil {
					log.Errorf("Failed to remove workspace %s: %v", w.ID, err)
					return
				}
				log.Donef("Successfully removed workspace %s", w.ID)
			}(*w)
		}

		log.Infof("Waiting for %d workspace(s) to be removed locally", len(usedWorkspaces))
		wg.Wait()
	}
}

func waitDaemonStopped(ctx context.Context, providerName string) error {
	return wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, 5*time.Second, true, func(ctx context.Context) (done bool, err error) {
		_, err = daemon.Dial(daemon.GetSocketAddr(providerName))
		if err != nil {
			return true, nil
		}

		return false, nil
	})
}
