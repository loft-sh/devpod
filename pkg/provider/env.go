package provider

import (
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/compress"
	"os"
	"path/filepath"
	"strings"
)

const (
	DEVPOD                   = "DEVPOD"
	WORKSPACE_ID             = "WORKSPACE_ID"
	WORKSPACE_FOLDER         = "WORKSPACE_FOLDER"
	WORKSPACE_CONTEXT        = "WORKSPACE_CONTEXT"
	WORKSPACE_ORIGIN         = "WORKSPACE_ORIGIN"
	WORKSPACE_GIT_REPOSITORY = "WORKSPACE_GIT_REPOSITORY"
	WORKSPACE_GIT_BRANCH     = "WORKSPACE_GIT_BRANCH"
	WORKSPACE_GIT_COMMIT     = "WORKSPACE_GIT_COMMIT"
	WORKSPACE_LOCAL_FOLDER   = "WORKSPACE_LOCAL_FOLDER"
	WORKSPACE_IMAGE          = "WORKSPACE_IMAGE"
	WORKSPACE_PROVIDER       = "WORKSPACE_PROVIDER"
)

func FromEnvironment() *Workspace {
	return &Workspace{
		ID:     os.Getenv(WORKSPACE_ID),
		Folder: os.Getenv(WORKSPACE_FOLDER),
		Source: WorkspaceSource{
			GitRepository: os.Getenv(WORKSPACE_GIT_REPOSITORY),
			GitBranch:     os.Getenv(WORKSPACE_GIT_BRANCH),
			GitCommit:     os.Getenv(WORKSPACE_GIT_COMMIT),
			LocalFolder:   os.Getenv(WORKSPACE_LOCAL_FOLDER),
			Image:         os.Getenv(WORKSPACE_IMAGE),
		},
		Provider: WorkspaceProviderConfig{
			Name: os.Getenv(WORKSPACE_PROVIDER),
		},
		Context: os.Getenv(WORKSPACE_CONTEXT),
		Origin:  os.Getenv(WORKSPACE_ORIGIN),
	}
}

func ToOptions(workspace *Workspace) (map[string]string, error) {
	retVars := map[string]string{}
	if workspace == nil {
		return retVars, nil
	}
	for optionName, optionValue := range workspace.Provider.Options {
		retVars[strings.ToUpper(optionName)] = optionValue.Value
	}
	if workspace.ID != "" {
		retVars[WORKSPACE_ID] = workspace.ID
	}
	if workspace.Folder != "" {
		retVars[WORKSPACE_FOLDER] = workspace.Folder
	}
	if workspace.Context != "" {
		retVars[WORKSPACE_CONTEXT] = workspace.Context
	}
	if workspace.Origin != "" {
		retVars[WORKSPACE_ORIGIN] = workspace.Origin
	}
	if workspace.Source.LocalFolder != "" {
		retVars[WORKSPACE_LOCAL_FOLDER] = workspace.Source.LocalFolder
	}
	if workspace.Source.GitRepository != "" {
		retVars[WORKSPACE_GIT_REPOSITORY] = workspace.Source.GitRepository
	}
	if workspace.Source.GitBranch != "" {
		retVars[WORKSPACE_GIT_BRANCH] = workspace.Source.GitBranch
	}
	if workspace.Source.GitCommit != "" {
		retVars[WORKSPACE_GIT_COMMIT] = workspace.Source.GitCommit
	}
	if workspace.Source.Image != "" {
		retVars[WORKSPACE_IMAGE] = workspace.Source.Image
	}
	if workspace.Provider.Name != "" {
		retVars[WORKSPACE_PROVIDER] = workspace.Provider.Name
	}

	// devpod binary
	devPodBinary, _ := os.Executable()
	retVars[DEVPOD] = filepath.ToSlash(devPodBinary)
	return retVars, nil
}

func NewAgentWorkspaceInfo(workspace *Workspace) (string, error) {
	// trim options that don't exist
	workspace = CloneWorkspace(workspace)
	if workspace.Provider.Options != nil {
		for name, option := range workspace.Provider.Options {
			if option.Local {
				delete(workspace.Provider.Options, name)
			}
		}
	}

	// marshal config
	out, err := json.Marshal(&AgentWorkspaceInfo{
		Workspace: *workspace,
	})
	if err != nil {
		return "", err
	}

	return compress.Compress(string(out))
}

func CloneWorkspace(workspace *Workspace) *Workspace {
	out, _ := json.Marshal(workspace)
	ret := &Workspace{}
	_ = json.Unmarshal(out, ret)
	ret.Origin = workspace.Origin
	return ret
}

func ToEnvironment(workspace *Workspace) ([]string, error) {
	options, err := ToOptions(workspace)
	if err != nil {
		return nil, err
	}
	retVars := []string{}
	for k, v := range options {
		retVars = append(retVars, k+"="+v)
	}
	return retVars, nil
}
