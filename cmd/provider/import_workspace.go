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
	"os"
)

type ImportCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewImportCmd creates a new command
func NewImportCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ImportCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	return &cobra.Command{
		Use:   "import-workspace",
		Short: "Imports a workspace",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Import(cobraCmd.Context(), args)
		},
	}
}

func (cmd *ImportCmd) Import(ctx context.Context, args []string) error {
	workspaceMetaData, err := cmd.readWorkspaceMetaData()
	if err != nil {
		return err
	}
	devPodConfig, err := config.LoadConfig(cmd.Context, workspaceMetaData.ProviderID)
	proxyProvider, err := cmd.getProxyProvider(devPodConfig, workspaceMetaData.ProviderID)
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

type WorkspaceMetaData struct {
	WorkspaceUID string `json:"workspaceUID"`
	WorkspaceID  string `json:"workspaceID"`
	ProviderID   string `json:"providerID"`
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

func (cmd *ImportCmd) readWorkspaceMetaData() (*WorkspaceMetaData, error) {
	workspaceUID := os.Getenv("WORKSPACE_UID")
	if workspaceUID == "" {
		return nil, fmt.Errorf("%s is missing in environment", "WORKSPACE_UID")
	}

	workspaceID := os.Getenv("WORKSPACE_ID")
	if workspaceID == "" {
		return nil, fmt.Errorf("%s is missing in environment", "WORKSPACE_ID")
	}

	providerID := os.Getenv("PROVIDER_ID")
	if providerID == "" {
		return nil, fmt.Errorf("%s is missing in environment", "PROVIDER_ID")
	}

	return &WorkspaceMetaData{
		WorkspaceUID: workspaceUID,
		WorkspaceID:  workspaceID,
		ProviderID:   providerID,
	}, nil
}
