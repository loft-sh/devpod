package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/version"
	perrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const DefaultInactivityTimeout = time.Minute * 20

const ContainerDevPodHelperLocation = "/usr/local/bin/devpod"

const RemoteDevPodHelperLocation = "/tmp/devpod"

const ContainerActivityFile = "/tmp/devpod.activity"

const WorkspaceDevContainerResult = "result.json"

const defaultAgentDownloadURL = "https://github.com/loft-sh/devpod/releases/download/"

func DefaultAgentDownloadURL() string {
	devPodAgentURL := os.Getenv("DEVPOD_AGENT_URL")
	if devPodAgentURL != "" {
		return strings.TrimSuffix(devPodAgentURL, "/") + "/"
	}

	if version.GetVersion() == version.DevVersion {
		return "https://github.com/loft-sh/devpod/releases/latest/download/"
	}

	return defaultAgentDownloadURL + version.GetVersion()
}

func DecodeWorkspaceInfo(workspaceInfoRaw string) (*provider2.AgentWorkspaceInfo, string, error) {
	decoded, err := compress.Decompress(workspaceInfoRaw)
	if err != nil {
		return nil, "", perrors.Wrap(err, "decode workspace info")
	}

	workspaceInfo := &provider2.AgentWorkspaceInfo{}
	err = json.Unmarshal([]byte(decoded), workspaceInfo)
	if err != nil {
		return nil, "", perrors.Wrap(err, "parse workspace info")
	}

	return workspaceInfo, decoded, nil
}

func readAgentWorkspaceInfo(agentFolder, context, id string) (*provider2.AgentWorkspaceInfo, error) {
	// get workspace folder
	workspaceDir, err := GetAgentWorkspaceDir(agentFolder, context, id)
	if err != nil {
		return nil, err
	}

	// parse agent workspace info
	return parseAgentWorkspaceInfo(filepath.Join(workspaceDir, provider2.WorkspaceConfigFile))
}

func parseAgentWorkspaceInfo(workspaceConfigFile string) (*provider2.AgentWorkspaceInfo, error) {
	// read workspace config
	out, err := os.ReadFile(workspaceConfigFile)
	if err != nil {
		return nil, err
	}

	// json unmarshal
	workspaceInfo := &provider2.AgentWorkspaceInfo{}
	err = json.Unmarshal(out, workspaceInfo)
	if err != nil {
		return nil, perrors.Wrap(err, "parse workspace info")
	}

	workspaceInfo.Origin = filepath.Dir(workspaceConfigFile)
	return workspaceInfo, nil
}

func ReadAgentWorkspaceDevContainerResult(agentFolder, context, id string) (*config.Result, error) {
	// get workspace folder
	workspaceDir, err := GetAgentWorkspaceDir(agentFolder, context, id)
	if err != nil {
		return nil, err
	}

	// read workspace config
	out, err := os.ReadFile(filepath.Join(workspaceDir, WorkspaceDevContainerResult))
	if err != nil {
		return nil, err
	}

	// json unmarshal
	workspaceResult := &config.Result{}
	err = json.Unmarshal(out, workspaceResult)
	if err != nil {
		return nil, perrors.Wrap(err, "parse workspace result")
	}

	return workspaceResult, nil
}

func WriteAgentWorkspaceDevContainerResult(agentFolder, context, id string, result *config.Result) error {
	// get workspace folder
	workspaceDir, err := GetAgentWorkspaceDir(agentFolder, context, id)
	if err != nil {
		return err
	}

	// marshal result
	out, err := json.Marshal(result)
	if err != nil {
		return err
	}

	// read workspace config
	err = os.WriteFile(filepath.Join(workspaceDir, WorkspaceDevContainerResult), out, 0666)
	if err != nil {
		return err
	}

	return nil
}

func ReadAgentWorkspaceInfo(agentFolder, context, id string, log log.Logger) (bool, *provider2.AgentWorkspaceInfo, error) {
	workspaceInfo, err := readAgentWorkspaceInfo(agentFolder, context, id)
	if err != nil && !errors.Is(err, ErrFindAgentHomeFolder) {
		return false, nil, err
	}

	// check if we need to become root
	shouldExit, err := rerunAsRoot(workspaceInfo, log)
	if err != nil {
		return false, nil, perrors.Wrap(err, "rerun as root")
	} else if shouldExit {
		return true, nil, nil
	} else if workspaceInfo == nil {
		return false, nil, ErrFindAgentHomeFolder
	}

	return false, workspaceInfo, nil
}

func WriteWorkspaceInfo(workspaceInfoEncoded string, log log.Logger) (bool, *provider2.AgentWorkspaceInfo, error) {
	return WriteWorkspaceInfoAndDeleteOld(workspaceInfoEncoded, nil, log)
}

func WriteWorkspaceInfoAndDeleteOld(workspaceInfoEncoded string, deleteWorkspace func(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error, log log.Logger) (bool, *provider2.AgentWorkspaceInfo, error) {
	workspaceInfo, _, err := DecodeWorkspaceInfo(workspaceInfoEncoded)
	if err != nil {
		return false, nil, err
	}

	// check if we need to become root
	shouldExit, err := rerunAsRoot(workspaceInfo, log)
	if err != nil {
		return false, nil, fmt.Errorf("rerun as root: %w", err)
	} else if shouldExit {
		return true, nil, nil
	}

	// write to workspace folder
	workspaceDir, err := CreateAgentWorkspaceDir(workspaceInfo.Agent.DataPath, workspaceInfo.Workspace.Context, workspaceInfo.Workspace.ID)
	if err != nil {
		return false, nil, err
	}
	log.Debugf("Use %s as workspace dir", workspaceDir)

	// check if workspace config already exists
	workspaceConfig := filepath.Join(workspaceDir, provider2.WorkspaceConfigFile)
	oldWorkspaceInfo, _ := parseAgentWorkspaceInfo(workspaceConfig)
	if oldWorkspaceInfo != nil && oldWorkspaceInfo.Workspace.UID != workspaceInfo.Workspace.UID {
		// delete the old workspace
		log.Infof("Delete old workspace '%s'", oldWorkspaceInfo.Workspace.ID)
		err = deleteWorkspace(oldWorkspaceInfo, log)
		if err != nil {
			return false, nil, perrors.Wrap(err, "delete old workspace")
		}

		// recreate workspace folder again
		workspaceDir, err = CreateAgentWorkspaceDir(workspaceInfo.Agent.DataPath, workspaceInfo.Workspace.Context, workspaceInfo.Workspace.ID)
		if err != nil {
			return false, nil, err
		}
	}

	// check content folder
	if workspaceInfo.Workspace.Source.LocalFolder != "" {
		_, err = os.Stat(workspaceInfo.Workspace.Source.LocalFolder)
		if err == nil {
			workspaceInfo.ContentFolder = workspaceInfo.Workspace.Source.LocalFolder
		}
	}

	// set content folder
	if workspaceInfo.ContentFolder == "" {
		workspaceInfo.ContentFolder = GetAgentWorkspaceContentDir(workspaceDir)
	}

	// encode workspace info
	encoded, err := json.Marshal(workspaceInfo)
	if err != nil {
		return false, nil, err
	}

	// write workspace config
	err = os.WriteFile(workspaceConfig, encoded, 0666)
	if err != nil {
		return false, nil, fmt.Errorf("write workspace config file: %w", err)
	}

	workspaceInfo.Origin = workspaceDir
	return false, workspaceInfo, nil
}

func rerunAsRoot(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (bool, error) {
	// check if root is required
	if runtime.GOOS != "linux" || os.Getuid() == 0 || (workspaceInfo != nil && workspaceInfo.Agent.Local == "true") {
		return false, nil
	}

	// check if we can reach docker with no problems
	dockerRootRequired := false
	if workspaceInfo != nil && (workspaceInfo.Agent.Driver == "" || workspaceInfo.Agent.Driver == provider2.DockerDriver) {
		var err error
		dockerRootRequired, err = dockerReachable(workspaceInfo.Agent.Docker.Path, workspaceInfo.Agent.Docker.Env)
		if err != nil {
			log.Debugf("Error trying to reach docker daemon: %v", err)
			dockerRootRequired = true
		}
	}

	// check if daemon needs to be installed
	agentRootRequired := false
	if workspaceInfo == nil || len(workspaceInfo.Agent.Exec.Shutdown) > 0 {
		agentRootRequired = true
	}

	// check if root required
	if !dockerRootRequired && !agentRootRequired {
		log.Debugf("No root required, because neither docker nor agent daemon needs to be installed")
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
	log.Debugf("Rerun as root: %s", strings.Join(args, " "))
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

func Tunnel(
	ctx context.Context,
	driver driver.Driver,
	containerID string,
	token string,
	user string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	trackActivity bool,
	log log.Logger,
) error {
	// inject agent
	err := InjectAgent(ctx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		return driver.CommandDevContainer(ctx, containerID, "root", command, stdin, stdout, stderr)
	}, false, ContainerDevPodHelperLocation, DefaultAgentDownloadURL(), false, log)
	if err != nil {
		return err
	}

	// build command
	command := fmt.Sprintf("'%s' helper ssh-server --token '%s' --stdio", ContainerDevPodHelperLocation, token)
	if trackActivity {
		command += " --track-activity"
	}
	if log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}
	if user == "" {
		user = "root"
	}

	// create tunnel
	err = driver.CommandDevContainer(ctx, containerID, user, command, stdin, stdout, stderr)
	if err != nil {
		return err
	}

	return nil
}

func dockerReachable(dockerOverride string, envs map[string]string) (bool, error) {
	docker := "docker"
	if dockerOverride != "" {
		docker = dockerOverride
	}

	if !command.Exists(docker) {
		// if docker is overridden, we assume that there is an error as we don't know how to install the command provided
		if dockerOverride != "" {
			return false, fmt.Errorf("docker command '%s' not found", dockerOverride)
		}
		// we need root to install docker
		return true, nil
	}

	cmd := exec.Command(docker, "ps")
	if len(envs) > 0 {
		newEnvs := os.Environ()
		for k, v := range envs {
			newEnvs = append(newEnvs, k+"="+v)
		}
		cmd.Env = newEnvs
	}

	_, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			if dockerOverride == "" {
				return true, nil
			}
		}

		return false, perrors.Wrapf(err, "%s ps", docker)
	}

	return false, nil
}
