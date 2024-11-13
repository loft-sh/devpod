package neovim

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/config"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/extract"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

const DefaultServerPort = 10720
const (
	DownLoadURL       = "DOWNLOAD_URL"
	BindAddressOption = "BIND_ADDRESS"
)

var Options = ide.Options{
	DownLoadURL: {
		Name:        DownLoadURL,
		Description: "URL to use to download nvim",
		Default:     "https://github.com/neovim/neovim/releases/latest/download/nvim-linux64.tar.gz",
	},
	BindAddressOption: {
		Name:        BindAddressOption,
		Description: "The address to bind the server to locally. E.g. 0.0.0.0:12345",
		Default:     "127.0.0.1:10720",
	},
}

// NewServer creates a new neovim server
func NewServer(userName string, values map[string]config.OptionValue, log log.Logger) *Server {
	return &Server{
		userName: userName,
		values:   values,
		log:      log,
	}
}

// Server provides the remote the ability to download, install and run the neovim server in headless mode
type Server struct {
	userName string
	values   map[string]config.OptionValue
	log      log.Logger
}

func (o *Server) Install(workspaceFolder string) error {
	o.log.Debugf("Setup neovim...")
	// Define out target install location and ensure it exists
	baseFolder, err := util.GetBaseFolder(o.userName)
	if err != nil {
		return err
	}
	targetLocation := path.Join(baseFolder, ".cache", "neovim")

	_, err = os.Stat(targetLocation)
	if err != nil {
		o.log.Debugf("Installing neovim")
		// Download and extract neovim
		o.log.Debugf("Download neovim archive")
		archivePath, err := o.download("/var/devpod/neovim", o.log)
		if err != nil {
			return err
		}
		o.log.Infof("Extract neovim...")
		err = o.extractArchive(archivePath, targetLocation)
		if err != nil {
			return err
		}
		// Ensure the remote user owns the neovim install
		err = copy2.ChownR(path.Join(baseFolder, ".cache"), o.userName)
		if err != nil {
			return errors.Wrap(err, "chown")
		}
	}

	return o.start(targetLocation, workspaceFolder)
}

func (o *Server) extractArchive(fromPath string, toPath string) error {
	file, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return extract.Extract(file, toPath, extract.StripLevels(1))
}

func (o *Server) download(targetFolder string, log log.Logger) (string, error) {
	// Ensure the target folder exists
	err := os.MkdirAll(targetFolder, os.ModePerm)
	if err != nil {
		return "", err
	}
	downloadURL := Options.GetValue(o.values, DownLoadURL)
	targetPath := path.Join(filepath.ToSlash(targetFolder), "nvim-linux64.tar.gz")

	// initiate download
	log.Infof("Download neovim from %s", downloadURL)
	defer log.Debugf("Successfully downloaded neovim")
	resp, err := devpodhttp.GetHTTPClient().Get(downloadURL)
	if err != nil {
		return "", errors.Wrap(err, "download binary")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", errors.Wrapf(err, "download binary returned status code %d", resp.StatusCode)
	}
	stat, err := os.Stat(targetPath)
	if err == nil && stat.Size() == resp.ContentLength {
		return targetPath, nil
	}
	// Download the response as a file
	file, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, util.NewProgressReader(resp, log))
	if err != nil {
		return "", errors.Wrap(err, "download file")
	}
	return targetPath, nil
}

// start runs the neovim server in headless mode using a known PID file to expose at most one instance
func (o *Server) start(targetLocation, workspaceFolder string) error {
	return single.Single("nvim.pid", func() (*exec.Cmd, error) {
		o.log.Debug("Starting nvim in background...")
		// Generate server start command using remote user
		addr := Options.GetValue(o.values, BindAddressOption)
		runCommand := fmt.Sprintf("%s/bin/nvim --listen %s --headless %s", targetLocation, addr, workspaceFolder)
		args := []string{}
		if o.userName != "" {
			args = append(args, "su", o.userName, "-l", "-c", runCommand)
		} else {
			args = append(args, "sh", "-l", "-c", runCommand)
		}
		// Execute the command in the workspace folder
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = workspaceFolder
		return cmd, nil
	})
}
