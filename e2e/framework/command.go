package framework

import (
	"context"
	"fmt"
)

// DevPodUp executes the `devpod up` command in the test framework
func (f *Framework) DevPodUp(ctx context.Context, workspace string) error {
	err := f.ExecCommand(ctx, true, true, "Successfully started vscode in browser mode.", []string{"up", "--ide", "none", workspace})
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

func (f *Framework) DevPodProviderUse(ctx context.Context, provider string) error {
	err := f.ExecCommand(ctx, false, true, "", []string{"provider", "use", provider})
	if err != nil {
		return fmt.Errorf("devpod provider use failed: %s", err.Error())
	}
	return nil
}
