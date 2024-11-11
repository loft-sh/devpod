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
	// general
	DEVPOD      = "DEVPOD"
	DEVPOD_OS   = "DEVPOD_OS"
	DEVPOD_ARCH = "DEVPOD_ARCH"

	// workspace
	WORKSPACE_ID       = "WORKSPACE_ID"
	WORKSPACE_UID      = "WORKSPACE_UID"
	WORKSPACE_PICTURE  = "WORKSPACE_PICTURE"
	WORKSPACE_FOLDER   = "WORKSPACE_FOLDER"
	WORKSPACE_CONTEXT  = "WORKSPACE_CONTEXT"
	WORKSPACE_ORIGIN   = "WORKSPACE_ORIGIN"
	WORKSPACE_SOURCE   = "WORKSPACE_SOURCE"
	WORKSPACE_PROVIDER = "WORKSPACE_PROVIDER"

	// machine
	MACHINE_ID       = "MACHINE_ID"
	MACHINE_CONTEXT  = "MACHINE_CONTEXT"
	MACHINE_FOLDER   = "MACHINE_FOLDER"
	MACHINE_PROVIDER = "MACHINE_PROVIDER"

	// provider
	PROVIDER_ID      = "PROVIDER_ID"
	PROVIDER_CONTEXT = "PROVIDER_CONTEXT"
	PROVIDER_FOLDER  = "PROVIDER_FOLDER"

	// pro
	LOFT_PROJECT         = "LOFT_PROJECT"
	LOFT_FILTER_BY_OWNER = "LOFT_FILTER_BY_OWNER"
)

const (
	DEVCONTAINER_ID = "DEVCONTAINER_ID"
)

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
		if workspace.UID != "" {
			retVars[WORKSPACE_UID] = workspace.UID
		}
		retVars[WORKSPACE_FOLDER], _ = GetWorkspaceDir(workspace.Context, workspace.ID)
		retVars[WORKSPACE_FOLDER] = filepath.ToSlash(retVars[WORKSPACE_FOLDER])
		if workspace.Context != "" {
			retVars[WORKSPACE_CONTEXT] = workspace.Context
			retVars[MACHINE_CONTEXT] = workspace.Context
		}
		if workspace.Origin != "" {
			retVars[WORKSPACE_ORIGIN] = filepath.ToSlash(workspace.Origin)
		}
		if workspace.Picture != "" {
			retVars[WORKSPACE_PICTURE] = workspace.Picture
		}
		retVars[WORKSPACE_SOURCE] = workspace.Source.String()
		if workspace.Provider.Name != "" {
			retVars[WORKSPACE_PROVIDER] = workspace.Provider.Name
		}
		if workspace.Machine.ID != "" {
			retVars[MACHINE_ID] = workspace.Machine.ID
			machineDir, _ := GetMachineDir(workspace.Context, workspace.Machine.ID)
			retVars[MACHINE_FOLDER] = filepath.ToSlash(machineDir)
		}
		if workspace.Pro != nil && workspace.Pro.Project != "" {
			retVars[LOFT_PROJECT] = workspace.Pro.Project
		}
		for k, v := range GetBaseEnvironment(workspace.Context, workspace.Provider.Name) {
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
		retVars[MACHINE_FOLDER], _ = GetMachineDir(machine.Context, machine.ID)
		retVars[MACHINE_FOLDER] = filepath.ToSlash(retVars[MACHINE_FOLDER])
		if machine.Context != "" {
			retVars[MACHINE_CONTEXT] = machine.Context
		}
		if machine.Provider.Name != "" {
			retVars[MACHINE_PROVIDER] = machine.Provider.Name
		}
		for k, v := range GetBaseEnvironment(machine.Context, machine.Provider.Name) {
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

func GetBaseEnvironment(context, provider string) map[string]string {
	retVars := map[string]string{}

	// devpod binary
	devPodBinary, _ := os.Executable()
	retVars[DEVPOD] = filepath.ToSlash(devPodBinary)
	retVars[DEVPOD_OS] = runtime.GOOS
	retVars[DEVPOD_ARCH] = runtime.GOARCH
	retVars[PROVIDER_ID] = provider
	retVars[PROVIDER_CONTEXT] = context
	providerFolder, _ := GetProviderDir(context, provider)
	retVars[PROVIDER_FOLDER] = filepath.ToSlash(providerFolder)
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

func CloneAgentWorkspaceInfo(agentWorkspaceInfo *AgentWorkspaceInfo) *AgentWorkspaceInfo {
	if agentWorkspaceInfo == nil {
		return nil
	}
	out, _ := json.Marshal(agentWorkspaceInfo)
	ret := &AgentWorkspaceInfo{}
	_ = json.Unmarshal(out, ret)
	ret.Origin = agentWorkspaceInfo.Origin
	ret.Workspace = CloneWorkspace(agentWorkspaceInfo.Workspace)
	ret.Machine = CloneMachine(agentWorkspaceInfo.Machine)
	return ret
}

func CloneWorkspace(workspace *Workspace) *Workspace {
	if workspace == nil {
		return nil
	}
	out, _ := json.Marshal(workspace)
	ret := &Workspace{}
	_ = json.Unmarshal(out, ret)
	ret.Origin = workspace.Origin
	return ret
}

func CloneMachine(server *Machine) *Machine {
	if server == nil {
		return nil
	}
	out, _ := json.Marshal(server)
	ret := &Machine{}
	_ = json.Unmarshal(out, ret)
	ret.Origin = server.Origin
	return ret
}
