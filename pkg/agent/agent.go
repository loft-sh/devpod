package agent

import (
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"time"
)

const DefaultInactivityTimeout = time.Hour

const RemoteDevPodHelperLocation = "/tmp/devpod"

const DefaultAgentDownloadURL = "https://github.com/FabianKramm/foundation/releases/download/test"

func GetAgentConfig(provider provider2.Provider) (*provider2.ProviderAgentConfig, error) {
	agentConfig, err := provider.AgentConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get agent config")
	}
	if agentConfig.Path == "" {
		agentConfig.Path = RemoteDevPodHelperLocation
	}
	if agentConfig.DownloadURL == "" {
		agentConfig.DownloadURL = DefaultAgentDownloadURL
	}

	return agentConfig, nil
}
