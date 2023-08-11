package provider

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
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

	proxyProvider, err := cmd.getProxyProvider(workspaceMetaData.ProviderID)
	if err != nil {
		return err
	}
	devPodConfig, err := config.LoadConfig(cmd.Context, proxyProvider.Name)
	workspace := &provider2.Workspace{
		//todo: fill workspace data
	}

	workspaceClient, err := clientimplementation.NewProxyClient(devPodConfig, proxyProvider, workspace, cmd.log)
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

func (cmd *ImportCmd) getProxyProvider(providerID string) (*provider2.ProviderConfig, error) {
	provider, err := provider2.ParseProvider(bytes.NewReader([]byte(providerID)))
	if err != nil {
		return nil, errors.Wrap(err, "parse provider")
	}

	if !provider.IsProxyProvider() {
		return nil, fmt.Errorf("provider is not a proxy provider")
	}

	return provider, nil
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
