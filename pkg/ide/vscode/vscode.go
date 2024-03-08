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

type ReleaseChannel string

const (
	ReleaseChannelStable   ReleaseChannel = "stable"
	ReleaseChannelInsiders ReleaseChannel = "insiders"
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

func NewVSCodeServer(extensions []string, settings string, userName string, values map[string]config.OptionValue, releaseChannel ReleaseChannel, log log.Logger) *VsCodeServer {
	if releaseChannel == "" {
		releaseChannel = ReleaseChannelStable
	}

	return &VsCodeServer{
		values:         values,
		extensions:     extensions,
		settings:       settings,
		userName:       userName,
		log:            log,
		releaseChannel: releaseChannel,
	}
}

type VsCodeServer struct {
	values         map[string]config.OptionValue
	extensions     []string
	settings       string
	userName       string
	releaseChannel ReleaseChannel
	log            log.Logger
}

func (o *VsCodeServer) InstallExtensions() error {
	location, err := prepareServerLocation(o.userName, false, o.releaseChannel)
	if err != nil {
		return err
	}

	binPath := ""
	if o.releaseChannel == ReleaseChannelStable {
		binDir := filepath.Join(location, "bin")

		for {
			entries, err := os.ReadDir(binDir)
			if err != nil {
				o.log.Infof("Read dir %s: %v", binDir, err)
				o.log.Info("Wait until vscode-server is installed...")
				time.Sleep(time.Second * 3)
				continue
			} else if len(entries) == 0 {
				o.log.Infof("Read dir %s: install dir is missing", binDir)
				o.log.Info("Wait until vscode-server (%s) is installed...")
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
	} else if o.releaseChannel == ReleaseChannelInsiders {
		serversDir := filepath.Join(location, "cli", "servers")
		for {
			entries, err := os.ReadDir(serversDir)
			if err != nil {
				o.log.Infof("Read dir %s: %v", serversDir, err)
				o.log.Info("Wait until vscode-server-insiders is installed...")
				time.Sleep(time.Second * 3)
				continue
			} else if len(entries) == 0 {
				o.log.Infof("Read dir %s: install dir is missing", serversDir)
				o.log.Infof("Wait until vscode-server-insiders is installed...")
				time.Sleep(time.Second * 3)
				continue
			}

			insidersDir := ""
			// find first entry with `Insiders-` prefix
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				if !strings.HasPrefix(entry.Name(), "Insiders-") {
					continue
				}

				insidersDir = filepath.Join(serversDir, entry.Name())
			}

			if insidersDir == "" {
				o.log.Infof("Read dir %s: install dir is missing", serversDir)
				o.log.Infof("Wait until vscode-server-insiders is installed...")
				time.Sleep(time.Second * 3)
				continue
			}

			binPath = filepath.Join(insidersDir, "server", "bin", "code-server-insiders")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			out, err := exec.CommandContext(ctx, binPath, "--help").CombinedOutput()
			cancel()
			if err != nil {
				o.log.Infof("Execute %s: %v", binPath, command.WrapCommandError(out, err))
				o.log.Info("Wait until vscode-server-insiders  is installed...")
				time.Sleep(time.Second * 3)
				continue
			}

			break
		}
	}

	// start log writer
	writer := o.log.Writer(logrus.InfoLevel, false)
	errwriter := o.log.Writer(logrus.ErrorLevel, false)
	defer writer.Close()
	defer errwriter.Close()

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
		cmd.Stderr = errwriter
		err := cmd.Run()
		if err != nil {
			o.log.Info("Failed installing extension " + extension)
		}
		o.log.Info("Successfully installed extension " + extension)
	}

	return nil
}

func (o *VsCodeServer) Install() error {
	location, err := prepareServerLocation(o.userName, true, o.releaseChannel)
	if err != nil {
		return err
	}

	settingsDir := filepath.Join(location, "data", "Machine")
	err = os.MkdirAll(settingsDir, 0755)
	if err != nil {
		return err
	}

	// is installed
	settingsFile := filepath.Join(settingsDir, "settings.json")
	_, err = os.Stat(settingsFile)
	if err == nil {
		return nil
	}

	InstallAlpineRequirements(o.log)

	// add settings
	if o.settings == "" {
		o.settings = "{}"
	}

	// set settings
	err = os.WriteFile(settingsFile, []byte(o.settings), 0600)
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

func prepareServerLocation(userName string, create bool, releaseChannel ReleaseChannel) (string, error) {
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

	folderName := ".vscode-server"
	if releaseChannel == ReleaseChannelInsiders {
		folderName = ".vscode-server-insiders"
	}

	folder := filepath.Join(homeFolder, folderName)
	if create {
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			return "", err
		}
	}

	return folder, nil
}
