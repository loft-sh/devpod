package agent

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/loft-sh/api/v4/pkg/devpod"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	"github.com/takama/daemon"
)

type SshConfig struct {
	Workdir string `json:"workdir,omitempty"`
	User    string `json:"user,omitempty"`
}

type DaemonConfig struct {
	Platform devpod.PlatformOptions `json:"platform,omitempty"`
	Ssh      SshConfig              `json:"ssh,omitempty"`
	Timeout  string                 `json:"timeout"`
}

func BuildWorkspaceDaemonConfig(platformOptions devpod.PlatformOptions, workspaceConfig *provider2.Workspace, substitutionContext *config.SubstitutionContext, mergedConfig *config.MergedDevContainerConfig) (*DaemonConfig, error) {
	var workdir string
	if workspaceConfig.Source.GitSubPath != "" {
		substitutionContext.ContainerWorkspaceFolder = filepath.Join(substitutionContext.ContainerWorkspaceFolder, workspaceConfig.Source.GitSubPath)
		workdir = substitutionContext.ContainerWorkspaceFolder
	}
	if workdir == "" && mergedConfig != nil {
		workdir = mergedConfig.WorkspaceFolder
	}
	if workdir == "" && substitutionContext != nil {
		workdir = substitutionContext.ContainerWorkspaceFolder
	}

	// Get remote user; default to "root" if empty.
	user := mergedConfig.RemoteUser
	if user == "" {
		user = "root"
	}

	// build info isn't required in the workspace and can be omitted
	platformOptions.Build = nil

	daemonConfig := &DaemonConfig{
		Platform: platformOptions,
		Ssh: SshConfig{
			Workdir: workdir,
			User:    user,
		},
	}

	return daemonConfig, nil
}

func GetEncodedWorkspaceDaemonConfig(platformOptions devpod.PlatformOptions, workspaceConfig *provider2.Workspace, substitutionContext *config.SubstitutionContext, mergedConfig *config.MergedDevContainerConfig) (string, error) {
	daemonConfig, err := BuildWorkspaceDaemonConfig(platformOptions, workspaceConfig, substitutionContext, mergedConfig)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(daemonConfig)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}

func InstallDaemon(agentDir string, interval string, log log.Logger) error {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return fmt.Errorf("unsupported daemon os")
	}

	// check if admin
	service, err := daemon.New("devpod", "DevPod Agent Service", daemon.SystemDaemon)
	if err != nil {
		return err
	}

	// install ourselves with devpod watch
	args := []string{"agent", "daemon"}
	if agentDir != "" {
		args = append(args, "--agent-dir", agentDir)
	}
	if interval != "" {
		args = append(args, "--interval", interval)
	}
	_, err = service.Install(args...)
	if err != nil && !errors.Is(err, daemon.ErrAlreadyInstalled) {
		return perrors.Wrap(err, "install service")
	}

	// make sure daemon is started
	_, err = service.Start()
	if err != nil && !errors.Is(err, daemon.ErrAlreadyRunning) {
		log.Warnf("Error starting service: %v", err)

		err = single.Single("daemon.pid", func() (*exec.Cmd, error) {
			executable, err := os.Executable()
			if err != nil {
				return nil, err
			}

			log.Infof("Successfully started DevPod daemon into server")
			return exec.Command(executable, args...), nil
		})
		if err != nil {
			return fmt.Errorf("start daemon: %w", err)
		}
	} else if err == nil {
		log.Infof("Successfully installed DevPod daemon into server")
	}

	return nil
}

func RemoveDaemon() error {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return fmt.Errorf("unsupported daemon os")
	}

	// check if admin
	service, err := daemon.New("devpod", "DevPod Agent Service", daemon.SystemDaemon)
	if err != nil {
		return err
	}

	// remove daemon
	_, err = service.Remove()
	if err != nil && !errors.Is(err, daemon.ErrNotInstalled) {
		return err
	}

	return nil
}
