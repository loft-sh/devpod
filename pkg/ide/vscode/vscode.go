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
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	OpenNewWindow = "OPEN_NEW_WINDOW"
)

type Flavor string

const (
	FlavorStable   Flavor = "stable"
	FlavorInsiders Flavor = "insiders"
	FlavorCursor   Flavor = "cursor"
	FlavorPositron Flavor = "positron"
	FlavorCodium   Flavor = "codium"
)

func (f Flavor) DisplayName() string {
	switch f {
	case FlavorStable:
		return "VSCode"
	case FlavorInsiders:
		return "VSCode Insiders"
	case FlavorCursor:
		return "Cursor"
	case FlavorPositron:
		return "positron"
	case FlavorCodium:
		return "VSCodium"
	default:
		return "VSCode"
	}
}

var Options = ide.Options{
	OpenNewWindow: {
		Name:        OpenNewWindow,
		Description: "If true, DevPod will open the project in a new window",
		Default:     "true",
		Enum: []string{
			"false",
			"true",
		},
	},
}

func NewVSCodeServer(extensions []string, settings string, userName string, values map[string]config.OptionValue, flavor Flavor, log log.Logger) *VsCodeServer {
	if flavor == "" {
		flavor = FlavorStable
	}

	return &VsCodeServer{
		values:     values,
		extensions: extensions,
		settings:   settings,
		userName:   userName,
		log:        log,
		flavor:     flavor,
	}
}

type VsCodeServer struct {
	values     map[string]config.OptionValue
	extensions []string
	settings   string
	userName   string
	flavor     Flavor
	log        log.Logger
}

func (o *VsCodeServer) InstallExtensions() error {
	location, err := prepareServerLocation(o.userName, false, o.flavor)
	if err != nil {
		return err
	}

	binPath := o.findServerBinaryPath(location)
	if binPath == "" {
		return fmt.Errorf("unable to locate server binary in workspace")
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
			o.log.Warn("Failed installing extension " + extension)
		}
		o.log.Info("Successfully installed extension " + extension)
	}

	return nil
}

func (o *VsCodeServer) Install() error {
	location, err := prepareServerLocation(o.userName, true, o.flavor)
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

	InstallAPKRequirements(o.log)

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

func (o *VsCodeServer) findServerBinaryPath(location string) string {
	binPath := ""
	// Limit time we spend to look for code server binary.
	// Potentially expose as context option in the future if problems arise
	deadline := time.Now().Add(time.Minute * 10)

	if o.flavor == FlavorStable {
		// check legacy location `$HOME/.vscode-server/bin`
		binDir := filepath.Join(location, "bin")
		for {
			if time.Now().After(deadline) {
				o.log.Warn("Timed out installing vscode-server")
				break
			}
			entries, err := os.ReadDir(binDir)
			if err != nil || len(entries) == 0 {
				o.log.Infof("Read dir %s: %v", binDir, err)
				o.log.Info("Wait until vscode-server is installed...")
				// check new location `$HOME/.vscode-server/cli/servers/Stable-<version>/server/bin/code-server`
				newBinPath, err := o.findCodeServerBinary(location)
				if err != nil {
					o.log.Infof("Read new location %s: %v", location, err)
					o.log.Info("Wait until vscode-server-insiders is installed...")
					time.Sleep(time.Second * 3)
					continue
				}

				binPath = newBinPath
				break
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

		return binPath
	}

	if o.flavor == FlavorCursor {
		// check legacy location `$HOME/.cursor-server/bin`
		binDir := filepath.Join(location, "bin")
		for {
			if time.Now().After(deadline) {
				o.log.Warn("Timed out installing cursor-server")
				break
			}
			entries, err := os.ReadDir(binDir)
			if err != nil || len(entries) == 0 {
				o.log.Infof("Read dir %s: %v", binDir, err)
				o.log.Info("Wait until cursor-server is installed...")
				// check new location `$HOME/.cursor-server/cli/servers/Stable-<version>/server/bin/cursor-server`
				newBinPath, err := o.findCodeServerBinary(location)
				if err != nil {
					o.log.Infof("Read new location %s: %v", location, err)
					o.log.Info("Wait until cursor-server is installed...")
					time.Sleep(time.Second * 3)
					continue
				}
				binPath = newBinPath
				break
			}

			binPath = filepath.Join(binDir, entries[0].Name(), "bin", "cursor-server")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			out, err := exec.CommandContext(ctx, binPath, "--help").CombinedOutput()
			cancel()
			if err != nil {
				o.log.Infof("Execute %s: %v", binPath, command.WrapCommandError(out, err))
				o.log.Info("Wait until cursor-server is installed...")
				time.Sleep(time.Second * 3)
				continue
			}

			break
		}

		return binPath
	}

	if o.flavor == FlavorPositron {
		// check legacy location `$HOME/.positron-server/bin`
		binDir := filepath.Join(location, "bin")
		for {
			if time.Now().After(deadline) {
				o.log.Warn("Timed out installing positron-server")
				break
			}
			entries, err := os.ReadDir(binDir)
			if err != nil || len(entries) == 0 {
				o.log.Infof("Read dir %s: %v", binDir, err)
				o.log.Info("Wait until positron-server is installed...")
				// check new location `$HOME/.positron-server/cli/servers/Stable-<version>/server/bin/positron-server`
				newBinPath, err := o.findCodeServerBinary(location)
				if err != nil {
					o.log.Infof("Read new location %s: %v", location, err)
					o.log.Info("Wait until positron-server is installed...")
					time.Sleep(time.Second * 3)
					continue
				}
				binPath = newBinPath
				break
			}

			binPath = filepath.Join(binDir, entries[0].Name(), "bin", "positron-server")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			out, err := exec.CommandContext(ctx, binPath, "--help").CombinedOutput()
			cancel()
			if err != nil {
				o.log.Infof("Execute %s: %v", binPath, command.WrapCommandError(out, err))
				o.log.Info("Wait until positron-server is installed...")
				time.Sleep(time.Second * 3)
				continue
			}

			break
		}

		return binPath
	}

	if o.flavor == FlavorCodium {
		// check legacy location `$HOME/.vscodium-server/bin`
		binDir := filepath.Join(location, "bin")
		for {
			if time.Now().After(deadline) {
				o.log.Warn("Timed out installing vscodium-server")
				break
			}
			entries, err := os.ReadDir(binDir)
			if err != nil || len(entries) == 0 {
				o.log.Infof("Read dir %s: %v", binDir, err)
				o.log.Info("Wait until vscodium-server is installed...")
				// check new location `$HOME/.vscodium-server/cli/servers/Stable-<version>/server/bin/code-server`
				newBinPath, err := o.findCodeServerBinary(location)
				if err != nil {
					o.log.Infof("Read new location %s: %v", location, err)
					o.log.Info("Wait until vscodium is installed...")
					time.Sleep(time.Second * 3)
					continue
				}

				binPath = newBinPath
				break
			}

			binPath = filepath.Join(binDir, entries[0].Name(), "bin", "codium-server")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			out, err := exec.CommandContext(ctx, binPath, "--help").CombinedOutput()
			cancel()
			if err != nil {
				o.log.Infof("Execute %s: %v", binPath, command.WrapCommandError(out, err))
				o.log.Info("Wait until vscodium-server is installed...")
				time.Sleep(time.Second * 3)
				continue
			}

			break
		}

		return binPath
	}

	if o.flavor == FlavorInsiders {
		serversDir := filepath.Join(location, "cli", "servers")
		for {
			if time.Now().After(deadline) {
				o.log.Warn("Timed out installing vscode-server-insiders")
				break
			}
			entries, err := os.ReadDir(serversDir)
			if err != nil || len(entries) == 0 {
				o.log.Infof("Read dir %s: %v", serversDir, err)
				o.log.Info("Wait until vscode-server-insiders is installed...")
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

		return binPath
	}

	return binPath
}

func (o *VsCodeServer) findCodeServerBinary(location string) (string, error) {
	serversDir := filepath.Join(location, "cli", "servers")
	entries, err := os.ReadDir(serversDir)
	if err != nil {
		return "", fmt.Errorf("read dir %s: %w", serversDir, err)
	} else if len(entries) == 0 {
		return "", fmt.Errorf("read dir %s: install dir is missing", serversDir)
	}

	stableDir := ""
	// find first entry with `Stable-` prefix
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), "Stable-") {
			continue
		}

		stableDir = filepath.Join(serversDir, entry.Name())
	}

	if stableDir == "" {
		return "", fmt.Errorf("read dir %s: install dir is missing", serversDir)
	}

	binPath := filepath.Join(stableDir, "server", "bin", "code-server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
	out, err := exec.CommandContext(ctx, binPath, "--help").CombinedOutput()
	cancel()
	if err != nil {
		return "", fmt.Errorf("execute %s: %w", binPath, command.WrapCommandError(out, err))
	}

	return binPath, nil
}

func prepareServerLocation(userName string, create bool, flavor Flavor) (string, error) {
	var err error
	homeFolder := ""
	if userName != "" {
		homeFolder, err = command.GetHome(userName)
	} else {
		homeFolder, err = util.UserHomeDir()
	}
	if err != nil {
		return "", err
	}

	folderName := ".vscode-server"
	switch flavor {
	case FlavorStable:
		folderName = ".vscode-server"
	case FlavorInsiders:
		folderName = ".vscode-server-insiders"
	case FlavorCursor:
		folderName = ".cursor-server"
	case FlavorPositron:
		folderName = ".positron-server"
	case FlavorCodium:
		folderName = ".vscodium-server"
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
