package provider

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/json"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"os"
)

func NewServerProviderWrapper(provider types.ServerProvider) types.ServerProvider {
	return &serverProviderWrapper{ServerProvider: provider}
}

type serverProviderWrapper struct {
	types.ServerProvider
}

func (s *serverProviderWrapper) Create(ctx context.Context, workspace *config.Workspace, options types.CreateOptions) error {
	err := createWorkspaceFolder(workspace, s.Name())
	if err != nil {
		return err
	}

	return s.ServerProvider.Create(ctx, workspace, options)
}

func (s *serverProviderWrapper) Destroy(ctx context.Context, workspace *config.Workspace, options types.DestroyOptions) error {
	err := s.ServerProvider.Destroy(ctx, workspace, options)
	if err != nil {
		return err
	}

	return deleteWorkspaceFolder(workspace.ID)
}

func NewWorkspaceProviderWrapper(provider types.WorkspaceProvider) types.WorkspaceProvider {
	return &workspaceProviderWrapper{WorkspaceProvider: provider}
}

type workspaceProviderWrapper struct {
	types.WorkspaceProvider
}

func (w *workspaceProviderWrapper) WorkspaceCreate(ctx context.Context, workspace *config.Workspace, options types.WorkspaceCreateOptions) error {
	err := createWorkspaceFolder(workspace, w.Name())
	if err != nil {
		return err
	}

	return w.WorkspaceProvider.WorkspaceCreate(ctx, workspace, options)
}

func (w *workspaceProviderWrapper) WorkspaceDestroy(ctx context.Context, workspace *config.Workspace, options types.WorkspaceDestroyOptions) error {
	err := w.WorkspaceProvider.WorkspaceDestroy(ctx, workspace, options)
	if err != nil {
		return err
	}

	return deleteWorkspaceFolder(workspace.ID)
}

func createWorkspaceFolder(workspace *config.Workspace, provider string) error {
	// save config
	workspace.CreationTimestamp = json.Now()
	workspace.Provider.Name = provider
	err := config.SaveWorkspaceConfig(workspace)
	if err != nil {
		return err
	}

	return nil
}

func deleteWorkspaceFolder(id string) error {
	workspaceFolder, err := config.GetWorkspaceDir(id)
	if err != nil {
		return err
	}

	// remove workspace folder
	err = os.RemoveAll(workspaceFolder)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
