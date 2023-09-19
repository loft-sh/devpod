package vscode

import (
	"os/exec"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
)

func InstallAlpineRequirements(logger log.Logger) {
	if !command.Exists("apk") {
		return
	}
	logger.Debugf("Install alpine requirements...")
	dependencies := []string{"build-base", "gcompat"}
	if !command.Exists("git") {
		dependencies = append(dependencies, "git")
	}
	if !command.Exists("bash") {
		dependencies = append(dependencies, "bash")
	}
	if !command.Exists("curl") {
		dependencies = append(dependencies, "curl")
	}

	out, err := exec.Command("sh", "-c", "apk update && apk add "+strings.Join(dependencies, " ")).CombinedOutput()
	if err != nil {
		logger.Infof("Error updating alpine: %w", command.WrapCommandError(out, err))
	}
}
