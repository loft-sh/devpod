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
	"time"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/crane"
	"github.com/loft-sh/devpod/pkg/devcontainer/sshtunnel"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/ide"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (r *runner) setupContainer(
	ctx context.Context,
	rawConfig *config.DevContainerConfig,
	containerDetails *config.ContainerDetails,
	mergedConfig *config.MergedDevContainerConfig,
	substitutionContext *config.SubstitutionContext,
	timeout time.Duration,
) (*config.Result, error) {
	// inject agent
	err := agent.InjectAgent(ctx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		return r.Driver.CommandDevContainer(ctx, r.ID, "root", command, stdin, stdout, stderr)
	}, false, agent.ContainerDevPodHelperLocation, agent.DefaultAgentDownloadURL(), false, r.Log, timeout)
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

	// Ensure workspace mounts cannot escape their content folder for local agents in proxy mode.
	// There _might_ be a use-case that requires an allowlist for certain directories
	// when running as a standalone runner with docker-in-docker set up. Let's add it when/if the time comes.
	if r.WorkspaceConfig.Agent.Local == "true" && r.WorkspaceConfig.CLIOptions.Platform.Enabled {
		result.MergedConfig.Mounts = filterWorkspaceMounts(result.MergedConfig.Mounts, r.WorkspaceConfig.ContentFolder, r.Log)
	}

	marshalled, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	compressed, err := compress.Compress(string(marshalled))
	if err != nil {
		return nil, err
	}

	workspaceConfig := &provider2.ContainerWorkspaceInfo{
		IDE:              r.WorkspaceConfig.Workspace.IDE,
		CLIOptions:       r.WorkspaceConfig.CLIOptions,
		Dockerless:       r.WorkspaceConfig.Agent.Dockerless,
		ContainerTimeout: r.WorkspaceConfig.Agent.ContainerTimeout,
		Source:           r.WorkspaceConfig.Workspace.Source,
		Agent:            r.WorkspaceConfig.Agent,
		ContentFolder:    r.WorkspaceConfig.ContentFolder,
	}
	if crane.ShouldUse(&r.WorkspaceConfig.CLIOptions) {
		workspaceConfig.PullFromInsideContainer = "true"
	}
	// compress container workspace info
	workspaceConfigRaw, err := json.Marshal(workspaceConfig)
	if err != nil {
		return nil, err
	}
	workspaceConfigCompressed, err := compress.Compress(string(workspaceConfigRaw))
	if err != nil {
		return nil, err
	}

	// check if docker driver
	_, isDockerDriver := r.Driver.(driver.DockerDriver)

	// setup container
	r.Log.Infof("Setup container...")

	setupCommand := fmt.Sprintf(
		"'%s' agent container setup --setup-info '%s' --container-workspace-info '%s'",
		agent.ContainerDevPodHelperLocation,
		compressed,
		workspaceConfigCompressed,
	)
	if runtime.GOOS == "linux" || !isDockerDriver {
		setupCommand += " --chown-workspace"
	}
	if !isDockerDriver {
		setupCommand += " --stream-mounts"
	}
	if r.WorkspaceConfig.Agent.InjectGitCredentials != "false" {
		setupCommand += " --inject-git-credentials"
	}
	if r.WorkspaceConfig.CLIOptions.Platform.AccessKey != "" &&
		r.WorkspaceConfig.CLIOptions.Platform.WorkspaceHost != "" &&
		r.WorkspaceConfig.CLIOptions.Platform.PlatformHost != "" {
		setupCommand += fmt.Sprintf(" --access-key '%s' --workspace-host '%s' --platform-host '%s'", r.WorkspaceConfig.CLIOptions.Platform.AccessKey, r.WorkspaceConfig.CLIOptions.Platform.WorkspaceHost, r.WorkspaceConfig.CLIOptions.Platform.PlatformHost)
	}
	if r.Log.GetLevel() == logrus.DebugLevel {
		setupCommand += " --debug"
	}

	// run setup server
	runSetupServer := func(ctx context.Context, stdin io.WriteCloser, stdout io.Reader) (*config.Result, error) {
		return tunnelserver.RunSetupServer(
			ctx,
			stdout,
			stdin,
			r.WorkspaceConfig.Agent.InjectGitCredentials != "false",
			r.WorkspaceConfig.Agent.InjectDockerCredentials != "false",
			config.GetMounts(result),
			r.Log,
			tunnelserver.WithPlatformOptions(&r.WorkspaceConfig.CLIOptions.Platform),
		)
	}

	// ssh tunnel
	sshTunnelCmd := fmt.Sprintf("'%s' helper ssh-server --stdio", agent.ContainerDevPodHelperLocation)
	if ide.ReusesAuthSock(r.WorkspaceConfig.Workspace.IDE.Name) {
		sshTunnelCmd += fmt.Sprintf(" --reuse-ssh-auth-sock=%s", r.WorkspaceConfig.CLIOptions.SSHAuthSockID)
	}
	if r.Log.GetLevel() == logrus.DebugLevel {
		sshTunnelCmd += " --debug"
	}

	agentInjectFunc := func(cancelCtx context.Context, sshCmd string, sshTunnelStdinReader, sshTunnelStdoutWriter *os.File, writer io.WriteCloser) error {
		return r.Driver.CommandDevContainer(cancelCtx, r.ID, "root", sshCmd, sshTunnelStdinReader, sshTunnelStdoutWriter, writer)
	}
	return sshtunnel.ExecuteCommand(
		ctx,
		nil,
		false,
		agentInjectFunc,
		sshTunnelCmd,
		setupCommand,
		r.Log,
		runSetupServer,
	)
}

func getRelativeDevContainerJson(origin, localWorkspaceFolder string) string {
	relativePath := strings.TrimPrefix(filepath.ToSlash(origin), filepath.ToSlash(localWorkspaceFolder))
	return strings.TrimPrefix(relativePath, "/")
}

func filterWorkspaceMounts(mounts []*config.Mount, baseFolder string, log log.Logger) []*config.Mount {
	retMounts := []*config.Mount{}
	for _, mount := range mounts {
		rel, err := filepath.Rel(baseFolder, mount.Source)
		if err != nil || strings.Contains(rel, "..") {
			log.Infof("Dropping workspace mount %s because it possibly accesses data outside of it's content directory", mount.Source)
			continue
		}

		retMounts = append(retMounts, mount)
	}

	return retMounts
}
