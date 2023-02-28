package provider

import (
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/config"
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
	SERVER_ID                = "SERVER_ID"
	SERVER_CONTEXT           = "SERVER_CONTEXT"
	SERVER_FOLDER            = "SERVER_FOLDER"
	SERVER_PROVIDER          = "SERVER_PROVIDER"
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
		Server: WorkspaceServerConfig{
			ID: os.Getenv(SERVER_ID),
		},
		Provider: WorkspaceProviderConfig{
			Name: os.Getenv(WORKSPACE_PROVIDER),
		},
		Context: os.Getenv(WORKSPACE_CONTEXT),
		Origin:  os.Getenv(WORKSPACE_ORIGIN),
	}
}

func ToOptions(workspace *Workspace, server *Server, options map[string]config.OptionValue) map[string]string {
	retVars := map[string]string{}
	for optionName, optionValue := range options {
		retVars[strings.ToUpper(optionName)] = optionValue.Value
	}
	if workspace != nil {
		if workspace.ID != "" {
			retVars[WORKSPACE_ID] = workspace.ID
		}
		if workspace.Folder != "" {
			retVars[WORKSPACE_FOLDER] = filepath.ToSlash(workspace.Folder)
		}
		if workspace.Context != "" {
			retVars[WORKSPACE_CONTEXT] = workspace.Context
			retVars[SERVER_CONTEXT] = workspace.Context
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
		if workspace.Server.ID != "" {
			retVars[SERVER_ID] = workspace.Server.ID
			retVars[SERVER_FOLDER], _ = GetServerDir(workspace.Context, workspace.Server.ID)
		}
	}
	if server != nil {
		if server.ID != "" {
			retVars[SERVER_ID] = server.ID
		}
		if server.Folder != "" {
			retVars[SERVER_FOLDER] = filepath.ToSlash(server.Folder)
		}
		if server.Context != "" {
			retVars[SERVER_CONTEXT] = server.Context
		}
		if server.Provider.Name != "" {
			retVars[SERVER_PROVIDER] = server.Provider.Name
		}
	}

	// devpod binary
	devPodBinary, _ := os.Executable()
	retVars[DEVPOD] = filepath.ToSlash(devPodBinary)
	return retVars
}

func GetProviderOptions(workspace *Workspace, server *Server, devConfig *config.Config) map[string]config.OptionValue {
	retValues := map[string]config.OptionValue{}
	providerName := ""
	if workspace != nil {
		providerName = workspace.Provider.Name
	}
	if server != nil {
		providerName = server.Provider.Name
	}
	if devConfig != nil && providerName != "" {
		for k, v := range devConfig.Current().ProviderOptions(providerName) {
			retValues[k] = v
		}
	}
	return retValues
}

func CloneWorkspace(workspace *Workspace) *Workspace {
	out, _ := json.Marshal(workspace)
	ret := &Workspace{}
	_ = json.Unmarshal(out, ret)
	ret.Origin = workspace.Origin
	return ret
}

func CloneServer(server *Server) *Server {
	out, _ := json.Marshal(server)
	ret := &Server{}
	_ = json.Unmarshal(out, ret)
	ret.Origin = server.Origin
	return ret
}
