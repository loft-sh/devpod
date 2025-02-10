package provider

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/extract"
)

var excludedPaths = []string{
	".cache/",
	"cache/",
	"binaries/",
	"source/",
	".temp/",
	"temp/",
	".tmp/",
	"tmp/",
}

type ExportConfig struct {
	// Workspace is the workspace that was exported
	Workspace *ExportWorkspaceConfig `json:"workspace,omitempty"`

	// Machine is the machine that was exported
	Machine *ExportMachineConfig `json:"machine,omitempty"`

	// Provider is the provider that was exported
	Provider *ExportProviderConfig `json:"provider,omitempty"`
}

type ExportWorkspaceConfig struct {
	// ID is the workspace id
	ID string `json:"id,omitempty"`

	// Context is the workspace context
	Context string `json:"context,omitempty"`

	// UID is used to identify this specific workspace
	UID string `json:"uid,omitempty"`

	// Data is the workspace folder data
	Data string `json:"data,omitempty"`
}

type ExportMachineConfig struct {
	// ID is the machine id
	ID string `json:"id,omitempty"`

	// Context is the machine context
	Context string `json:"context,omitempty"`

	// Data is the machine folder data
	Data string `json:"data,omitempty"`
}

type ExportProviderConfig struct {
	// ID is the provider id
	ID string `json:"id,omitempty"`

	// Context is the provider context
	Context string `json:"context,omitempty"`

	// Data is the provider folder data
	Data string `json:"data,omitempty"`

	// Config is the provider config within the config.yaml
	Config *config.ProviderConfig `json:"config,omitempty"`
}

func ExportWorkspace(context, workspaceID string) (*ExportWorkspaceConfig, error) {
	workspaceDir, err := GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return nil, err
	}

	workspaceConfig, err := LoadWorkspaceConfig(context, workspaceID)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	err = extract.WriteTarExclude(buf, workspaceDir, true, excludedPaths)
	if err != nil {
		return nil, fmt.Errorf("compress workspace dir: %w", err)
	}

	return &ExportWorkspaceConfig{
		ID:      workspaceID,
		UID:     workspaceConfig.UID,
		Context: context,
		Data:    base64.RawStdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

func ExportMachine(context, machineID string) (*ExportMachineConfig, error) {
	machineDir, err := GetMachineDir(context, machineID)
	if err != nil {
		return nil, err
	}

	_, err = LoadMachineConfig(context, machineID)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	err = extract.WriteTarExclude(buf, machineDir, true, excludedPaths)
	if err != nil {
		return nil, fmt.Errorf("compress machine dir: %w", err)
	}

	return &ExportMachineConfig{
		ID:      machineID,
		Context: context,
		Data:    base64.RawStdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

func ExportProvider(devPodConfig *config.Config, context, providerID string) (*ExportProviderConfig, error) {
	providerDir, err := GetProviderDir(context, providerID)
	if err != nil {
		return nil, err
	}

	_, err = LoadProviderConfig(context, providerID)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	err = extract.WriteTarExclude(buf, providerDir, true, excludedPaths)
	if err != nil {
		return nil, fmt.Errorf("compress provider dir: %w", err)
	}

	var providerConfig *config.ProviderConfig
	if devPodConfig != nil && devPodConfig.Contexts[context] != nil && devPodConfig.Contexts[context].Providers != nil && devPodConfig.Contexts[context].Providers[providerID] != nil {
		providerConfig = devPodConfig.Contexts[context].Providers[providerID]
	}

	return &ExportProviderConfig{
		ID:      providerID,
		Context: context,
		Data:    base64.RawStdEncoding.EncodeToString(buf.Bytes()),
		Config:  providerConfig,
	}, nil
}
