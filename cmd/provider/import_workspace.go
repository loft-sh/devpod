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

	WorkspaceId      string
	WorkspaceUid     string
	ProviderId       string
	WorkspaceOptions []string
	log              log.Logger
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
	importCmd.Flags().StringArrayVarP(
		&cmd.WorkspaceOptions, "option", "o", []string{}, "Workspace option in the form KEY=VALUE")

	return importCmd
}

func (cmd *ImportCmd) prepareImportWorkspaceOptions(options []string) (client2.ImportWorkspaceOptions, error) {
	importWorkspaceOptions := client2.ImportWorkspaceOptions{
		"WORKSPACE_ID":  cmd.WorkspaceId,
		"WORKSPACE_UID": cmd.WorkspaceUid,
		"PROVIDER_ID":   cmd.ProviderId,
	}

	userOptions, err := provider2.ParseOptions(options)
	if err != nil {
		return nil, errors.Wrap(err, "parse options")
	}

	for key, value := range userOptions {
		importWorkspaceOptions[key] = value
	}

	return importWorkspaceOptions, nil
}

func (cmd *ImportCmd) Import(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.ProviderId)
	proxyProvider, err := cmd.getProxyProvider(devPodConfig, cmd.ProviderId)
	if err != nil {
		return err
	}

	workspaceClient, err := clientimplementation.NewProxyClient(
		devPodConfig, proxyProvider, &provider2.Workspace{}, cmd.log)
	if err != nil {
		return err
	}

	options, err := cmd.prepareImportWorkspaceOptions(cmd.WorkspaceOptions)
	if err != nil {
		return err
	}

	return workspaceClient.ImportWorkspace(ctx, options)
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
