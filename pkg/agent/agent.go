package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const DefaultInactivityTimeout = time.Hour

const RemoteDevPodHelperLocation = "/tmp/devpod"

const ContainerActivityFile = "/tmp/devpod.activity"

const DefaultAgentDownloadURL = "https://github.com/FabianKramm/foundation/releases/download/test"

func DecodeWorkspaceInfo(workspaceInfoRaw string) (*provider2.AgentWorkspaceInfo, string, error) {
	decoded, err := compress.Decompress(workspaceInfoRaw)
	if err != nil {
		return nil, "", errors.Wrap(err, "decode workspace info")
	}

	workspaceInfo := &provider2.AgentWorkspaceInfo{}
	err = json.Unmarshal([]byte(decoded), workspaceInfo)
	if err != nil {
		return nil, "", errors.Wrap(err, "parse workspace info")
	}

	return workspaceInfo, decoded, nil
}

func ReadAgentWorkspaceInfo(context, id string) (*provider2.AgentWorkspaceInfo, error) {
	// get workspace folder
	workspaceDir, err := GetAgentWorkspaceDir(context, id)
	if err != nil {
		return nil, err
	}

	// read workspace config
	out, err := os.ReadFile(filepath.Join(workspaceDir, provider2.WorkspaceConfigFile))
	if err != nil {
		return nil, err
	}

	// json unmarshal
	workspaceInfo := &provider2.AgentWorkspaceInfo{}
	err = json.Unmarshal(out, workspaceInfo)
	if err != nil {
		return nil, errors.Wrap(err, "parse workspace info")
	}

	workspaceInfo.Folder = GetAgentWorkspaceContentDir(workspaceDir)
	return workspaceInfo, nil
}

func WriteWorkspaceInfo(workspaceInfoRaw string) (*provider2.AgentWorkspaceInfo, error) {
	workspaceInfo, decoded, err := DecodeWorkspaceInfo(workspaceInfoRaw)
	if err != nil {
		return nil, err
	}

	// write to workspace folder
	workspaceDir, err := CreateAgentWorkspaceDir(workspaceInfo.Workspace.Context, workspaceInfo.Workspace.ID)
	if err != nil {
		return nil, err
	}

	// write workspace config
	err = os.WriteFile(filepath.Join(workspaceDir, provider2.WorkspaceConfigFile), []byte(decoded), 0666)
	if err != nil {
		return nil, fmt.Errorf("write workspace config file")
	}

	workspaceInfo.Folder = GetAgentWorkspaceContentDir(workspaceDir)
	return workspaceInfo, nil
}

func RerunAsRoot(workspaceInfo *provider2.AgentWorkspaceInfo) (bool, error) {
	// check if root is required
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		return false, nil
	}

	// check if we can reach docker with no problems
	dockerRootRequired, err := dockerReachable()
	if err != nil {
		return false, nil
	}

	// check if daemon needs to be installed
	agentRootRequired := false
	if runtime.GOOS == "linux" && len(workspaceInfo.Workspace.Provider.Agent.Exec.Shutdown) > 0 {
		agentRootRequired = true
	}

	// check if root required
	if !dockerRootRequired && !agentRootRequired {
		return false, nil
	}

	// execute ourself as root
	binary, err := os.Executable()
	if err != nil {
		return false, err
	}

	// call ourself
	args := []string{binary}
	args = append(args, os.Args[1:]...)
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return false, err
	}

	return true, nil
}

func Tunnel(ctx context.Context, dockerHelper *docker.DockerHelper, agentPath, agentDownloadURL string, containerID string, token string, stdin io.Reader, stdout io.Writer, stderr io.Writer, trackActivity bool, log log.Logger) error {
	// inject agent
	err := InjectAgent(ctx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		args := []string{"exec", "-i", "-u", "root", containerID, "sh", "-c", command}
		return dockerHelper.Run(ctx, args, stdin, stdout, stderr)
	}, agentPath, agentDownloadURL, false, log)
	if err != nil {
		return err
	}

	// build command
	command := fmt.Sprintf("%s helper ssh-server --token %s --stdio", RemoteDevPodHelperLocation, token)
	if trackActivity {
		command += " --track-activity"
	}

	// create tunnel
	args := []string{
		"exec",
		"-i",
		"-u", "root",
		containerID,
		"sh", "-c", command,
	}
	err = dockerHelper.Run(ctx, args, stdin, stdout, stderr)
	if err != nil {
		return err
	}

	return nil
}

func dockerReachable() (bool, error) {
	if !command.Exists("docker") {
		// we need root to install docker
		return true, nil
	}

	_, err := exec.Command("docker", "ps").CombinedOutput()
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return true, nil
		}

		return false, errors.Wrap(err, "docker ps")
	}

	return false, nil
}
