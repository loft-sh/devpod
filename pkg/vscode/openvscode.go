package vscode

import (
	"crypto/tls"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

const OpenVSCodeDownloadAmd64 = "https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-v1.75.1/openvscode-server-v1.75.1-linux-x64.tar.gz"
const OpenVSCodeDownloadArm64 = "https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-v1.75.1/openvscode-server-v1.75.1-linux-arm64.tar.gz"

const DefaultVSCodePort = 10800

type OpenVSCodeServer struct{}

func (o *OpenVSCodeServer) InstallAndStart(host, port string, out io.Writer) error {
	err := o.Install()
	if err != nil {
		return err
	}

	return o.Start(host, port, out)
}

func (o *OpenVSCodeServer) Install() error {
	location, err := prepareOpenVSCodeServerLocation()
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

	return nil
}

func (o *OpenVSCodeServer) Start(host, port string, out io.Writer) error {
	location, err := prepareOpenVSCodeServerLocation()
	if err != nil {
		return err
	}

	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = strconv.Itoa(DefaultVSCodePort)
	}

	binaryPath := filepath.Join(location, "bin", "openvscode-server")
	_, err = os.Stat(binaryPath)
	if err != nil {
		return errors.Wrap(err, "find binary")
	}

	args := []string{
		"server-local",
		"--without-connection-token",
		"--host", host,
		"--port", port,
	}
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = out
	cmd.Stderr = out
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func prepareOpenVSCodeServerLocation() (string, error) {
	homeFolder, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	folder := filepath.Join(homeFolder, ".openvscodeserver")
	err = os.MkdirAll(folder, 0755)
	if err != nil {
		return "", err
	}

	return folder, nil
}
