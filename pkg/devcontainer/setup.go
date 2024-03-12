package devcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/sshtunnel"
	"github.com/loft-sh/devpod/pkg/driver"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (r *runner) setupContainer(
	ctx context.Context,
	rawConfig *config.DevContainerConfig,
	containerDetails *config.ContainerDetails,
	mergedConfig *config.MergedDevContainerConfig,
	substitutionContext *config.SubstitutionContext,
) (*config.Result, error) {
	// inject agent
	err := agent.InjectAgent(ctx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		return r.Driver.CommandDevContainer(ctx, r.ID, "root", command, stdin, stdout, stderr)
	}, false, agent.ContainerDevPodHelperLocation, agent.DefaultAgentDownloadURL(), false, r.Log)
	if err != nil {
		return nil, errors.Wrap(err, "inject agent")
	}
	r.Log.Debugf("Injected into container")
	defer r.Log.Debugf("Done setting up container")

	// compress info
	result := &config.Result{
		DevContainerConfigWithPath: &config.DevContainerConfigWithPath{
			Config: rawConfig,
			Path:   getRelativeDevContainerJson(rawConfig.Origin, r.LocalWorkspaceFolder),
		},

		MergedConfig:        mergedConfig,
		SubstitutionContext: substitutionContext,
		ContainerDetails:    containerDetails,
	}
	marshalled, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	compressed, err := compress.Compress(string(marshalled))
	if err != nil {
		return nil, err
	}

	// compress container workspace info
	workspaceConfigRaw, err := json.Marshal(&provider2.ContainerWorkspaceInfo{
		IDE:              r.WorkspaceConfig.Workspace.IDE,
		CLIOptions:       r.WorkspaceConfig.CLIOptions,
		Dockerless:       r.WorkspaceConfig.Agent.Dockerless,
		ContainerTimeout: r.WorkspaceConfig.Agent.ContainerTimeout,
	})
	if err != nil {
		return nil, err
	}
	workspaceConfigCompressed, err := compress.Compress(string(workspaceConfigRaw))
	if err != nil {
		return nil, err
	}

	// check if docker driver
	_, isDockerDriver := r.Driver.(driver.DockerDriver)

	// ssh tunnel
	sshTunnelCmd := fmt.Sprintf("'%s' helper ssh-server --stdio", agent.ContainerDevPodHelperLocation)
	if r.Log.GetLevel() == logrus.DebugLevel {
		sshTunnelCmd += " --debug"
	}

	// setup container
	r.Log.Infof("Setup container...")
	setupCommand := fmt.Sprintf("'%s' agent container setup --setup-info '%s' --container-workspace-info '%s'", agent.ContainerDevPodHelperLocation, compressed, workspaceConfigCompressed)
	if runtime.GOOS == "linux" || !isDockerDriver {
		setupCommand += " --chown-workspace"
	}
	if !isDockerDriver {
		setupCommand += " --stream-mounts"
	}
	if r.WorkspaceConfig.Agent.InjectGitCredentials != "false" {
		setupCommand += " --inject-git-credentials"
	}
	if r.Log.GetLevel() == logrus.DebugLevel {
		setupCommand += " --debug"
	}

	agentInjectFunc := func(cancelCtx context.Context, sshCmd string, sshTunnelStdinReader, sshTunnelStdoutWriter *os.File, writer io.WriteCloser) error {
		return r.Driver.CommandDevContainer(cancelCtx, r.ID, "root", sshCmd, sshTunnelStdinReader, sshTunnelStdoutWriter, writer)
	}

	return sshtunnel.ExecuteCommand(
		ctx,
		nil,
		agentInjectFunc,
		sshTunnelCmd,
		setupCommand,
		false,
		true,
		r.WorkspaceConfig.Agent.InjectGitCredentials != "false",
		r.WorkspaceConfig.Agent.InjectDockerCredentials != "false",
		config.GetMounts(result),
		r.Log,
	)
}

func getRelativeDevContainerJson(origin, localWorkspaceFolder string) string {
	relativePath := strings.TrimPrefix(filepath.ToSlash(origin), filepath.ToSlash(localWorkspaceFolder))
	return strings.TrimPrefix(relativePath, "/")
}
