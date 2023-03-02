package provider

import (
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/config"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DEVPOD                   = "DEVPOD"
	DEVPOD_OS                = "DEVPOD_OS"
	DEVPOD_ARCH              = "DEVPOD_ARCH"
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
	MACHINE_ID               = "MACHINE_ID"
	MACHINE_CONTEXT          = "MACHINE_CONTEXT"
	MACHINE_FOLDER           = "MACHINE_FOLDER"
	MACHINE_PROVIDER         = "MACHINE_PROVIDER"
)

func FromEnvironment() *Machine {
	return &Machine{
		ID:     os.Getenv(MACHINE_ID),
		Folder: os.Getenv(MACHINE_FOLDER),
		Provider: MachineProviderConfig{
			Name: os.Getenv(MACHINE_PROVIDER),
		},
		Context: os.Getenv(MACHINE_CONTEXT),
	}
}

func ToOptions(workspace *Workspace, server *Machine, options map[string]config.OptionValue) map[string]string {
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
			retVars[MACHINE_CONTEXT] = workspace.Context
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
		if workspace.Machine.ID != "" {
			retVars[MACHINE_ID] = workspace.Machine.ID
			retVars[MACHINE_FOLDER], _ = GetMachineDir(workspace.Context, workspace.Machine.ID)
		}
	}
	if server != nil {
		if server.ID != "" {
			retVars[MACHINE_ID] = server.ID
		}
		if server.Folder != "" {
			retVars[MACHINE_FOLDER] = filepath.ToSlash(server.Folder)
		}
		if server.Context != "" {
			retVars[MACHINE_CONTEXT] = server.Context
		}
		if server.Provider.Name != "" {
			retVars[MACHINE_PROVIDER] = server.Provider.Name
		}
	}
	for k, v := range GetBaseEnvironment() {
		retVars[k] = v
	}

	return retVars
}

func GetBaseEnvironment() map[string]string {
	retVars := map[string]string{}

	// devpod binary
	devPodBinary, _ := os.Executable()
	retVars[DEVPOD] = filepath.ToSlash(devPodBinary)
	retVars[DEVPOD_OS] = runtime.GOOS
	retVars[DEVPOD_ARCH] = runtime.GOARCH
	return retVars
}

func GetProviderOptions(workspace *Workspace, server *Machine, devConfig *config.Config) map[string]config.OptionValue {
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

func CloneMachine(server *Machine) *Machine {
	out, _ := json.Marshal(server)
	ret := &Machine{}
	_ = json.Unmarshal(out, ret)
	ret.Origin = server.Origin
	return ret
}
