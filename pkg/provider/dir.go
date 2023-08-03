package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/config"
)

const (
	WorkspaceConfigFile = "workspace.json"
	MachineConfigFile   = "machine.json"
	EngineConfigFile    = "engine.json"
	ProviderConfigFile  = "provider.json"
)

func GetEnginesDir(context string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "engines"), nil
}

func GetMachinesDir(context string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "machines"), nil
}

func GetLocksDir(context string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "locks"), nil
}

func GetWorkspacesDir(context string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "workspaces"), nil
}

func GetProvidersDir(context string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "providers"), nil
}

func GetProviderDir(context, providerName string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "providers", providerName), nil
}

func GetProviderBinariesDir(context, providerName string) (string, error) {
	providerDir, err := GetProviderDir(context, providerName)
	if err != nil {
		return "", err
	}

	return filepath.Join(providerDir, "binaries"), nil
}

func GetMachineDir(context, machineID string) (string, error) {
	if machineID == "" {
		return "", fmt.Errorf("machine id is empty")
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "machines", machineID), nil
}

func GetWorkspaceDir(context, workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", fmt.Errorf("workspace id is empty")
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "workspaces", workspaceID), nil
}

func GetEngineDir(context, engineID string) (string, error) {
	if engineID == "" {
		return "", fmt.Errorf("engine id is empty")
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "engines", engineID), nil
}

func WorkspaceExists(context, workspaceID string) bool {
	workspaceDir, err := GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return false
	}

	_, err = os.Stat(workspaceDir)
	return err == nil
}

func EngineExists(context, engineID string) bool {
	engineDir, err := GetEngineDir(context, engineID)
	if err != nil {
		return false
	}

	_, err = os.Stat(engineDir)
	return err == nil
}

func SaveProviderConfig(context string, provider *ProviderConfig) error {
	providerDir, err := GetProviderDir(context, provider.Name)
	if err != nil {
		return err
	}

	err = os.MkdirAll(providerDir, 0755)
	if err != nil {
		return err
	}

	providerDirBytes, err := json.Marshal(provider)
	if err != nil {
		return err
	}

	providerConfigFile := filepath.Join(providerDir, ProviderConfigFile)
	err = os.WriteFile(providerConfigFile, providerDirBytes, 0666)
	if err != nil {
		return err
	}

	return nil
}

func SaveEngineConfig(context string, engine *Engine) error {
	providerDir, err := GetEngineDir(context, engine.ID)
	if err != nil {
		return err
	}

	err = os.MkdirAll(providerDir, 0755)
	if err != nil {
		return err
	}

	engineBytes, err := json.Marshal(engine)
	if err != nil {
		return err
	}

	engineConfigFile := filepath.Join(providerDir, EngineConfigFile)
	err = os.WriteFile(engineConfigFile, engineBytes, 0666)
	if err != nil {
		return err
	}

	return nil
}

func SaveWorkspaceConfig(workspace *Workspace) error {
	workspaceDir, err := GetWorkspaceDir(workspace.Context, workspace.ID)
	if err != nil {
		return err
	}

	err = os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		return err
	}

	workspaceConfigBytes, err := json.Marshal(workspace)
	if err != nil {
		return err
	}

	workspaceConfigFile := filepath.Join(workspaceDir, WorkspaceConfigFile)
	err = os.WriteFile(workspaceConfigFile, workspaceConfigBytes, 0666)
	if err != nil {
		return err
	}

	return nil
}

func SaveMachineConfig(machine *Machine) error {
	machineDir, err := GetMachineDir(machine.Context, machine.ID)
	if err != nil {
		return err
	}

	err = os.MkdirAll(machineDir, 0755)
	if err != nil {
		return err
	}

	machineConfigBytes, err := json.Marshal(machine)
	if err != nil {
		return err
	}

	machineConfigFile := filepath.Join(machineDir, MachineConfigFile)
	err = os.WriteFile(machineConfigFile, machineConfigBytes, 0666)
	if err != nil {
		return err
	}

	return nil
}

func MachineExists(context, machineID string) bool {
	machineDir, err := GetMachineDir(context, machineID)
	if err != nil {
		return false
	}

	_, err = os.Stat(machineDir)

	return err == nil
}

func LoadProviderConfig(context, provider string) (*ProviderConfig, error) {
	providerDir, err := GetProviderDir(context, provider)
	if err != nil {
		return nil, err
	}

	providerFile := filepath.Join(providerDir, ProviderConfigFile)
	providerConfigBytes, err := os.ReadFile(providerFile)
	if err != nil {
		return nil, err
	}

	providerConfig, err := ParseProvider(bytes.NewReader(providerConfigBytes))
	if err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func LoadMachineConfig(context, machineID string) (*Machine, error) {
	machineDir, err := GetMachineDir(context, machineID)
	if err != nil {
		return nil, err
	}

	machineConfigFile := filepath.Join(machineDir, MachineConfigFile)
	machineConfigBytes, err := os.ReadFile(machineConfigFile)
	if err != nil {
		return nil, err
	}

	machineConfig := &Machine{}
	err = json.Unmarshal(machineConfigBytes, machineConfig)
	if err != nil {
		return nil, err
	}

	machineConfig.Context = context
	machineConfig.Origin = machineConfigFile
	return machineConfig, nil
}

func LoadEngineConfig(context, engineID string) (*Engine, error) {
	engineDir, err := GetEngineDir(context, engineID)
	if err != nil {
		return nil, err
	}

	engineConfigFile := filepath.Join(engineDir, EngineConfigFile)
	engineConfigBytes, err := os.ReadFile(engineConfigFile)
	if err != nil {
		return nil, err
	}

	engineConfig := &Engine{}
	err = json.Unmarshal(engineConfigBytes, engineConfig)
	if err != nil {
		return nil, err
	}

	return engineConfig, nil
}

func LoadWorkspaceConfig(context, workspaceID string) (*Workspace, error) {
	workspaceDir, err := GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return nil, err
	}

	workspaceConfigFile := filepath.Join(workspaceDir, WorkspaceConfigFile)
	workspaceConfigBytes, err := os.ReadFile(workspaceConfigFile)
	if err != nil {
		return nil, err
	}

	workspaceConfig := &Workspace{}
	err = json.Unmarshal(workspaceConfigBytes, workspaceConfig)
	if err != nil {
		return nil, err
	}

	workspaceConfig.Context = context
	workspaceConfig.Origin = workspaceConfigFile
	return workspaceConfig, nil
}
