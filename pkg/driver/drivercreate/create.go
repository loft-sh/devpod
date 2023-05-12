package drivercreate

import (
	"fmt"

	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/driver/docker"
	"github.com/loft-sh/devpod/pkg/driver/kubernetes"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
)

func NewDriver(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (driver.Driver, error) {
	driver := workspaceInfo.Agent.Driver
	if driver == "" || driver == provider2.DockerDriver {
		return docker.NewDockerDriver(workspaceInfo, log), nil
	} else if driver == provider2.KubernetesDriver {
		return kubernetes.NewKubernetesDriver(workspaceInfo, log), nil
	}

	return nil, fmt.Errorf("unrecognized driver '%s', possible values are %s or %s", driver, provider2.DockerDriver, provider2.KubernetesDriver)
}
