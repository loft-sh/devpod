package openvscode

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const DownloadAmd64Template = "https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-%s/openvscode-server-%s-linux-x64.tar.gz"
const DownloadArm64Template = "https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-%s/openvscode-server-%s-linux-arm64.tar.gz"

const (
	OpenOption          = "OPEN"
	VersionOption       = "VERSION"
	DownloadAmd64Option = "DOWNLOAD_AMD64"
	DownloadArm64Option = "DOWNLOAD_ARM64"
)

var Options = ide.Options{
	VersionOption: {
		Name:        VersionOption,
		Description: "The version for the open vscode binary",
		Default:     "v1.78.2",
	},
	OpenOption: {
		Name:        OpenOption,
		Description: "If DevPod should automatically open the browser",
		Default:     "true",
		Enum: []string{
			"true",
			"false",
		},
	},
	DownloadArm64Option: {
		Name:        DownloadArm64Option,
		Description: "The download url for the arm64 vscode server binary",
	},
	DownloadAmd64Option: {
		Name:        DownloadAmd64Option,
		Description: "The download url for the amd64 vscode server binary",
	},
}

const DefaultVSCodePort = 10800

func NewOpenVSCodeServer(extensions []string, settings string, userName string, host, port string, values map[string]config.OptionValue, log log.Logger) *OpenVSCodeServer {
	return &OpenVSCodeServer{
		values:     values,
		extensions: extensions,
		settings:   settings,
		userName:   userName,
		host:       host,
		port:       port,
		log:        log,
	}
}

type OpenVSCodeServer struct {
	values     map[string]config.OptionValue
	extensions []string
	settings   string
	userName   string
	host       string
	port       string
	log        log.Logger
}

func (o *OpenVSCodeServer) InstallExtensions() error {
	// install extensions
	err := o.installExtensions()
	if err != nil {
		return errors.Wrap(err, "install extensions")
	}

	return nil
}

func (o *OpenVSCodeServer) Install() error {
	location, err := prepareOpenVSCodeServerLocation(o.userName)
	if err != nil {
		return err
	}

	// is installed
	_, err = os.Stat(filepath.Join(location, "bin"))
	if err == nil {
		return nil
	}

	// check what release we need to download
	var url string
	version := Options.GetValue(o.values, VersionOption)
	if runtime.GOARCH == "arm64" {
		url = Options.GetValue(o.values, DownloadArm64Option)
		if url == "" {
			url = fmt.Sprintf(DownloadArm64Template, version, version)
		}
	} else {
		url = Options.GetValue(o.values, DownloadAmd64Option)
		if url == "" {
			url = fmt.Sprintf(DownloadAmd64Template, version, version)
		}
	}

	// download tar
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

	err = extract.Extract(resp.Body, location, extract.StripLevels(1))
	if err != nil {
		return errors.Wrap(err, "extract vscode")
	}

	// chown location
	if o.userName != "" {
		err = copy2.ChownR(location, o.userName)
		if err != nil {
			return errors.Wrap(err, "chown")
		}
	}

	// paste settings
	err = o.installSettings()
	if err != nil {
		return errors.Wrap(err, "install settings")
	}

	return nil
}

func (o *OpenVSCodeServer) installExtensions() error {
	if len(o.extensions) == 0 {
		return nil
	}

	location, err := prepareOpenVSCodeServerLocation(o.userName)
	if err != nil {
		return err
	}

	out := o.log.Writer(logrus.InfoLevel, false)
	defer out.Close()

	binaryPath := filepath.Join(location, "bin", "openvscode-server")
	for _, extension := range o.extensions {
		o.log.Info("Install extension " + extension + "...")
		runCommand := fmt.Sprintf("%s --install-extension '%s'", binaryPath, extension)
		args := []string{}
		if o.userName != "" {
			args = append(args, "su", o.userName, "-c", runCommand)
		} else {
			args = append(args, "sh", "-c", runCommand)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = out
		cmd.Stderr = out
		err = cmd.Run()
		if err != nil {
			o.log.Info("Failed installing extension " + extension)
		} else {
			o.log.Info("Successfully installed extension " + extension)
		}
	}

	return nil
}

func (o *OpenVSCodeServer) installSettings() error {
	if len(o.settings) == 0 {
		return nil
	}

	location, err := prepareOpenVSCodeServerLocation(o.userName)
	if err != nil {
		return err
	}

	settingsDir := filepath.Join(location, "data", "Machine")
	err = os.MkdirAll(settingsDir, 0777)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(o.settings), 0666)
	if err != nil {
		return err
	}

	err = copy2.ChownR(location, o.userName)
	if err != nil {
		return err
	}

	return nil
}

func (o *OpenVSCodeServer) Start() error {
	location, err := prepareOpenVSCodeServerLocation(o.userName)
	if err != nil {
		return err
	}

	if o.host == "" {
		o.host = "0.0.0.0"
	}
	if o.port == "" {
		o.port = strconv.Itoa(DefaultVSCodePort)
	}

	binaryPath := filepath.Join(location, "bin", "openvscode-server")
	_, err = os.Stat(binaryPath)
	if err != nil {
		return errors.Wrap(err, "find binary")
	}

	return single.Single("openvscode.pid", func() (*exec.Cmd, error) {
		o.log.Infof("Starting openvscode in background...")
		runCommand := fmt.Sprintf("%s server-local --without-connection-token --host '%s' --port '%s'", binaryPath, o.host, o.port)
		args := []string{}
		if o.userName != "" {
			args = append(args, "su", o.userName, "-c", runCommand)
		} else {
			args = append(args, "sh", "-c", runCommand)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = location
		return cmd, nil
	})
}

func prepareOpenVSCodeServerLocation(userName string) (string, error) {
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

	folder := filepath.Join(homeFolder, ".openvscode-server")
	err = os.MkdirAll(folder, 0777)
	if err != nil {
		return "", err
	}

	return folder, nil
}
