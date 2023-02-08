package agent

import (
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/compress"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"time"
)

const DefaultInactivityTimeout = time.Hour

const RemoteDevPodHelperLocation = "/tmp/devpod"

const DefaultAgentDownloadURL = "https://github.com/FabianKramm/foundation/releases/download/test"

type AgentWorkspaceInfo struct {
	// Workspace holds the workspace info
	Workspace *provider2.Workspace `json:"workspace,omitempty"`

	// AgentConfig holds the agent configuration
	AgentConfig *provider2.ProviderAgentConfig `json:"agentConfig,omitempty"`

	// Folder holds the workspace folder on the remote server
	Folder string `json:"-"`
}

func NewAgentWorkspaceInfo(workspace *provider2.Workspace, provider provider2.Provider) (string, error) {
	// trim options that don't exist
	workspace = cloneWorkspace(workspace)
	if workspace.Provider.Options != nil {
		for name, option := range provider.Options() {
			_, ok := workspace.Provider.Options[name]
			if ok && option.Local {
				delete(workspace.Provider.Options, name)
			}
		}
	}

	// get agent config
	agentConfig, err := provider.AgentConfig()
	if err != nil {
		return "", errors.Wrap(err, "get agent config")
	}

	// marshal config
	out, err := json.Marshal(&AgentWorkspaceInfo{
		Workspace:   workspace,
		AgentConfig: agentConfig,
	})
	if err != nil {
		return "", err
	}

	return compress.Compress(string(out))
}

func cloneWorkspace(workspace *provider2.Workspace) *provider2.Workspace {
	out, _ := json.Marshal(workspace)
	ret := &provider2.Workspace{}
	_ = json.Unmarshal(out, ret)
	return ret
}
