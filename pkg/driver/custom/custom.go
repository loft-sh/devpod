package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
	"github.com/sirupsen/logrus"
)

func NewCustomDriver(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) driver.Driver {
	return &customDriver{
		log:           log,
		workspaceInfo: workspaceInfo,
	}
}

var _ driver.Driver = (*customDriver)(nil)

type customDriver struct {
	log log.Logger

	workspaceInfo *provider2.AgentWorkspaceInfo
}

// FindDevContainer returns a running devcontainer details
func (c *customDriver) FindDevContainer(ctx context.Context, workspaceId string) (*config.ContainerDetails, error) {
	writer := c.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// run command
	stdout := &bytes.Buffer{}
	err := c.runCommand(
		ctx,
		workspaceId,
		"findDevContainer",
		c.workspaceInfo.Agent.Custom.FindDevContainer,
		nil,
		stdout,
		writer,
		nil,
		c.log,
	)
	if err != nil {
		return nil, fmt.Errorf("error finding dev container: %s%w", stdout.String(), err)
	} else if len(stdout.Bytes()) == 0 {
		return nil, nil
	}

	// parse stdout
	containerDetails := &config.ContainerDetails{}
	err = json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), containerDetails)
	if err != nil {
		return nil, fmt.Errorf("error parsing container details %s: %w", stdout.String(), err)
	}

	return containerDetails, nil
}

// CommandDevContainer runs the given command inside the devcontainer
func (c *customDriver) CommandDevContainer(ctx context.Context, workspaceId, user, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	// run command
	err := c.runCommand(
		ctx,
		workspaceId,
		"commandDevContainer",
		c.workspaceInfo.Agent.Custom.CommandDevContainer,
		stdin,
		stdout,
		stderr,
		[]string{
			"DEVCONTAINER_USER=" + user,
			"DEVCONTAINER_COMMAND=" + command,
		},
		c.log,
	)
	if err != nil {
		return err
	}

	return nil
}

// TargetArchitecture returns the architecture of the container runtime. e.g. amd64 or arm64
func (c *customDriver) TargetArchitecture(ctx context.Context, workspaceId string) (string, error) {
	writer := c.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// run command
	stdout := &bytes.Buffer{}
	err := c.runCommand(
		ctx,
		workspaceId,
		"getTargetArchitecture",
		c.workspaceInfo.Agent.Custom.TargetArchitecture,
		nil,
		stdout,
		writer,
		nil,
		c.log,
	)
	if err != nil {
		return "", fmt.Errorf("error getting target architecture: %s%w", stdout.String(), err)
	}

	// parse stdout
	targetArchitecture := strings.ToLower(strings.TrimSpace(stdout.String()))
	if targetArchitecture != "amd64" && targetArchitecture != "arm64" {
		return "", fmt.Errorf("invalid target architecture %s, expected either arm64 or amd64", targetArchitecture)
	}

	return targetArchitecture, nil
}

// DeleteDevContainer deletes the devcontainer
func (c *customDriver) DeleteDevContainer(ctx context.Context, workspaceId string) error {
	writer := c.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// run command
	err := c.runCommand(
		ctx,
		workspaceId,
		"deleteDevContainer",
		c.workspaceInfo.Agent.Custom.DeleteDevContainer,
		nil,
		writer,
		writer,
		nil,
		c.log,
	)
	if err != nil {
		return fmt.Errorf("error deleting devcontainer: %w", err)
	}

	return nil
}

// StartDevContainer starts the devcontainer
func (c *customDriver) StartDevContainer(ctx context.Context, workspaceId string) error {
	writer := c.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// run command
	err := c.runCommand(
		ctx,
		workspaceId,
		"startDevContainer",
		c.workspaceInfo.Agent.Custom.StartDevContainer,
		nil,
		writer,
		writer,
		nil,
		c.log,
	)
	if err != nil {
		return fmt.Errorf("error starting devcontainer: %w", err)
	}

	return nil
}

// StopDevContainer stops the devcontainer
func (c *customDriver) StopDevContainer(ctx context.Context, workspaceId string) error {
	writer := c.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// run command
	err := c.runCommand(
		ctx,
		workspaceId,
		"stopDevContainer",
		c.workspaceInfo.Agent.Custom.StopDevContainer,
		nil,
		writer,
		writer,
		nil,
		c.log,
	)
	if err != nil {
		return fmt.Errorf("error stopping devcontainer: %w", err)
	}

	return nil
}

// RunDevContainer runs a devcontainer
func (c *customDriver) RunDevContainer(ctx context.Context, workspaceId string, options *driver.RunOptions) error {
	out, err := json.Marshal(options)
	if err != nil {
		return fmt.Errorf("marshal run options: %w", err)
	}

	done := make(chan struct{})
	reader, writer := io.Pipe()
	defer writer.Close()
	go func() {
		scan := scanner.NewScanner(reader)
		for scan.Scan() {
			c.log.Info(scan.Text())
		}
		done <- struct{}{}
	}()

	// run command
	err = c.runCommand(
		ctx,
		workspaceId,
		"runDevContainer",
		c.workspaceInfo.Agent.Custom.RunDevContainer,
		nil,
		writer,
		writer,
		[]string{
			"DEVCONTAINER_RUN_OPTIONS=" + string(out),
		},
		c.log,
	)
	if err != nil {
		// close writer, wait for logging to flush and shut down
		writer.Close()
		select {
		case <-done:
		// forcibly shut down after 1 second
		case <-time.After(1 * time.Second):
		}
		return fmt.Errorf("error running devcontainer: %w", err)
	}

	return nil
}

func (c *customDriver) GetDevContainerLogs(ctx context.Context, workspaceID string, stdout io.Writer, stderr io.Writer) error {
	// run command
	err := c.runCommand(
		ctx,
		workspaceID,
		"getDevContainerLogs",
		c.workspaceInfo.Agent.Custom.GetDevContainerLogs,
		nil,
		stdout,
		stderr,
		nil,
		c.log,
	)
	if err != nil {
		return fmt.Errorf("error getting devcontainer logs: %w", err)
	}

	return nil
}

var _ driver.ReprovisioningDriver = (*customDriver)(nil)

func (c *customDriver) CanReprovision() bool {
	return c.workspaceInfo.Agent.Custom.CanReprovision == "true"
}

func (c *customDriver) runCommand(
	ctx context.Context,
	workspaceId string,
	name string,
	command types.StrArray,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	extraEnv []string,
	log log.Logger,
) error {
	if len(command) == 0 {
		return nil
	}

	// log
	log.Debugf("Run %s driver command: %s", name, strings.Join(command, " "))

	// get environ
	environ, err := ToEnvironWithBinaries(c.workspaceInfo, log)
	if err != nil {
		return err
	}
	environ = append(environ, provider2.DEVCONTAINER_ID+"="+workspaceId)
	environ = append(environ, extraEnv...)

	// set debug level
	if log.GetLevel() == logrus.DebugLevel {
		environ = append(environ, clientimplementation.DevPodDebug+"=true")
	}

	// run the command
	return clientimplementation.RunCommand(ctx, command, environ, stdin, stdout, stderr)
}

func ToEnvironWithBinaries(workspace *provider2.AgentWorkspaceInfo, log log.Logger) ([]string, error) {
	// get binaries dir
	binariesDir, err := agent.GetAgentBinariesDirFromWorkspaceDir(workspace.Origin)
	if err != nil {
		return nil, fmt.Errorf("error getting workspace %s binaries dir: %s %w", workspace.Workspace.ID, workspace.Origin, err)
	}

	// download binaries
	agentBinaries, err := binaries.DownloadBinaries(workspace.Agent.Binaries, binariesDir, log)
	if err != nil {
		return nil, fmt.Errorf("error downloading workspace %s binaries: %w", workspace.Workspace.ID, err)
	}

	// get environ
	environ := provider2.ToEnvironment(workspace.Workspace, workspace.Machine, workspace.Options, nil)
	for k, v := range agentBinaries {
		environ = append(environ, k+"="+v)
	}

	return environ, nil
}
