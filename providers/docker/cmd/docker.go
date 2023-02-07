package cmd

import (
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
)

func NewDockerProvider() *dockerProvider {
	return &dockerProvider{
		docker: &docker.DockerHelper{
			DockerCommand: "docker",
		},
		log: log.Default,
	}
}

type dockerProvider struct {
	docker *docker.DockerHelper
	log    log.Logger
}

func (d *dockerProvider) newRunner(workspace *provider.Workspace, log log.Logger) *devcontainer.Runner {
	return devcontainer.NewRunner(agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, workspace.Source.LocalFolder, workspace.ID, log)
}
