package machine

import (
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/types"
	"os"
)

func CreateMachine(context, machineID, providerName string) (*provider.Machine, error) {
	// get the machine dir
	machineDir, err := provider.GetMachineDir(context, machineID)
	if err != nil {
		return nil, err
	}

	// save machine config
	machine := &provider.Machine{
		ID:      machineID,
		Folder:  machineDir,
		Context: context,
		Provider: provider.MachineProviderConfig{
			Name: providerName,
		},
		CreationTimestamp: types.Now(),
	}

	// create machine folder
	err = provider.SaveMachineConfig(machine)
	if err != nil {
		_ = os.RemoveAll(machineDir)
		return nil, err
	}

	// create machine ssh keys
	_, err = ssh.GetPublicKeyBase(machine.Folder)
	if err != nil {
		_ = os.RemoveAll(machineDir)
		return nil, err
	}

	return machine, nil
}
