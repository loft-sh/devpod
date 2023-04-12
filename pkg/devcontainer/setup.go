package devcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"runtime"
)

func (r *Runner) setupContainer(containerDetails *config.ContainerDetails, mergedConfig *config.MergedDevContainerConfig) error {
	// inject agent
	err := agent.InjectAgent(context.TODO(), func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		return r.Driver.CommandDevContainer(ctx, containerDetails.Id, "root", command, stdin, stdout, stderr)
	}, agent.ContainerDevPodHelperLocation, agent.DefaultAgentDownloadURL, false, r.Log)
	if err != nil {
		return errors.Wrap(err, "inject agent")
	}
	r.Log.Debugf("Injected into container")
	defer r.Log.Debugf("Done setting up container")

	// compress info
	marshalled, err := json.Marshal(&config.Result{
		ContainerDetails:    containerDetails,
		MergedConfig:        mergedConfig,
		SubstitutionContext: r.SubstitutionContext,
	})
	if err != nil {
		return err
	}
	compressed, err := compress.Compress(string(marshalled))
	if err != nil {
		return err
	}

	// compress workspace info
	workspaceConfigRaw, err := json.Marshal(r.WorkspaceConfig)
	if err != nil {
		return err
	}
	workspaceConfigCompressed, err := compress.Compress(string(workspaceConfigRaw))
	if err != nil {
		return err
	}

	writer := r.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// execute docker command
	r.Log.Infof("Setup container...")
	command := fmt.Sprintf("%s agent container setup --setup-info '%s' --workspace-info '%s'", agent.ContainerDevPodHelperLocation, compressed, workspaceConfigCompressed)
	if runtime.GOOS == "linux" || r.WorkspaceConfig.Agent.Driver != provider2.DockerDriver {
		command += " --chown-workspace"
	}
	if r.Log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}
	r.Log.Debugf("Run command: %s", command)
	err = r.Driver.CommandDevContainer(context.TODO(), containerDetails.Id, "root", command, nil, writer, writer)
	if err != nil {
		return err
	}

	return nil
}
