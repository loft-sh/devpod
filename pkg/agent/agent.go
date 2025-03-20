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
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/version"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const DefaultInactivityTimeout = time.Minute * 20

const ContainerDevPodHelperLocation = "/usr/local/bin/devpod"

const RemoteDevPodHelperLocation = "/tmp/devpod"

const ContainerActivityFile = "/tmp/devpod.activity"

const defaultAgentDownloadURL = "https://github.com/loft-sh/devpod/releases/download/"

const EnvDevPodAgentURL = "DEVPOD_AGENT_URL"

const WorkspaceBusyFile = "workspace.lock"

func DefaultAgentDownloadURL() string {
	devPodAgentURL := os.Getenv(EnvDevPodAgentURL)
	if devPodAgentURL != "" {
		return strings.TrimSuffix(devPodAgentURL, "/") + "/"
	}

	if version.GetVersion() == version.DevVersion {
		return "https://github.com/loft-sh/devpod/releases/latest/download/"
	}

	return defaultAgentDownloadURL + version.GetVersion()
}

func DecodeContainerWorkspaceInfo(workspaceInfoRaw string) (*provider2.ContainerWorkspaceInfo, string, error) {
	decoded, err := compress.Decompress(workspaceInfoRaw)
	if err != nil {
		return nil, "", perrors.Wrap(err, "decode workspace info")
	}

	workspaceInfo := &provider2.ContainerWorkspaceInfo{}
	err = json.Unmarshal([]byte(decoded), workspaceInfo)
	if err != nil {
		return nil, "", perrors.Wrap(err, "parse workspace info")
	}

	return workspaceInfo, decoded, nil
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
	return ParseAgentWorkspaceInfo(filepath.Join(workspaceDir, provider2.WorkspaceConfigFile))
}

func ParseAgentWorkspaceInfo(workspaceConfigFile string) (*provider2.AgentWorkspaceInfo, error) {
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

func ReadAgentWorkspaceInfo(agentFolder, context, id string, log log.Logger) (bool, *provider2.AgentWorkspaceInfo, error) {
	workspaceInfo, err := readAgentWorkspaceInfo(agentFolder, context, id)
	if err != nil && !(errors.Is(err, ErrFindAgentHomeFolder) || errors.Is(err, os.ErrPermission)) {
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

func WorkspaceInfo(workspaceInfoEncoded string, log log.Logger) (bool, *provider2.AgentWorkspaceInfo, error) {
	return decodeWorkspaceInfoAndWrite(workspaceInfoEncoded, false, nil, log)
}

func WriteWorkspaceInfo(workspaceInfoEncoded string, log log.Logger) (bool, *provider2.AgentWorkspaceInfo, error) {
	return WriteWorkspaceInfoAndDeleteOld(workspaceInfoEncoded, nil, log)
}

func WriteWorkspaceInfoAndDeleteOld(workspaceInfoEncoded string, deleteWorkspace func(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error, log log.Logger) (bool, *provider2.AgentWorkspaceInfo, error) {
	return decodeWorkspaceInfoAndWrite(workspaceInfoEncoded, true, deleteWorkspace, log)
}

func decodeWorkspaceInfoAndWrite(
	workspaceInfoEncoded string,
	writeInfo bool,
	deleteWorkspace func(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error,
	log log.Logger,
) (bool, *provider2.AgentWorkspaceInfo, error) {
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
	if deleteWorkspace != nil {
		oldWorkspaceInfo, _ := ParseAgentWorkspaceInfo(workspaceConfig)
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
	}

	// check content folder for local folder workspace source
	//
	// We don't want to initialize the content folder with the value of the local workspace folder
	// if we're running in proxy mode.
	// We only have write access to /var/lib/loft/* by default causing nearly all local folders to run into permissions issues
	if workspaceInfo.Workspace.Source.LocalFolder != "" && !workspaceInfo.CLIOptions.Platform.Enabled {
		_, err = os.Stat(workspaceInfo.WorkspaceOrigin)
		if err == nil {
			workspaceInfo.ContentFolder = workspaceInfo.Workspace.Source.LocalFolder
		}
	}

	// set content folder
	if workspaceInfo.ContentFolder == "" {
		workspaceInfo.ContentFolder = GetAgentWorkspaceContentDir(workspaceDir)
	}

	// write workspace info
	if writeInfo {
		err = writeWorkspaceInfo(workspaceConfig, workspaceInfo)
		if err != nil {
			return false, nil, err
		}
	}

	workspaceInfo.Origin = workspaceDir
	return false, workspaceInfo, nil
}

func CreateWorkspaceBusyFile(folder string) {
	filePath := filepath.Join(folder, WorkspaceBusyFile)
	_, err := os.Stat(filePath)
	if err == nil {
		return
	}

	_ = os.WriteFile(filePath, nil, 0600)
}

func HasWorkspaceBusyFile(folder string) bool {
	filePath := filepath.Join(folder, WorkspaceBusyFile)
	_, err := os.Stat(filePath)
	return err == nil
}

func DeleteWorkspaceBusyFile(folder string) {
	_ = os.Remove(filepath.Join(folder, WorkspaceBusyFile))
}

func writeWorkspaceInfo(file string, workspaceInfo *provider2.AgentWorkspaceInfo) error {
	// copy workspace info
	cloned := provider2.CloneAgentWorkspaceInfo(workspaceInfo)

	// never save cli options
	cloned.CLIOptions = provider2.CLIOptions{}

	// encode workspace info
	encoded, err := json.Marshal(workspaceInfo)
	if err != nil {
		return err
	}

	// write workspace config
	err = os.WriteFile(file, encoded, 0600)
	if err != nil {
		return fmt.Errorf("write workspace config file: %w", err)
	}

	return nil
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
	args := []string{"--preserve-env", binary}
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

type Exec func(ctx context.Context, user string, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

func Tunnel(
	ctx context.Context,
	exec Exec,
	user string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	log log.Logger,
	timeout time.Duration,
) error {
	// inject agent
	err := InjectAgent(ctx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		return exec(ctx, "root", command, stdin, stdout, stderr)
	}, false, ContainerDevPodHelperLocation, DefaultAgentDownloadURL(), false, log, timeout)
	if err != nil {
		return err
	}

	// build command
	command := fmt.Sprintf("'%s' helper ssh-server --stdio", ContainerDevPodHelperLocation)
	if log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}
	if user == "" {
		user = "root"
	}

	// create tunnel
	err = exec(ctx, user, command, stdin, stdout, stderr)
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
