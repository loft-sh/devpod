package framework

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/machine"
	"github.com/loft-sh/devpod/cmd/provider"
)

// DevPodList executes the `devpod list` command in the test framework
func (f *Framework) DevPodList(ctx context.Context) (string, error) {
	listArgs := []string{"list", "--output", "json"}

	out, _, err := f.ExecCommandCapture(ctx, listArgs)
	if err != nil {
		return "", fmt.Errorf("devpod list failed: %s", err.Error())
	}
	return out, nil
}

// DevPodUp executes the `devpod up` command in the test framework
func (f *Framework) DevPodUp(ctx context.Context, workspace string, additionalArgs ...string) error {
	upArgs := []string{"up", "--ide", "none", workspace}
	upArgs = append(upArgs, additionalArgs...)

	err := f.ExecCommand(ctx, true, true, fmt.Sprintf("Run 'ssh %s.devpod' to ssh into the devcontainer", filepath.Base(workspace)), upArgs)
	if err != nil {
		return fmt.Errorf("devpod up failed: %s", err.Error())
	}
	return nil
}

func (f *Framework) DevPodSSHEchoTestString(ctx context.Context, workspace string) error {
	err := f.ExecCommand(ctx, true, true, "mYtEsTsTrInG", []string{"ssh", "--command", "echo 'bVl0RXNUc1RySW5H' | base64 -d", workspace})
	if err != nil {
		return fmt.Errorf("devpod ssh failed: %s", err.Error())
	}
	return nil
}

func (f *Framework) DevPodProviderOptionsCheckNamespaceDescription(ctx context.Context, provider, searchStr string) error {
	err := f.ExecCommand(ctx, true, true, searchStr, []string{"provider", "options", provider})
	if err != nil {
		return fmt.Errorf("did not found value %s in devpod provider options output. error: %s", searchStr, err.Error())
	}
	return nil
}

func (f *Framework) DevPodProviderUse(ctx context.Context, provider string) error {
	err := f.ExecCommand(ctx, false, true, "", []string{"provider", "use", provider})
	if err != nil {
		return fmt.Errorf("devpod provider use failed: %s", err.Error())
	}
	return nil
}

func (f *Framework) DevPodProviderAdd(args []string) error {
	addCmd := provider.NewAddCmd(&flags.GlobalFlags{})
	return addCmd.RunE(nil, args)
}

func (f *Framework) DevPodProviderDelete(args []string) error {
	deleteCmd := provider.NewDeleteCmd(&flags.GlobalFlags{})
	return deleteCmd.RunE(nil, args)
}

func (f *Framework) DevPodProviderUpdate(args []string) error {
	updateCmd := provider.NewUpdateCmd(&flags.GlobalFlags{})
	return updateCmd.RunE(nil, args)
}

func (f *Framework) DevPodMachineCreate(args []string) error {
	createCmd := machine.NewCreateCmd(&flags.GlobalFlags{})
	return createCmd.RunE(nil, args)
}

func (f *Framework) DevPodMachineDelete(args []string) error {
	deleteCmd := machine.NewDeleteCmd(&flags.GlobalFlags{})
	return deleteCmd.RunE(nil, args)
}

func (f *Framework) DevPodWorkspaceDelete(ctx context.Context, workspace string) error {
	return f.ExecCommand(ctx, false, true, fmt.Sprintf("Successfully deleted workspace '%s'", workspace), []string{"delete", workspace})
}
