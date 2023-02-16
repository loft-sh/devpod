package vscode

import (
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const VSCodeDownloadAmd64 = "https://aka.ms/vscode-server-launcher/x86_64-unknown-linux-gnu"
const VSCodeDownloadArm64 = "https://aka.ms/vscode-server-launcher/aarch64-unknown-linux-gnu"

type VSCodeServer struct{}

func (o *VSCodeServer) Install(extensions []string, settings string, userName string, out io.Writer) error {
	location, err := prepareVSCodeServerLocation(userName)
	if err != nil {
		return err
	}

	// is installed
	_, err = os.Stat(filepath.Join(location, "bin", "code-server"))
	if err == nil {
		return nil
	}

	// download
	binPath := filepath.Join(location, "bin", "code-server")
	err = DownloadVSCode(binPath)
	if err != nil {
		_ = os.RemoveAll(location)
		return err
	}

	// set settings
	settingsDir := filepath.Join(location, "data", "Machine")
	err = os.MkdirAll(settingsDir, 0777)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(settings), 0666)
	if err != nil {
		return err
	}

	// chown location
	if userName != "" {
		err = ChownR(location, userName)
		if err != nil {
			return errors.Wrap(err, "chown")
		}
	}

	// download extensions
	for _, extension := range extensions {
		fmt.Fprintln(out, "Install extension "+extension+"...")
		cmd := exec.Command(binPath, "serve-local", "--accept-server-license-terms", "--install-extension", extension)
		cmd.Stdout = out
		cmd.Stderr = out
		cmd.Env = os.Environ()
		command.AsUser(userName, cmd)
		err = cmd.Run()
		if err != nil {
			fmt.Fprintln(out, "Failed installing extension "+extension)
		}
		fmt.Fprintln(out, "Successfully installed extension "+extension)
	}

	return nil
}

func DownloadVSCode(binPath string) error {
	err := os.MkdirAll(filepath.Dir(binPath), 0777)
	if err != nil {
		return err
	}

	// check what release we need to download
	url := VSCodeDownloadAmd64
	if runtime.GOARCH == "arm64" {
		url = VSCodeDownloadArm64
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

func prepareVSCodeServerLocation(userName string) (string, error) {
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
	err = os.MkdirAll(folder, 0777)
	if err != nil {
		return "", err
	}

	return folder, nil
}
