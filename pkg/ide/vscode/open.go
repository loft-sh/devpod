package vscode

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	"github.com/skratchdot/open-golang/open"
)

const (
	FlatpakStable         string = "com.visualstudio.code"
	FlatpakInsiders       string = "com.visualstudio.code.insiders"
	FlatpakCodium         string = "com.vscodium.codium"
	FlatpakCodiumInsiders string = "com.vscodium.codium-insiders"
)

func Open(ctx context.Context, workspace, folder string, newWindow bool, flavor Flavor, sshConfigPath string, log log.Logger) error {
	log.Infof("Starting %s...", flavor.DisplayName())
	cliErr := openViaCLI(ctx, workspace, folder, newWindow, flavor, sshConfigPath, log)
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
	protocol := `vscode://`
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
	case FlavorCodiumInsiders:
		protocol = `codium-insiders://`
	}

	openURL := protocol + `vscode-remote/ssh-remote+` + workspace + `.devpod/` + folder
	if newWindow {
		openURL += "?windowId=_blank"
	}

	err := open.Run(openURL)
	if err != nil {
		log.Debugf("Starting %s caused error: %v", flavor, err)
		log.Errorf("Seems like you don't have %s installed on your computer locally", flavor)
		return err
	}

	return nil
}

func openViaCLI(ctx context.Context, workspace, folder string, newWindow bool, flavor Flavor, sshConfigPath string, log log.Logger) error {
	// try to find code cli
	codePath := findCLI(flavor, log)
	if codePath == nil {
		return fmt.Errorf("couldn't find the %s binary", flavor)
	}

	if codePath[0] == "flatpak" {
		log.Debugf("Running with Flatpak suing the package %s.", codePath[2])
		out, err := exec.Command(codePath[0], "ps", "--columns=application").Output()
		if err != nil {
			return command.WrapCommandError(out, err)
		}
		splitted := strings.Split(string(out), "\n")
		foundRunning := false
		// Ignore the header
		for _, str := range splitted[1:] {
			if strings.TrimSpace(str) == codePath[2] {
				foundRunning = true
				break
			}
		}

		if foundRunning {
			log.Warnf("The IDE is already running via Flatpak. If you are encountering SSH connectivity issues, make sure to give read access to your SSH config file (e.g flatpak override %s --filesystem=%s) and restart your IDE.", codePath[2], sshConfigPath)
		}

		codePath = slices.Insert(codePath, 2, fmt.Sprintf("--filesystem=%s:ro", sshConfigPath))
	}

	sshExtension := "ms-vscode-remote.remote-ssh"
	if flavor == FlavorCodium || flavor == FlavorCodiumInsiders {
		sshExtension = "jeanp413.open-remote-ssh"
	}

	// make sure ms-vscode-remote.remote-ssh is installed
	listArgs := append(codePath, "--list-extensions")
	out, err := exec.Command(listArgs[0], listArgs[1:]...).Output()
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
		args := append(codePath, "--install-extension", sshExtension)
		log.Debugf("Run vscode command %s %s", args[0], strings.Join(args[1:], " "))
		out, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
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
	args = append(codePath, args...)
	args = append(args, folderUriArg)
	log.Debugf("Run %s command %s %s", flavor.DisplayName(), args[0], strings.Join(args[1:], " "))
	out, err = exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return command.WrapCommandError(out, err)
	}

	return nil
}

func existsInFlatpak(packageName string, log log.Logger) bool {
	if err := exec.Command("flatpak", "info", packageName).Run(); err == nil {
		return true
	} else {
		log.Debugf("Flatpak command for %s returned: %s", packageName, err)
	}
	return false
}

func getCommandArgs(execName, macOSPath, flatpakPackage string, log log.Logger) []string {
	if command.Exists(execName) {
		return []string{execName}
	}

	if runtime.GOOS == "darwin" && command.Exists(macOSPath) {
		return []string{macOSPath}
	}

	if runtime.GOOS == "linux" && flatpakPackage != "" && command.Exists("flatpak") && existsInFlatpak(flatpakPackage, log) {
		return []string{"flatpak", "run", flatpakPackage}
	}

	return nil
}

func findCLI(flavor Flavor, log log.Logger) []string {
	switch flavor {
	case FlavorStable:
		return getCommandArgs("code", "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code", FlatpakStable, log)
	case FlavorInsiders:
		return getCommandArgs("code-insiders", "/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code", FlatpakInsiders, log)
	case FlavorCursor:
		return getCommandArgs("cursor", "/Applications/Cursor.app/Contents/Resources/app/bin/cursor", "", log)
	case FlavorPositron:
		return getCommandArgs("positron", "/Applications/Positron.app/Contents/Resources/app/bin/positron", "", log)
	case FlavorCodium:
		return getCommandArgs("codium", "/Applications/Codium.app/Contents/Resources/app/bin/codium", FlatpakCodium, log)
	case FlavorCodiumInsiders:
		return getCommandArgs("codium-insiders", "/Applications/CodiumInsiders.app/Contents/Resources/app/bin/codium-insiders", FlatpakCodiumInsiders, log)
	}
	return nil
}
