package zed

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"

	"github.com/loft-sh/devpod/pkg/config"
)

// Open first finds the zed binary for the local platform and then opens the zed editor with the given workspace folder
func Open(ctx context.Context, values map[string]config.OptionValue, userName, workspaceFolder, workspaceID string, log log.Logger) error {
	log.Info("Opening Zed editor ...")
	// Find the zed binary for the local platform
	zedCmd := "zed"
	if runtime.GOOS == "darwin" && command.Exists("/Applications/Zed.app/Contents/Resources/app/bin/zed") {
		zedCmd = "/Applications/Zed.app/Contents/Resources/app/bin/zed"
	}
	// Check if zed is installed and in the PATH
	if !command.Exists(zedCmd) {
		return fmt.Errorf("seems like you don't have zed installed on your computer locally")
	}
	// Open the zed editor with the given workspace ID as the SSH host and workspace folder as path
	sshHost := fmt.Sprintf("ssh://%s.devpod/%s", workspaceID, workspaceFolder)
	out, err := exec.CommandContext(ctx, zedCmd, sshHost).CombinedOutput()
	if err != nil {
		return command.WrapCommandError(out, err)
	}

	return nil
}
