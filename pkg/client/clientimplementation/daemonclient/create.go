package daemonclient

import (
	"context"
	"fmt"
	"io"

	"github.com/loft-sh/devpod/pkg/platform/form"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log/terminal"
)

func (c *client) Create(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := c.initPlatformClient(ctx)
	if err != nil {
		return err
	}

	instance, err := c.localClient.GetWorkspace(ctx, c.workspace.UID)
	if err != nil {
		return err
	}
	// Nothing left to do if we already have an instance
	if instance != nil {
		return nil
	}
	if !terminal.IsTerminalIn {
		return fmt.Errorf("unable to create new instance through CLI if stdin is not a terminal")
	}

	instance, err = form.CreateInstance(ctx, baseClient, c.workspace.ID, c.workspace.UID, c.log)
	if err != nil {
		return err
	}

	_, err = c.localClient.CreateWorkspace(ctx, instance)
	if err != nil {
		return err
	}

	// once we have the instance, update workspace and save config
	// TODO: Do we need a file lock?
	workspaceConfig, err := provider.LoadWorkspaceConfig(c.workspace.Context, c.workspace.ID)
	if err != nil {
		return fmt.Errorf("load workspace config: %w", err)
	}
	workspaceConfig.Pro = &provider.ProMetadata{
		InstanceName: instance.GetName(),
		Project:      project.ProjectFromNamespace(instance.GetNamespace()),
		DisplayName:  instance.Spec.DisplayName,
	}

	err = provider.SaveWorkspaceConfig(workspaceConfig)
	if err != nil {
		return fmt.Errorf("save workspace config: %w", err)
	}

	return nil
}
