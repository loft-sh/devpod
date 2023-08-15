package provider

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
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
	WorkspaceContext string
	WorkspaceFolder  string
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
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	importCmd.Flags().StringVar(&cmd.WorkspaceId, "workspace-id", "", "ID of a workspace to import")
	importCmd.Flags().StringVar(&cmd.WorkspaceUid, "workspace-uid", "", "UID of a workspace to import")
	importCmd.Flags().StringVar(&cmd.WorkspaceContext, "workspace-context", "", "Target context for a workspace")
	importCmd.Flags().StringVar(&cmd.WorkspaceFolder, "workspace-folder", "", "Path to the directory for a new workspace")
	importCmd.Flags().StringVar(
		&cmd.ProviderId, "provider-id", "", "Provider to use for importing. Must be a proxy provider")
	importCmd.Flags().StringArrayVarP(
		&cmd.WorkspaceOptions, "option", "o", []string{}, "Workspace option in the form KEY=VALUE")

	_ = importCmd.MarkFlagRequired("workspace-id")
	_ = importCmd.MarkFlagRequired("workspace-uid")
	_ = importCmd.MarkFlagRequired("provider-id")
	_ = importCmd.MarkFlagRequired("workspace-folder")

	return importCmd
}

func (cmd *ImportCmd) prepareWorkspaceToImportDefinition(devPodConfig *config.Config) (*provider2.Workspace, error) {
	var workspaceContext string

	if cmd.WorkspaceContext == "" {
		workspaceContext = devPodConfig.DefaultContext
	} else if devPodConfig.Contexts[cmd.WorkspaceContext] != nil {
		workspaceContext = cmd.WorkspaceContext
	} else {
		return nil, fmt.Errorf("context '%s' doesn't exist", cmd.WorkspaceContext)
	}

	return &provider2.Workspace{
		ID:       cmd.WorkspaceId,
		UID:      cmd.WorkspaceUid,
		Folder:   cmd.WorkspaceFolder,
		Provider: provider2.WorkspaceProviderConfig{Name: cmd.ProviderId},
		Context:  workspaceContext,
	}, nil
}

func (cmd *ImportCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.ProviderId)
	if err != nil {
		return err
	}
	proxyProvider, err := cmd.getProxyProvider(devPodConfig, cmd.ProviderId)
	if err != nil {
		return err
	}

	options, err := provider2.ParseOptions(cmd.WorkspaceOptions)
	if err != nil {
		return errors.Wrap(err, "parse options")
	}

	workspaceDefinition, err := cmd.prepareWorkspaceToImportDefinition(devPodConfig)
	if err != nil {
		return err
	}

	workspaceClient, err := clientimplementation.NewProxyClient(
		devPodConfig, proxyProvider, workspaceDefinition, cmd.log)
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
