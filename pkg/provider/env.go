package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/config"
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

func combineOptions(resolvedOptions map[string]config.OptionValue, otherOptions map[string]config.OptionValue) map[string]config.OptionValue {
	options := map[string]config.OptionValue{}
	for k, v := range resolvedOptions {
		options[k] = v
	}
	for k, v := range otherOptions {
		options[k] = v
	}
	return options
}

func ToEnvironment(workspace *Workspace, machine *Machine, options map[string]config.OptionValue, extraEnv map[string]string) []string {
	env := ToOptions(workspace, machine, options)

	// create environment variables for command
	osEnviron := os.Environ()
	for k, v := range env {
		osEnviron = append(osEnviron, k+"="+v)
	}
	for k, v := range extraEnv {
		osEnviron = append(osEnviron, k+"="+v)
	}

	return osEnviron
}

func CombineOptions(workspace *Workspace, machine *Machine, options map[string]config.OptionValue) map[string]config.OptionValue {
	providerOptions := map[string]config.OptionValue{}
	if options != nil {
		providerOptions = combineOptions(providerOptions, options)
	}
	if workspace != nil {
		providerOptions = combineOptions(providerOptions, workspace.Provider.Options)
	}
	if machine != nil {
		providerOptions = combineOptions(providerOptions, machine.Provider.Options)
	}
	return providerOptions
}

func ToOptionsWorkspace(workspace *Workspace) map[string]string {
	retVars := map[string]string{}
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
		for k, v := range GetBaseEnvironment() {
			retVars[k] = v
		}
	}
	return retVars
}

func ToOptionsMachine(machine *Machine) map[string]string {
	retVars := map[string]string{}
	if machine != nil {
		if machine.ID != "" {
			retVars[MACHINE_ID] = machine.ID
		}
		if machine.Folder != "" {
			retVars[MACHINE_FOLDER] = filepath.ToSlash(machine.Folder)
		}
		if machine.Context != "" {
			retVars[MACHINE_CONTEXT] = machine.Context
		}
		if machine.Provider.Name != "" {
			retVars[MACHINE_PROVIDER] = machine.Provider.Name
		}
		for k, v := range GetBaseEnvironment() {
			retVars[k] = v
		}
	}
	return retVars
}

func ToOptions(workspace *Workspace, machine *Machine, options map[string]config.OptionValue) map[string]string {
	providerOptions := CombineOptions(workspace, machine, options)
	retVars := map[string]string{}
	for optionName, optionValue := range providerOptions {
		retVars[strings.ToUpper(optionName)] = optionValue.Value
	}

	retVars = Merge(retVars, ToOptionsWorkspace(workspace))
	retVars = Merge(retVars, ToOptionsMachine(machine))
	return retVars
}

func Merge(m1 map[string]string, m2 map[string]string) map[string]string {
	retMap := map[string]string{}
	for k, v := range m1 {
		retMap[k] = v
	}
	for k, v := range m2 {
		retMap[k] = v
	}

	return retMap
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
