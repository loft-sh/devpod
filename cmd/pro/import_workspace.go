package pro

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
	WorkspaceOptions []string
	providerResolver *ProviderResolver
	log              log.Logger
}

// NewImportCmd creates a new command
func NewImportCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	logger := log.GetInstance()
	cmd := &ImportCmd{
		GlobalFlags:      globalFlags,
		providerResolver: &ProviderResolver{log: logger},
		log:              logger,
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
	// optional
	importCmd.Flags().StringVar(&cmd.WorkspaceContext, "workspace-context", "", "Target context for a workspace")
	importCmd.Flags().StringArrayVarP(
		&cmd.WorkspaceOptions, "option", "o", []string{}, "Workspace option in the form KEY=VALUE")

	_ = importCmd.MarkFlagRequired("workspace-id")
	_ = importCmd.MarkFlagRequired("workspace-uid")

	return importCmd
}

func (cmd *ImportCmd) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: devpod pro import-workspace <devpod-pro-host>")
	}
	devPodProHost := args[0]
	devPodConfig, err := config.LoadConfig(cmd.Context, "")
	if err != nil {
		return err
	}

	provider, err := cmd.providerResolver.Resolve(devPodConfig, devPodProHost)
	if err != nil {
		return errors.Wrap(err, "resolve provider")
	}

	options, err := provider2.ParseOptions(cmd.WorkspaceOptions)
	if err != nil {
		return errors.Wrap(err, "parse options")
	}

	workspaceDefinition, err := cmd.prepareWorkspaceToImportDefinition(devPodConfig, provider)
	if err != nil {
		return err
	}

	workspaceClient, err := clientimplementation.NewProxyClient(
		devPodConfig, provider, workspaceDefinition, cmd.log)
	if err != nil {
		return err
	}

	return workspaceClient.ImportWorkspace(ctx, options)
}

func (cmd *ImportCmd) context(devPodConfig *config.Config) (string, error) {
	if cmd.WorkspaceContext == "" {
		return devPodConfig.DefaultContext, nil
	}

	if devPodConfig.Contexts[cmd.WorkspaceContext] != nil {
		return cmd.WorkspaceContext, nil
	}

	return "", fmt.Errorf("context '%s' doesn't exist", cmd.WorkspaceContext)
}

func (cmd *ImportCmd) prepareWorkspaceToImportDefinition(
	devPodConfig *config.Config, provider *provider2.ProviderConfig) (*provider2.Workspace, error) {
	workspaceContext, err := cmd.context(devPodConfig)
	if err != nil {
		return nil, err
	}

	workspaceFolder, err := provider2.GetWorkspaceDir(workspaceContext, cmd.WorkspaceId)
	if err != nil {
		return nil, errors.Wrap(err, "get workspace dir")
	}

	return &provider2.Workspace{
		ID:       cmd.WorkspaceId,
		UID:      cmd.WorkspaceUid,
		Folder:   workspaceFolder,
		Provider: provider2.WorkspaceProviderConfig{Name: provider.Name},
		Context:  workspaceContext,
	}, nil
}

type ProviderResolver struct {
	log log.Logger
}

func (r *ProviderResolver) proInstance(
	devPodConfig *config.Config, devPodProHost string) (*provider2.ProInstance, error) {
	instances, err := workspace.ListProInstances(devPodConfig, r.log)
	if err != nil {
		return nil, errors.Wrap(err, "list pro instances")
	}
	for _, instance := range instances {
		if instance.Host == devPodProHost {
			return instance, nil
		}
	}
	return nil, fmt.Errorf("pro instance with host '%s' doesn't exist", devPodProHost)
}

func (r *ProviderResolver) Resolve(devPodConfig *config.Config, devPodProHost string) (*provider2.ProviderConfig, error) {
	instance, err := r.proInstance(devPodConfig, devPodProHost)
	if err != nil {
		return nil, errors.Wrap(err, "pro instance")
	}

	provider, err := workspace.FindProvider(devPodConfig, instance.Provider, r.log)
	if err != nil {
		return nil, errors.Wrap(err, "find provider")
	}

	if !provider.Config.IsProxyProvider() {
		return nil, fmt.Errorf("provider is not a proxy provider")
	}

	return provider.Config, nil
}
