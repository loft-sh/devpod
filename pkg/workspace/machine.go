package workspace

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
	"os"
)

// GetMachine creates a machine client
func GetMachine(ctx context.Context, devPodConfig *config.Config, args []string, log log.Logger) (client.Client, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectMachine(ctx, devPodConfig, log)
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0], log)

	// convert to id
	machineID := ToWorkspaceID(name)

	// already exists?
	if !provider2.MachineExists(devPodConfig.DefaultContext, machineID) {
		return nil, fmt.Errorf("machine %s doesn't exist", machineID)
	}

	// load workspace config
	return loadExistingMachine(ctx, machineID, devPodConfig, log)
}

func selectMachine(ctx context.Context, devPodConfig *config.Config, log log.Logger) (client.Client, error) {
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
	return loadExistingMachine(ctx, answer, devPodConfig, log)
}

func loadExistingMachine(ctx context.Context, machineID string, devPodConfig *config.Config, log log.Logger) (client.Client, error) {
	machineConfig, err := provider2.LoadMachineConfig(devPodConfig.DefaultContext, machineID)
	if err != nil {
		return nil, err
	}

	providerWithOptions, err := FindProvider(devPodConfig, machineConfig.Provider.Name, log)
	if err != nil {
		return nil, err
	}

	return clientimplementation.NewMachineClient(devPodConfig, providerWithOptions.Config, machineConfig, log), nil
}
