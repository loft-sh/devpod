package openvscode

import (
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/command"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const OpenVSCodeDownloadAmd64 = "https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-v1.75.1/openvscode-server-v1.75.1-linux-x64.tar.gz"
const OpenVSCodeDownloadArm64 = "https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-v1.75.1/openvscode-server-v1.75.1-linux-arm64.tar.gz"

const DefaultVSCodePort = 10800

func NewOpenVSCodeServer(extensions []string, settings string, userName string, host, port string, log log.Logger) ide.IDE {
	return &openVSCodeServer{
		extensions: extensions,
		settings:   settings,
		userName:   userName,
		host:       host,
		port:       port,
		log:        log,
	}
}

type openVSCodeServer struct {
	extensions []string
	settings   string
	userName   string
	host       string
	port       string
	log        log.Logger
}

func (o *openVSCodeServer) Install() error {
	err := o.install()
	if err != nil {
		return err
	}

	err = o.Start()
	if err != nil {
		return err
	}

	return nil
}

func (o *openVSCodeServer) install() error {
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
	url := OpenVSCodeDownloadAmd64
	if runtime.GOARCH == "arm64" {
		url = OpenVSCodeDownloadArm64
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

	// install extensions
	err = o.InstallExtensions()
	if err != nil {
		return errors.Wrap(err, "install extensions")
	}

	// paste settings
	err = o.InstallSettings()
	if err != nil {
		return errors.Wrap(err, "install settings")
	}

	return nil
}

func (o *openVSCodeServer) InstallExtensions() error {
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
		}
		o.log.Info("Successfully installed extension " + extension)
	}

	return nil
}

func (o *openVSCodeServer) InstallSettings() error {
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

	err = copy2.ChownR(settingsDir, o.userName)
	if err != nil {
		return err
	}

	return nil
}

func (o *openVSCodeServer) Start() error {
	isRunning, markerFile := singleProcess(o.log)
	if isRunning {
		o.log.Debugf("OpenVSCode is already started")
		return nil
	}

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
	err = cmd.Start()
	if err != nil {
		return err
	}

	// wait until we have a process id
	for cmd.Process.Pid < 0 {
		time.Sleep(time.Millisecond)
	}

	// write pid to file
	err = os.WriteFile(markerFile, []byte(strconv.Itoa(cmd.Process.Pid)), os.ModePerm)
	if err != nil {
		return err
	}

	// release process resources
	err = cmd.Process.Release()
	if err != nil {
		return err
	}

	o.log.Infof("Successfully started openvscode...")
	return nil
}

func singleProcess(log log.Logger) (bool, string) {
	// check if marker file is there
	markerFile := filepath.Join(os.TempDir(), "openvscode.pid")
	pid, err := os.ReadFile(markerFile)
	if err != nil {
		return false, markerFile
	}

	// check if process id exists
	isRunning, err := command.IsRunning(string(pid))
	if err != nil {
		log.Debugf("Error retrieving running status: %v", err)
		return false, markerFile
	}

	return isRunning, markerFile
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
