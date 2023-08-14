package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ImportCmd struct {
	*flags.GlobalFlags

	WorkspaceId  string
	WorkspaceUid string
	ProviderId   string
	log          log.Logger
}

// NewImportCmd creates a new command
func NewImportCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ImportCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	importCmd := &cobra.Command{
		Use:   "import-workspace",
		Short: "Imports a workspace",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Import(cobraCmd.Context(), args)
		},
	}

	// todo: enforce usage of flags
	importCmd.Flags().StringVar(&cmd.WorkspaceId, "workspace-id", "", "ID of a workspace to import")
	importCmd.Flags().StringVar(&cmd.WorkspaceUid, "workspace-uid", "", "UID of a workspace to import")
	importCmd.Flags().StringVar(
		&cmd.ProviderId, "provider-id", "", "Provider to use for importing. Must be a proxy provider")

	return importCmd
}

func (cmd *ImportCmd) Import(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.ProviderId)
	proxyProvider, err := cmd.getProxyProvider(devPodConfig, cmd.ProviderId)
	if err != nil {
		return err
	}
	w := &provider2.Workspace{
		//todo: fill workspace data
	}

	workspaceClient, err := clientimplementation.NewProxyClient(devPodConfig, proxyProvider, w, cmd.log)
	if err != nil {
		return err
	}
	return workspaceClient.ImportWorkspace(ctx, client2.ImportWorkspaceOptions{})
}

func (cmd *ImportCmd) getProxyProvider(devpodConfig *config.Config, providerID string) (*provider2.ProviderConfig, error) {
	provider, err := workspace.FindProvider(devpodConfig, providerID, cmd.log)
	if err != nil {
		return nil, errors.Wrap(err, "find provider")
	}

	if !provider.Config.IsProxyProvider() {
		return nil, fmt.Errorf("provider is not a proxy provider")
	}

	return provider.Config, nil
}
