package vscode

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	"github.com/skratchdot/open-golang/open"
)

func Open(ctx context.Context, workspace, folder string, newWindow bool, log log.Logger) error {
	log.Infof("Starting VSCode...")
	err := openViaCLI(ctx, workspace, folder, newWindow, log)
	if err != nil {
		log.Debugf("Error opening vscode via cli: %v", err)
	} else {
		return nil
	}

	return openViaBrowser(workspace, folder, newWindow, log)
}

func openViaBrowser(workspace, folder string, newWindow bool, log log.Logger) error {
	openURL := `vscode://vscode-remote/ssh-remote+` + workspace + `.devpod/` + folder
	if newWindow {
		openURL += "?windowId=_blank"
	}

	err := open.Run(openURL)
	if err != nil {
		log.Debugf("Starting VSCode caused error: %v", err)
		log.Errorf("Seems like you don't have Visual Studio Code installed on your computer locally. Please install VSCode via https://code.visualstudio.com/")
		return err
	}

	return nil
}

func openViaCLI(ctx context.Context, workspace, folder string, newWindow bool, log log.Logger) error {
	// try to find code cli
	codePath := findCLI()
	if codePath == "" {
		return fmt.Errorf("couldn't find the code binary")
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
		if strings.TrimSpace(str) == "ms-vscode-remote.remote-ssh" {
			found = true
		} else if strings.TrimSpace(str) == "ms-vscode-remote.remote-containers" {
			foundContainers = true
		}
	}

	// install remote-ssh extension
	if !found {
		args := []string{"--install-extension", "ms-vscode-remote.remote-ssh"}
		log.Debugf("Run vscode command %s %s", codePath, strings.Join(args, " "))
		out, err := exec.CommandContext(ctx, codePath, args...).Output()
		if err != nil {
			return fmt.Errorf("install ssh extension: %w", command.WrapCommandError(out, err))
		}
	}

	// open vscode via cli
	args := []string{
		"--folder-uri",
		"vscode-remote://ssh-remote+" + workspace + ".devpod/" + folder,
	}
	if foundContainers {
		args = append(args, "--disable-extension", "ms-vscode-remote.remote-containers")
	}
	if newWindow {
		args = append(args, "--new-window")
	} else {
		args = append(args, "--reuse-window")
	}
	log.Debugf("Run vscode command %s %s", codePath, strings.Join(args, " "))
	out, err = exec.CommandContext(ctx, codePath, args...).CombinedOutput()
	if err != nil {
		return command.WrapCommandError(out, err)
	}

	return nil
}

func findCLI() string {
	if command.Exists("code") {
		return "code"
	} else if runtime.GOOS == "darwin" && command.Exists("/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code") {
		return "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
	}

	return ""
}
