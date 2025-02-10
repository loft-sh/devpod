package fleet

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
	"github.com/pkg/errors"
)

const (
	FleetURLFile = "/tmp/devpod-fleet.url.txt"
)

const (
	VersionOption       = "VERSION"
	DownloadAmd64Option = "DOWNLOAD_AMD64"
	DownloadArm64Option = "DOWNLOAD_ARM64"
)

var Options = ide.Options{
	VersionOption: {
		Name:        VersionOption,
		Description: "The version of fleet to install",
		Default:     "latest",
	},
	DownloadArm64Option: {
		Name:        DownloadArm64Option,
		Description: "The download url for the arm64 install script",
		Default:     "https://download.jetbrains.com/product?code=FLL&release.type=preview&release.type=eap&platform=linux_aarch64",
	},
	DownloadAmd64Option: {
		Name:        DownloadAmd64Option,
		Description: "The download url for the amd64 install script",
		Default:     "https://download.jetbrains.com/product?code=FLL&release.type=preview&release.type=eap&platform=linux_x64",
	},
}

func NewFleetServer(userName string, values map[string]config.OptionValue, log log.Logger) *FleetServer {
	return &FleetServer{
		values:   values,
		userName: userName,
		log:      log,
	}
}

type FleetServer struct {
	values   map[string]config.OptionValue
	userName string
	log      log.Logger
}

func (o *FleetServer) Install(projectDir string) error {
	location, err := prepareFleetServerLocation(o.userName)
	if err != nil {
		return err
	}

	// is installed
	fleetBinary := filepath.Join(location, "fleet")
	_, err = os.Stat(fleetBinary)
	if err == nil {
		return o.Start(fleetBinary, location, projectDir)
	}

	// check what release we need to download
	var url string
	if runtime.GOARCH == "arm64" {
		url = Options.GetValue(o.values, DownloadArm64Option)
	} else {
		url = Options.GetValue(o.values, DownloadAmd64Option)
	}

	// download binary
	o.log.Infof("Downloading fleet...")
	resp, err := devpodhttp.GetHTTPClient().Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code while trying to download fleet from %s: %d", url, resp.StatusCode)
	}

	f, err := os.OpenFile(fleetBinary, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("download fleet: %w", err)
	}
	_ = f.Close()

	// chown location
	if o.userName != "" {
		err = copy2.ChownR(location, o.userName)
		if err != nil {
			return errors.Wrap(err, "chown")
		}
	}

	o.log.Infof("Successfully downloaded fleet...")
	return o.Start(fleetBinary, location, projectDir)
}

func (o *FleetServer) Start(binaryPath, location, projectDir string) error {
	wasStarted := false
	var readCloser io.ReadCloser
	stderrBuffer := &bytes.Buffer{}

	err := single.Single("fleet.pid", func() (*exec.Cmd, error) {
		o.log.Infof("Starting fleet in background...")
		// Determine version of fleet to use
		var runCommand string
		version := Options.GetValue(o.values, VersionOption)
		if version == "latest" {
			runCommand = fmt.Sprintf(
				"%s launch workspace -- --projectDir '%s' --cache-path '%s' --auth=accept-everyone --publish --enableSmartMode",
				binaryPath, projectDir, location,
			)
		} else {
			runCommand = fmt.Sprintf(
				"%s launch workspace --workspace-version %s -- --projectDir '%s' --cache-path '%s' --auth=accept-everyone --publish --enableSmartMode",
				binaryPath, version, projectDir, location,
			)
		}

		args := []string{}
		if o.userName != "" {
			args = append(args, "su", o.userName, "-c", runCommand)
		} else {
			args = append(args, "sh", "-c", runCommand)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = location
		var err error
		readCloser, err = cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		cmd.Stderr = stderrBuffer
		wasStarted = true
		return cmd, nil
	})
	if err != nil {
		return err
	} else if !wasStarted {
		return nil
	}
	defer readCloser.Close()

	// wait for the jet brains url and then exit
	o.log.Infof("Waiting for fleet to start...")
	s := scanner.NewScanner(readCloser)
	stdoutBuffer := &bytes.Buffer{}
	for s.Scan() {
		text := s.Text()
		if strings.Contains(text, "https://fleet.jetbrains.com/") {
			index := strings.Index(text, "https://fleet.jetbrains.com/")
			err = os.WriteFile(FleetURLFile, []byte(strings.TrimSpace(text[index:])), 0600)
			if err != nil {
				return err
			}

			o.log.Infof("Fleet has successfully started")
			return o.startMonitor()
		} else {
			_, _ = stdoutBuffer.Write([]byte(text + "\n"))
		}
	}

	return fmt.Errorf("seems like there was an error starting up fleet: %s%s", stdoutBuffer.String(), stderrBuffer.String())
}

func (o *FleetServer) startMonitor() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	return single.Single("fleet-monitor.pid", func() (*exec.Cmd, error) {
		o.log.Infof("Starting fleet monitor in background...")
		runCommand := fmt.Sprintf("%s helper fleet-server --workspaceid %s", self, "test")
		args := []string{}
		if o.userName != "" {
			args = append(args, "su", o.userName, "-c", runCommand)
		} else {
			args = append(args, "sh", "-c", runCommand)
		}
		cmd := exec.Command(args[0], args[1:]...)
		return cmd, nil
	})
}

func prepareFleetServerLocation(userName string) (string, error) {
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

	folder := filepath.Join(homeFolder, ".fleet-server")
	err = os.MkdirAll(folder, 0755)
	if err != nil {
		return "", err
	}

	return folder, nil
}
