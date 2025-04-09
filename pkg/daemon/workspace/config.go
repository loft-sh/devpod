package workspace

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"

	"github.com/loft-sh/api/v4/pkg/devpod"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
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
