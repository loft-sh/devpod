package daemonclient

import (
	"context"
	"fmt"
	"io"

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

	instance, err = createInstanceInteractive(ctx, baseClient, c.workspace.ID, c.workspace.UID, c.workspace.Source.String(), c.workspace.Picture, c.log)
	if err != nil {
		return err
	}

	instance, err = c.localClient.CreateWorkspace(ctx, instance)
	if err != nil {
		return err
	}

	c.workspace.Pro = &provider.ProMetadata{
		InstanceName: instance.Name,
		Project:      project.ProjectFromNamespace(instance.Namespace),
		DisplayName:  instance.Spec.DisplayName,
	}

	err = provider.SaveWorkspaceConfig(c.workspace)
	if err != nil {
		return fmt.Errorf("save workspace config: %w", err)
	}

	return nil
}
