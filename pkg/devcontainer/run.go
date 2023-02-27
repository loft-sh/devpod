package devcontainer

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func NewRunner(agentPath, agentDownloadURL string, workspaceConfig *provider2.AgentWorkspaceInfo, log log.Logger) *Runner {
	return &Runner{
		Docker: &docker.DockerHelper{DockerCommand: "docker"},

		AgentPath:            agentPath,
		AgentDownloadURL:     agentDownloadURL,
		LocalWorkspaceFolder: workspaceConfig.Folder,
		ID:                   workspaceConfig.Workspace.ID,
		WorkspaceConfig:      workspaceConfig,
		Log:                  log,
	}
}

type Runner struct {
	Docker *docker.DockerHelper

	WorkspaceConfig  *provider2.AgentWorkspaceInfo
	AgentPath        string
	AgentDownloadURL string

	LocalWorkspaceFolder string
	SubstitutionContext  *config.SubstitutionContext

	ID  string
	Log log.Logger
}

type UpOptions struct{}

func (r *Runner) prepare() (*config.SubstitutedConfig, *WorkspaceConfig, error) {
	rawParsedConfig, err := config.ParseDevContainerJSON(r.LocalWorkspaceFolder)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing devcontainer.json")
	} else if rawParsedConfig == nil {
		// TODO: use a default config
		return nil, nil, fmt.Errorf("couldn't find a devcontainer.json")
	}
	configFile := rawParsedConfig.Origin

	// get workspace folder within container
	workspace := getWorkspace(r.LocalWorkspaceFolder, r.ID, rawParsedConfig)
	r.SubstitutionContext = &config.SubstitutionContext{
		DevContainerID:           config.GetDevContainerID(config.ListToObject(r.getLabels())),
		LocalWorkspaceFolder:     r.LocalWorkspaceFolder,
		ContainerWorkspaceFolder: workspace.RemoteWorkspaceFolder,
		Env:                      config.ListToObject(os.Environ()),
	}

	// substitute & load
	parsedConfig := &config.DevContainerConfig{}
	err = config.Substitute(r.SubstitutionContext, rawParsedConfig, parsedConfig)
	if parsedConfig.WorkspaceFolder != "" {
		workspace.RemoteWorkspaceFolder = parsedConfig.WorkspaceFolder
	}
	if parsedConfig.WorkspaceMount != "" {
		workspace.WorkspaceMount = parsedConfig.WorkspaceMount
	}
	parsedConfig.Origin = configFile
	return &config.SubstitutedConfig{
		Config: parsedConfig,
		Raw:    rawParsedConfig,
	}, &workspace, nil
}

func (r *Runner) Up(options UpOptions) (*config.Result, error) {
	substitutedConfig, workspace, err := r.prepare()
	if err != nil {
		return nil, err
	}

	// run initializeCommand
	err = runInitializeCommand(r.LocalWorkspaceFolder, substitutedConfig.Config, r.Log)
	if err != nil {
		return nil, err
	}

	// check if its a compose devcontainer.json
	if isDockerFileConfig(substitutedConfig.Config) || substitutedConfig.Config.Image != "" {
		return r.runSingleContainer(substitutedConfig, workspace.WorkspaceMount)
	} else if len(substitutedConfig.Config.DockerComposeFile) > 0 {
		// TODO: implement
		panic("unimplemented")
	}

	return nil, fmt.Errorf("dev container config is missing one of \"image\", \"dockerFile\" or \"dockerComposeFile\" properties")
}

func (r *Runner) FindDevContainer() (*config.ContainerDetails, error) {
	labels := r.getLabels()
	containerDetails, err := r.Docker.FindDevContainer(labels)
	if err != nil {
		return nil, errors.Wrap(err, "find dev container")
	}

	return containerDetails, nil
}

func (r *Runner) getLabels() []string {
	return []string{DockerIDLabel + "=" + r.ID}
}

func isDockerFileConfig(config *config.DevContainerConfig) bool {
	return config.Dockerfile != "" || config.Build.Dockerfile != ""
}

func runInitializeCommand(workspaceFolder string, config *config.DevContainerConfig, log log.Logger) error {
	if len(config.InitializeCommand) == 0 {
		return nil
	}

	// should run in shell?
	var args []string
	if len(config.InitializeCommand) == 1 {
		args = []string{"sh", "-c", config.InitializeCommand[0]}
	} else {
		args = config.InitializeCommand
	}

	// run the command
	log.Infof("Running initializeCommand from devcontainer.json: '%s'", strings.Join(args, " "))
	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	cmd.Dir = workspaceFolder
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

type WorkspaceConfig struct {
	WorkspaceMount        string
	RemoteWorkspaceFolder string
}

func getWorkspace(workspaceFolder, workspaceID string, conf *config.DevContainerConfig) WorkspaceConfig {
	if conf.WorkspaceMount != "" {
		mount := config.ParseMount(conf.WorkspaceMount)
		return WorkspaceConfig{
			WorkspaceMount:        conf.WorkspaceMount,
			RemoteWorkspaceFolder: mount.Target,
		}
	}

	containerMountFolder := "/workspaces/" + workspaceID
	consistency := ""
	if runtime.GOOS != "linux" {
		consistency = ",consistency='consistent'"
	}

	return WorkspaceConfig{
		RemoteWorkspaceFolder: containerMountFolder,
		WorkspaceMount:        fmt.Sprintf("type=bind,source=%s,target=%s%s", workspaceFolder, containerMountFolder, consistency),
	}
}
