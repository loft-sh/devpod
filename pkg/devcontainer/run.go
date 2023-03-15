package devcontainer

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/compose"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/language"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func NewRunner(agentPath, agentDownloadURL string, workspaceConfig *provider2.AgentWorkspaceInfo, log log.Logger) *Runner {
	dockerHelper := &docker.DockerHelper{DockerCommand: "docker"}
	composeHelper, err := compose.NewComposeHelper("docker-compose", dockerHelper)
	if err != nil {
		log.Debugf("Could not create docker-compose helper: %+v", err)
	}

	return &Runner{
		Docker:  dockerHelper,
		Compose: composeHelper,

		AgentPath:            agentPath,
		AgentDownloadURL:     agentDownloadURL,
		LocalWorkspaceFolder: workspaceConfig.Folder,
		ID:                   workspaceConfig.Workspace.ID,
		WorkspaceConfig:      workspaceConfig,
		Log:                  log,
	}
}

type Runner struct {
	Docker  *docker.DockerHelper
	Compose *compose.ComposeHelper

	WorkspaceConfig  *provider2.AgentWorkspaceInfo
	AgentPath        string
	AgentDownloadURL string

	LocalWorkspaceFolder string
	SubstitutionContext  *config.SubstitutionContext

	ID  string
	Log log.Logger
}

type UpOptions struct {
	PrebuildRepositories []string

	NoBuild bool

	ForceBuild bool
	Recreate   bool
}

func (r *Runner) prepare() (*config.SubstitutedConfig, *WorkspaceConfig, error) {
	rawParsedConfig, err := config.ParseDevContainerJSON(r.LocalWorkspaceFolder)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing devcontainer.json")
	} else if rawParsedConfig == nil {
		r.Log.Infof("Couldn't find a devcontainer.json")
		r.Log.Infof("Try detecting project programming language...")
		defaultConfig := language.DefaultConfig(r.LocalWorkspaceFolder, r.Log)
		defaultConfig.Origin = path.Join(filepath.ToSlash(r.LocalWorkspaceFolder), ".devcontainer.json")
		err = config.SaveDevContainerJSON(defaultConfig)
		if err != nil {
			return nil, nil, errors.Wrap(err, "write default devcontainer.json")
		}

		rawParsedConfig = defaultConfig
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
	var result *config.Result
	if isDockerFileConfig(substitutedConfig.Config) || substitutedConfig.Config.Image != "" {
		result, err = r.runSingleContainer(substitutedConfig, workspace.WorkspaceMount, options)
		if err != nil {
			return nil, err
		}
	} else if len(substitutedConfig.Config.DockerComposeFile) > 0 {
		result, err = r.runDockerCompose(substitutedConfig, options)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("dev container config is missing one of \"image\", \"dockerFile\" or \"dockerComposeFile\" properties")
	}

	// write result
	err = agent.WriteAgentWorkspaceDevContainerResult(r.WorkspaceConfig.Agent.DataPath, r.WorkspaceConfig.Workspace.Context, r.WorkspaceConfig.Workspace.ID, result)
	if err != nil {
		r.Log.Errorf("Error writing dev container result: %v", err)
	}

	// return result
	return result, nil
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

	containerMountFolder := conf.WorkspaceFolder
	if containerMountFolder == "" {
		containerMountFolder = "/workspaces/" + workspaceID
	}

	consistency := ""
	if runtime.GOOS != "linux" {
		consistency = ",consistency='consistent'"
	}

	return WorkspaceConfig{
		RemoteWorkspaceFolder: containerMountFolder,
		WorkspaceMount:        fmt.Sprintf("type=bind,source=%s,target=%s%s", workspaceFolder, containerMountFolder, consistency),
	}
}
