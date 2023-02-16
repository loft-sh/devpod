package vscode

import (
	"crypto/tls"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

const OpenVSCodeDownloadAmd64 = "https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-v1.75.1/openvscode-server-v1.75.1-linux-x64.tar.gz"
const OpenVSCodeDownloadArm64 = "https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-v1.75.1/openvscode-server-v1.75.1-linux-arm64.tar.gz"

const DefaultVSCodePort = 10800

type OpenVSCodeServer struct{}

func (o *OpenVSCodeServer) InstallAndStart(user, host, port string, out io.Writer) error {
	err := o.Install(user)
	if err != nil {
		return err
	}

	return o.Start(user, host, port, out)
}

func (o *OpenVSCodeServer) Install(userName string) error {
	location, err := prepareOpenVSCodeServerLocation(userName)
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

	if userName != "" {
		userId, err := user.Lookup(userName)
		if err != nil {
			return errors.Wrap(err, "lookup user")
		}

		uid, _ := strconv.Atoi(userId.Uid)
		gid, _ := strconv.Atoi(userId.Gid)
		err = ChownR(location, uid, gid)
		if err != nil {
			return errors.Wrap(err, "chown")
		}
	}

	return nil
}

func (o *OpenVSCodeServer) Start(userName, host, port string, out io.Writer) error {
	location, err := prepareOpenVSCodeServerLocation(userName)
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
		"--extensions-dir", filepath.Join(location, "extensions-data"),
		"--server-data-dir", filepath.Join(location, "server-data"),
		"--user-data-dir", filepath.Join(location, "user-data"),
		"--host", host,
		"--port", port,
	}
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = location
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = append(cmd.Env, os.Environ()...)
	command.AsUser(userName, cmd)
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func ChownR(path string, uid, gid int) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, uid, gid)
		}
		return err
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
