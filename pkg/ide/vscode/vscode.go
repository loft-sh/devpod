package vscode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	OpenNewWindow = "OPEN_NEW_WINDOW"
)

var Options = ide.Options{
	OpenNewWindow: {
		Name:        OpenNewWindow,
		Description: "If true, DevPod will open the project in a new vscode window",
		Default:     "true",
		Enum: []string{
			"false",
			"true",
		},
	},
}

func NewVSCodeServer(extensions []string, settings string, userName string, values map[string]config.OptionValue, log log.Logger) *VsCodeServer {
	return &VsCodeServer{
		values:     values,
		extensions: extensions,
		settings:   settings,
		userName:   userName,
		log:        log,
	}
}

type VsCodeServer struct {
	values     map[string]config.OptionValue
	extensions []string
	settings   string
	userName   string
	log        log.Logger
}

func (o *VsCodeServer) InstallExtensions() error {
	location, err := PrepareVSCodeServerLocation(o.userName, false)
	if err != nil {
		return err
	}

	// wait until vscode server is installed
	binPath := ""
	binDir := filepath.Join(location, "bin")
	for {
		entries, err := os.ReadDir(binDir)
		if err != nil {
			o.log.Debugf("Read dir %s: %v", binDir, err)
			o.log.Info("Wait until vscode-server is installed...")
			time.Sleep(time.Second * 3)
			continue
		} else if len(entries) == 0 {
			o.log.Debugf("Read dir %s: install dir is missing", binDir)
			o.log.Info("Wait until vscode-server is installed...")
			time.Sleep(time.Second * 3)
			continue
		}

		binPath = filepath.Join(binDir, entries[0].Name(), "bin", "code-server")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
		out, err := exec.CommandContext(ctx, binPath, "--help").CombinedOutput()
		cancel()
		if err != nil {
			o.log.Infof("Execute %s: %v", binPath, command.WrapCommandError(out, err))
			o.log.Info("Wait until vscode-server is installed...")
			time.Sleep(time.Second * 3)
			continue
		}

		break
	}

	// start log writer
	writer := o.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// download extensions
	for _, extension := range o.extensions {
		o.log.Info("Install extension " + extension + "...")
		runCommand := fmt.Sprintf("%s serve-local --accept-server-license-terms --install-extension '%s'", binPath, extension)
		args := []string{}
		if o.userName != "" {
			args = append(args, "su", o.userName, "-c", runCommand)
		} else {
			args = append(args, "sh", "-c", runCommand)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = writer
		cmd.Stderr = writer
		err := cmd.Run()
		if err != nil {
			o.log.Info("Failed installing extension " + extension)
		}
		o.log.Info("Successfully installed extension " + extension)
	}

	return nil
}

func (o *VsCodeServer) Install() error {
	location, err := PrepareVSCodeServerLocation(o.userName, true)
	if err != nil {
		return err
	}

	settingsDir := filepath.Join(location, "data", "Machine")
	err = os.MkdirAll(settingsDir, 0777)
	if err != nil {
		return err
	}

	// is installed
	settingsFile := filepath.Join(settingsDir, "settings.json")
	_, err = os.Stat(settingsFile)
	if err == nil {
		return nil
	}

	// install requirements alpine
	if command.Exists("apk") {
		o.log.Debugf("Install vscode dependencies...")
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
			o.log.Infof("Error updating alpine: %w", command.WrapCommandError(out, err))
		}
	}

	// add settings
	if o.settings == "" {
		o.settings = "{}"
	}

	// set settings
	err = os.WriteFile(settingsFile, []byte(o.settings), 0666)
	if err != nil {
		return err
	}

	// chown location
	if o.userName != "" {
		err = copy2.ChownR(location, o.userName)
		if err != nil {
			return errors.Wrap(err, "chown")
		}
	}

	return nil
}

func PrepareVSCodeServerLocation(userName string, create bool) (string, error) {
	var err error
	homeFolder := ""
	if userName != "" {
		homeFolder, err = command.GetHome(userName)
	} else {
		homeFolder, err = homedir.Dir()
	}
	if err != nil {
		return "", err
	}

	folder := filepath.Join(homeFolder, ".vscode-server")
	if create {
		err = os.MkdirAll(folder, 0777)
		if err != nil {
			return "", err
		}
	}

	return folder, nil
}
