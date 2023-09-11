package pro

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/random"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ImportCmd struct {
	*flags.GlobalFlags

	WorkspaceId      string
	WorkspaceUid     string
	WorkspaceProject string

	Own bool
	log log.Logger
}

// NewImportCmd creates a new command
func NewImportCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	logger := log.GetInstance()
	cmd := &ImportCmd{
		GlobalFlags: globalFlags,
		log:         logger,
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
	importCmd.Flags().StringVar(&cmd.WorkspaceProject, "workspace-project", "", "Project of the workspace to import")
	importCmd.Flags().BoolVar(&cmd.Own, "own", false, "If true, will behave as if workspace was not imported")
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

	// set uid as id
	if cmd.WorkspaceId == "" {
		cmd.WorkspaceId = cmd.WorkspaceUid
	}

	// check if workspace already exists
	if provider2.WorkspaceExists(devPodConfig.DefaultContext, cmd.WorkspaceId) {
		workspaceConfig, err := provider2.LoadWorkspaceConfig(devPodConfig.DefaultContext, cmd.WorkspaceId)
		if err != nil {
			return fmt.Errorf("load workspace: %w", err)
		} else if workspaceConfig.UID == cmd.WorkspaceUid {
			cmd.log.Infof("Workspace %s already imported", cmd.WorkspaceId)
			return nil
		}

		newWorkspaceId := cmd.WorkspaceId + "-" + random.String(5)
		if provider2.WorkspaceExists(devPodConfig.DefaultContext, newWorkspaceId) {
			return fmt.Errorf("workspace %s already exists", cmd.WorkspaceId)
		}

		cmd.log.Infof("Workspace %s already exists, will use name %s instead", cmd.WorkspaceId, newWorkspaceId)
		cmd.WorkspaceId = newWorkspaceId
	}

	provider, err := resolveProInstance(devPodConfig, devPodProHost, cmd.log)
	if err != nil {
		return errors.Wrap(err, "resolve provider")
	}

	err = cmd.writeWorkspaceDefinition(devPodConfig, provider)
	if err != nil {
		return errors.Wrap(err, "prepare workspace to import definition")
	}

	cmd.log.Infof("Successfully imported workspace %s", cmd.WorkspaceId)
	return nil
}

func (cmd *ImportCmd) writeWorkspaceDefinition(devPodConfig *config.Config, provider *provider2.ProviderConfig) error {
	workspaceFolder, err := provider2.GetWorkspaceDir(devPodConfig.DefaultContext, cmd.WorkspaceId)
	if err != nil {
		return errors.Wrap(err, "get workspace dir")
	}

	workspaceObj := &provider2.Workspace{
		ID:     cmd.WorkspaceId,
		UID:    cmd.WorkspaceUid,
		Folder: workspaceFolder,
		Provider: provider2.WorkspaceProviderConfig{
			Name:    provider.Name,
			Options: map[string]config.OptionValue{},
		},
		Context:  devPodConfig.DefaultContext,
		Imported: !cmd.Own,
	}
	if cmd.WorkspaceProject != "" {
		workspaceObj.Provider.Options["LOFT_PROJECT"] = config.OptionValue{
			Value:        cmd.WorkspaceProject,
			UserProvided: true,
		}
	}

	err = provider2.SaveWorkspaceConfig(workspaceObj)
	if err != nil {
		return err
	}

	return nil
}

func resolveProInstance(devPodConfig *config.Config, devPodProHost string, log log.Logger) (*provider2.ProviderConfig, error) {
	proInstanceConfig, err := provider2.LoadProInstanceConfig(devPodConfig.DefaultContext, devPodProHost)
	if err != nil {
		return nil, fmt.Errorf("load pro instance %s: %w", devPodProHost, err)
	}

	provider, err := workspace.FindProvider(devPodConfig, proInstanceConfig.Provider, log)
	if err != nil {
		return nil, errors.Wrap(err, "find provider")
	} else if !provider.Config.IsProxyProvider() {
		return nil, fmt.Errorf("provider is not a proxy provider")
	}

	return provider.Config, nil
}
