package workspace

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/machine"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
	"os"
)

func ResolveMachine(devPodConfig *config.Config, args []string, providerOverride string, log log.Logger) (client.Client, error) {
	// check if we have no args
	if len(args) == 0 {
		return nil, fmt.Errorf("please specify the machine name")
	}

	// convert to id
	machineID := ToID(args[0])

	// check if desired id already exists
	if provider2.MachineExists(devPodConfig.DefaultContext, machineID) {
		log.Infof("Machine %s already exists", machineID)
		return loadExistingMachine(machineID, devPodConfig, log)
	}

	// get default provider
	defaultProvider, allProviders, err := LoadProviders(devPodConfig, log)
	if err != nil {
		return nil, err
	}
	if providerOverride != "" {
		var ok bool
		defaultProvider, ok = allProviders[providerOverride]
		if !ok {
			return nil, fmt.Errorf("couldn't find provider %s", providerOverride)
		}
	}

	// resolve workspace
	machineObj, err := machine.CreateMachine(devPodConfig.DefaultContext, machineID, defaultProvider.Config.Name)
	if err != nil {
		return nil, err
	}

	// create a new client
	machineClient, err := clientimplementation.NewMachineClient(devPodConfig, defaultProvider.Config, machineObj, log)
	if err != nil {
		_ = os.RemoveAll(machineObj.Folder)
		return nil, err
	}

	return machineClient, nil
}

// MachineExists checks if the given workspace already exists
func MachineExists(devPodConfig *config.Config, args []string, log log.Logger) string {
	if len(args) == 0 {
		return ""
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0], log)

	// convert to id
	workspaceID := ToID(name)

	// already exists?
	if !provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		return ""
	}

	return workspaceID
}

// GetMachine creates a machine client
func GetMachine(devPodConfig *config.Config, args []string, log log.Logger) (client.Client, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectMachine(devPodConfig, log)
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0], log)

	// convert to id
	machineID := ToID(name)

	// already exists?
	if !provider2.MachineExists(devPodConfig.DefaultContext, machineID) {
		return nil, fmt.Errorf("machine %s doesn't exist", machineID)
	}

	// load workspace config
	return loadExistingMachine(machineID, devPodConfig, log)
}

func selectMachine(devPodConfig *config.Config, log log.Logger) (client.Client, error) {
	if !terminal.IsTerminalIn {
		return nil, provideWorkspaceArgErr
	}

	// ask which machine to use
	machinesDir, err := provider2.GetMachinesDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	machineIDs := []string{}
	seversDirs, err := os.ReadDir(machinesDir)
	for _, workspace := range seversDirs {
		machineIDs = append(machineIDs, workspace.Name())
	}
	if len(machineIDs) == 0 {
		return nil, provideWorkspaceArgErr
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please select a machine from the list below",
		DefaultValue: machineIDs[0],
		Options:      machineIDs,
		Sort:         true,
	})
	if err != nil {
		return nil, err
	}

	// load workspace
	return loadExistingMachine(answer, devPodConfig, log)
}

func loadExistingMachine(machineID string, devPodConfig *config.Config, log log.Logger) (client.Client, error) {
	machineConfig, err := provider2.LoadMachineConfig(devPodConfig.DefaultContext, machineID)
	if err != nil {
		return nil, err
	}

	providerWithOptions, err := FindProvider(devPodConfig, machineConfig.Provider.Name, log)
	if err != nil {
		return nil, err
	}

	return clientimplementation.NewMachineClient(devPodConfig, providerWithOptions.Config, machineConfig, log)
}
