package vscode

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	"github.com/skratchdot/open-golang/open"
)

func Open(ctx context.Context, workspace, folder string, newWindow bool, flavor Flavor, log log.Logger) error {
	log.Infof("Starting %s...", flavor.DisplayName())
	cliErr := openViaCLI(ctx, workspace, folder, newWindow, flavor, log)
	if cliErr == nil {
		return nil
	}

	browserErr := openViaBrowser(workspace, folder, newWindow, flavor, log)
	if browserErr == nil {
		return nil
	}

	return errors.Join(cliErr, browserErr)
}

func openViaBrowser(workspace, folder string, newWindow bool, flavor Flavor, log log.Logger) error {
	var protocol string
	switch flavor {
	case FlavorStable:
		protocol = `vscode://`
	case FlavorInsiders:
		protocol = `vscode-insiders://`
	case FlavorCursor:
		protocol = `cursor://`
	case FlavorPositron:
		protocol = `positron://`
	case FlavorCodium:
		protocol = `codium://`
	case FlavorWindsurf:
		protocol = `windsurf://`
	default:
		return fmt.Errorf("unknown flavor %s", flavor)
	}

	openURL := protocol + `vscode-remote/ssh-remote+` + workspace + `.devpod/` + folder
	if newWindow {
		openURL += "?windowId=_blank"
	}

	err := open.Run(openURL)
	if err != nil {
		log.Debugf("Starting %s caused error: %v", flavor, err)
		log.Errorf("Seems like you don't have %s installed on your computer locally", flavor.DisplayName())
		return err
	}

	return nil
}

func openViaCLI(ctx context.Context, workspace, folder string, newWindow bool, flavor Flavor, log log.Logger) error {
	// try to find code cli
	codePath := findCLI(flavor)
	if codePath == "" {
		return fmt.Errorf("couldn't find the %s binary", flavor)
	}

	sshExtension := "ms-vscode-remote.remote-ssh"
	if flavor == FlavorCodium {
		sshExtension = "jeanp413.open-remote-ssh"
	}

	// make sure ms-vscode-remote.remote-ssh is installed
	out, err := exec.Command(codePath, "--list-extensions").Output()
	if err != nil {
		return command.WrapCommandError(out, err)
	}
	splitted := strings.Split(string(out), "\n")
	found := false
	foundContainers := false
	for _, str := range splitted {
		if strings.TrimSpace(str) == sshExtension {
			found = true
		} else if strings.TrimSpace(str) == "ms-vscode-remote.remote-containers" {
			foundContainers = true
		}
	}

	// install remote-ssh extension
	if !found {
		args := []string{"--install-extension", sshExtension}
		log.Debugf("Run vscode command %s %s", codePath, strings.Join(args, " "))
		out, err := exec.CommandContext(ctx, codePath, args...).Output()
		if err != nil {
			return fmt.Errorf("install ssh extension: %w", command.WrapCommandError(out, err))
		}
	}

	// open vscode via cli
	args := make([]string, 0, 5)
	if foundContainers {
		args = append(args, "--disable-extension", "ms-vscode-remote.remote-containers")
	}
	if newWindow {
		args = append(args, "--new-window")
	} else {
		args = append(args, "--reuse-window")
	}
	// Needs to be separated by `=` because of windows
	folderUriArg := fmt.Sprintf("--folder-uri=vscode-remote://ssh-remote+%s.devpod/%s", workspace, folder)
	args = append(args, folderUriArg)
	log.Debugf("Run %s command %s %s", flavor.DisplayName(), codePath, strings.Join(args, " "))
	out, err = exec.CommandContext(ctx, codePath, args...).CombinedOutput()
	if err != nil {
		return command.WrapCommandError(out, err)
	}

	return nil
}

func findCLI(flavor Flavor) string {
	if flavor == FlavorStable {
		if command.Exists("code") {
			return "code"
		} else if runtime.GOOS == "darwin" && command.Exists("/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code") {
			return "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
		}

		return ""
	}

	if flavor == FlavorInsiders {
		if command.Exists("code-insiders") {
			return "code-insiders"
		} else if runtime.GOOS == "darwin" && command.Exists("/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code") {
			return "/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code"
		}

		return ""
	}

	if flavor == FlavorCursor {
		if command.Exists("cursor") {
			return "cursor"
		} else if runtime.GOOS == "darwin" && command.Exists("/Applications/Cursor.app/Contents/Resources/app/bin/cursor") {
			return "/Applications/Cursor.app/Contents/Resources/app/bin/cursor"
		}

		return ""
	}

	if flavor == FlavorPositron {
		if command.Exists("positron") {
			return "positron"
		} else if runtime.GOOS == "darwin" && command.Exists("/Applications/Positron.app/Contents/Resources/app/bin/positron") {
			return "/Applications/Positron.app/Contents/Resources/app/bin/positron"
		}

		return ""
	}

	if flavor == FlavorCodium {
		if command.Exists("codium") {
			return "codium"
		} else if runtime.GOOS == "darwin" && command.Exists("/Applications/Codium.app/Contents/Resources/app/bin/codium") {
			return "/Applications/Codium.app/Contents/Resources/app/bin/codium"
		}

		return ""
	}

	if flavor == FlavorWindsurf {
		if command.Exists("windsurf") {
			return "windsurf"
		} else if runtime.GOOS == "darwin" && command.Exists("/Applications/Windsurf.app/Contents/Resources/app/bin/windsurf") {
			return "/Applications/Windsurf.app/Contents/Resources/app/bin/windsurf"
		}

		return ""
	}

	return ""
}
