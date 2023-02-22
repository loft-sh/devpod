package devcontainer

import (
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

func (r *Runner) setupContainer(containerDetails *config.ContainerDetails, mergedConfig *config.MergedDevContainerConfig) error {
	// inject agent
	r.Log.Infof("Setup container...")
	err := agent.InjectAgent(func(command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		args := []string{"exec", "-i", "-u", "root", containerDetails.Id, "sh", "-c", command}
		return r.Docker.Run(args, stdin, stdout, stderr)
	}, agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, false, time.Second*10)
	if err != nil {
		return err
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
	workspaceConfig, err := json.Marshal(r.WorkspaceConfig)
	if err != nil {
		return err
	}
	workspaceConfigCompressed, err := compress.Compress(string(workspaceConfig))
	if err != nil {
		return err
	}

	// execute docker command
	writer := r.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// TODO: run post commands
	// TODO: install dot files

	// TODO: install openvscode, extensions & settings
	// TODO: install vscode, extensions & settings
	args := []string{"exec", "-u", "root", containerDetails.Id, agent.RemoteDevPodHelperLocation, "agent", "container", "setup", "--setup-info", compressed, "--workspace-info", workspaceConfigCompressed}
	if r.Log.GetLevel() == logrus.DebugLevel {
		args = append(args, "--debug")
	}
	r.Log.Debugf("Run docker %s", strings.Join(args, " "))
	err = r.Docker.Run(args, nil, writer, writer)
	if err != nil {
		return err
	}

	return nil
}
