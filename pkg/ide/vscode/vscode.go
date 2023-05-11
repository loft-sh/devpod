package vscode

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DownloadAmd64Option = "DOWNLOAD_AMD64"
	DownloadArm64Option = "DOWNLOAD_ARM64"
)

var Options = ide.Options{
	DownloadArm64Option: {
		Name:        DownloadArm64Option,
		Description: "The download url for the arm64 vscode server binary",
		Default:     "https://aka.ms/vscode-server-launcher/aarch64-unknown-linux-gnu",
	},
	DownloadAmd64Option: {
		Name:        DownloadAmd64Option,
		Description: "The download url for the amd64 vscode server binary",
		Default:     "https://aka.ms/vscode-server-launcher/x86_64-unknown-linux-gnu",
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
	binPath := filepath.Join(location, "bin", "code-server")

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

	// is installed
	_, err = os.Stat(filepath.Join(location, "bin", "code-server"))
	if err == nil {
		return nil
	}

	// download
	o.log.Info("Download vscode...")
	binPath := filepath.Join(location, "bin", "code-server")
	err = o.downloadVSCode(binPath)
	if err != nil {
		_ = os.RemoveAll(location)
		return err
	}
	o.log.Info("Successfully downloaded vscode")

	// set settings
	settingsDir := filepath.Join(location, "data", "Machine")
	err = os.MkdirAll(settingsDir, 0777)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(o.settings), 0666)
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

func (o *VsCodeServer) downloadVSCode(binPath string) error {
	err := os.MkdirAll(filepath.Dir(binPath), 0777)
	if err != nil {
		return err
	}

	// check what release we need to download
	url := Options.GetValue(o.values, DownloadAmd64Option)
	if runtime.GOARCH == "arm64" {
		url = Options.GetValue(o.values, DownloadArm64Option)
	}

	// download binary
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	outFile, err := os.Create(binPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Write the body to file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}

	// make file executable
	err = os.Chmod(binPath, 0777)
	if err != nil {
		return err
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
