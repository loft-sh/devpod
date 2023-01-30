package docker

import (
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/types"
)

func NewDockerProvider() types.WorkspaceProvider {
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

func (d *dockerProvider) Name() string {
	return "docker"
}

func (d *dockerProvider) newRunner(workspace *config.Workspace) *devcontainer.Runner {
	return devcontainer.NewRunner(agent.DefaultAgentDownloadURL, workspace.Source.LocalFolder, workspace.ID, d.log)
}
