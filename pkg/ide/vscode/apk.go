package vscode

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
)

// InstallAPKRequirements installs the requirements using apk.
//
// This is used by Alpine- and Wolfi-based images.
func InstallAPKRequirements(logger log.Logger) {
	if !command.Exists("apk") {
		return
	}

	dependencies := []string{"build-base"}
	if all, err := os.ReadFile("/etc/os-release"); err != nil {
		logger.Errorf("Error reading /etc/os-release: %v", err)
		return
	} else if !bytes.Contains(all, []byte("ID=alpine")) {
		// Alpine needs gcompat for compatibility with musl.
		// Wolfi-based distros don't need this, and Wolfi doesn't have it.
		dependencies = append(dependencies, "gcompat")
	}
	logger.Debugf("Install apk requirements...")
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
		logger.Errorf("Error updating apk dependencies: %v", command.WrapCommandError(out, err))
	}
}
