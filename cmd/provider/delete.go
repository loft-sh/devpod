package provider

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	logpkg "github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	IgnoreNotFound bool
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a provider",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	deleteCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "Treat \"provider not found\" as a successful delete")
	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("please specify a provider to delete")
	}

	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider := devPodConfig.Current().DefaultProvider
	if len(args) > 0 {
		provider = args[0]
	} else if provider == "" {
		return fmt.Errorf("please specify a provider to delete")
	}

	// check if this provider is associated with a pro instance
	proInstances, err := workspace.ListProInstances(devPodConfig, logpkg.Default)
	if err != nil {
		return fmt.Errorf("list pro instances: %w", err)
	}
	for _, instance := range proInstances {
		if instance.Provider == provider {
			return fmt.Errorf("cannot delete provider '%s', because it is connected to Pro instance '%s'. Removing the Pro instance will automatically delete this provider", instance.Provider, instance.Host)
		}
	}

	// delete the provider
	err = DeleteProvider(ctx, devPodConfig, provider, cmd.IgnoreNotFound, false, logpkg.Default)
	if err != nil {
		return err
	}

	logpkg.Default.Donef("Successfully deleted provider '%s'", provider)
	return nil
}

func DeleteProvider(ctx context.Context, devPodConfig *config.Config, provider string, ignoreNotFound bool, cleanup bool, log logpkg.Logger) error {
	// check if there are workspaces that still use this provider
	workspaces, err := workspace.List(ctx, devPodConfig, false, log)
	if err != nil {
		return err
	}
	usedWorkspaces := []*provider2.Workspace{}
	for _, workspace := range workspaces {
		if workspace.Provider.Name == provider {
			usedWorkspaces = append(usedWorkspaces, workspace)
		}
	}

	if len(usedWorkspaces) > 0 {
		if cleanup {
			wg := sync.WaitGroup{}
			// try to force delete all workspaces in the background
			for _, w := range usedWorkspaces {
				wg.Add(1)
				go func(w provider2.Workspace) {
					defer wg.Done()
					_, err := workspace.Delete(ctx, devPodConfig, []string{w.ID}, true, true, client.DeleteOptions{
						IgnoreNotFound: true,
						Force:          true,
					}, logpkg.Discard)
					if err != nil {
						log.Errorf("Failed to delete workspace %s: %v", w.ID, err)
						return
					}
					log.Donef("Successfully deleted workspace %s", w.ID)
				}(*w)
			}

			log.Infof("Waiting for %d workspaces to be deleted", len(usedWorkspaces))
			wg.Wait()
		} else {
			workspace := usedWorkspaces[0]
			return fmt.Errorf("cannot delete provider '%s', because workspace '%s' is still using it. Please delete the workspace '%s' before deleting the provider", workspace.Provider.Name, workspace.ID, workspace.ID)
		}
	}

	if devPodConfig.Current().DefaultProvider == provider {
		devPodConfig.Current().DefaultProvider = ""
	}
	delete(devPodConfig.Current().Providers, provider)
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	providerDir, err := provider2.GetProviderDir(devPodConfig.DefaultContext, provider)
	if err != nil {
		return err
	}
	_, err = os.Stat(providerDir)
	if err != nil {
		if os.IsNotExist(err) {
			if ignoreNotFound {
				return nil
			}

			return fmt.Errorf("provider '%s' does not exist", provider)
		}

		return err
	}
	err = os.RemoveAll(providerDir)
	if err != nil {
		return errors.Wrap(err, "delete provider dir")
	}

	return nil
}
