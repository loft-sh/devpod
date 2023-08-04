package devcontainer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/driver/drivercreate"
	"github.com/loft-sh/devpod/pkg/encoding"
	"github.com/loft-sh/devpod/pkg/language"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func NewRunner(
	agentPath, agentDownloadURL string,
	workspaceConfig *provider2.AgentWorkspaceInfo,
	log log.Logger,
) (*Runner, error) {
	driver, err := drivercreate.NewDriver(workspaceConfig, log)
	if err != nil {
		return nil, err
	}

	// we use the workspace uid as id to avoid conflicts between container names

	return &Runner{
		Driver: driver,

		AgentPath:            agentPath,
		AgentDownloadURL:     agentDownloadURL,
		LocalWorkspaceFolder: workspaceConfig.ContentFolder,
		ID:                   GetRunnerIDFromWorkspace(workspaceConfig.Workspace),
		WorkspaceConfig:      workspaceConfig,
		Log:                  log,
	}, nil
}

type Runner struct {
	Driver driver.Driver

	WorkspaceConfig  *provider2.AgentWorkspaceInfo
	AgentPath        string
	AgentDownloadURL string

	LocalWorkspaceFolder string
	SubstitutionContext  *config.SubstitutionContext

	ID string

	Log log.Logger
}

type UpOptions struct {
	provider2.CLIOptions

	NoBuild    bool
	ForceBuild bool
}

func (r *Runner) prepare(
	options provider2.CLIOptions,
) (*config.SubstitutedConfig, *WorkspaceConfig, error) {
	rawParsedConfig, err := config.ParseDevContainerJSON(
		r.LocalWorkspaceFolder,
		r.WorkspaceConfig.Workspace.DevContainerPath,
	)

	// We want to fail only in case of real errors, non-existing devcontainer.jon
	// will be gracefully handled by the auto-detection mechanism
	if err != nil && !os.IsNotExist(err) {
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
	workspace := getWorkspace(
		r.LocalWorkspaceFolder,
		r.WorkspaceConfig.Workspace.ID,
		rawParsedConfig,
	)
	r.SubstitutionContext = &config.SubstitutionContext{
		DevContainerID:           config.GetDevContainerID(config.ListToObject(r.getLabels())),
		LocalWorkspaceFolder:     r.LocalWorkspaceFolder,
		ContainerWorkspaceFolder: workspace.RemoteWorkspaceFolder,
		Env:                      config.ListToObject(os.Environ()),
	}

	// substitute & load
	parsedConfig := &config.DevContainerConfig{}
	err = config.Substitute(r.SubstitutionContext, rawParsedConfig, parsedConfig)
	if err != nil {
		return nil, nil, err
	}
	if parsedConfig.WorkspaceFolder != "" {
		workspace.RemoteWorkspaceFolder = parsedConfig.WorkspaceFolder
	}
	if parsedConfig.WorkspaceMount != "" {
		workspace.WorkspaceMount = parsedConfig.WorkspaceMount
	}

	if options.DevContainerImage != "" {
		parsedConfig.Build = config.ConfigBuildOptions{}
		parsedConfig.Dockerfile = ""
		parsedConfig.DockerfileContainer = config.DockerfileContainer{}
		parsedConfig.ImageContainer = config.ImageContainer{Image: options.DevContainerImage}
	}

	parsedConfig.Origin = configFile
	return &config.SubstitutedConfig{
		Config: parsedConfig,
		Raw:    rawParsedConfig,
	}, &workspace, nil
}

func (r *Runner) Up(ctx context.Context, options UpOptions) (*config.Result, error) {
	substitutedConfig, workspace, err := r.prepare(options.CLIOptions)
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
		result, err = r.runSingleContainer(
			ctx,
			substitutedConfig,
			workspace.WorkspaceMount,
			options,
		)
		if err != nil {
			return nil, err
		}
	} else if len(substitutedConfig.Config.DockerComposeFile) > 0 {
		result, err = r.runDockerCompose(ctx, substitutedConfig, options)
		if err != nil {
			return nil, err
		}
	} else {
		r.Log.Warn("dev container config is missing one of \"image\", \"dockerFile\" or \"dockerComposeFile\" properties, defaulting to auto-detection")

		lang, err := language.DetectLanguage(r.LocalWorkspaceFolder)
		if err != nil {
			return nil, fmt.Errorf("could not detect project language and dev container config is missing one of \"image\", \"dockerFile\" or \"dockerComposeFile\" properties")
		}

		if language.MapConfig[lang] == nil {
			return nil, fmt.Errorf("could not detect project language and dev container config is missing one of \"image\", \"dockerFile\" or \"dockerComposeFile\" properties")
		}

		substitutedConfig.Config.ImageContainer = language.MapConfig[lang].ImageContainer

		result, err = r.runSingleContainer(ctx, substitutedConfig, workspace.WorkspaceMount, options)
		if err != nil {
			return nil, err
		}
	}

	// return result
	return result, nil
}

func (r *Runner) CommandDevContainer(
	ctx context.Context,
	containerId string,
	user string,
	command string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) error {
	return r.Driver.CommandDevContainer(ctx, containerId, user, command, stdin, stdout, stderr)
}

func (r *Runner) FindDevContainer(ctx context.Context) (*config.ContainerDetails, error) {
	labels := r.getLabels()
	containerDetails, err := r.Driver.FindDevContainer(ctx, labels)
	if err != nil {
		return nil, errors.Wrap(err, "find dev container")
	}

	return containerDetails, nil
}

func (r *Runner) getLabels() []string {
	return []string{config.DockerIDLabel + "=" + r.ID}
}

func isDockerFileConfig(config *config.DevContainerConfig) bool {
	return config.Dockerfile != "" || config.Build.Dockerfile != ""
}

func runInitializeCommand(
	workspaceFolder string,
	config *config.DevContainerConfig,
	log log.Logger,
) error {
	if len(config.InitializeCommand) == 0 {
		return nil
	}

	for _, cmd := range config.InitializeCommand {
		// should run in shell?
		var args []string
		if len(cmd) == 1 {
			args = []string{"sh", "-c", cmd[0]}
		} else {
			args = cmd
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
	}

	return nil
}

type WorkspaceConfig struct {
	WorkspaceMount        string
	RemoteWorkspaceFolder string
}

func getWorkspace(
	workspaceFolder, workspaceID string,
	conf *config.DevContainerConfig,
) WorkspaceConfig {
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
		WorkspaceMount: fmt.Sprintf(
			"type=bind,source=%s,target=%s%s",
			workspaceFolder,
			containerMountFolder,
			consistency,
		),
	}
}

func GetRunnerIDFromWorkspace(workspace *provider2.Workspace) string {
	ID := workspace.UID
	if encoding.IsLegacyUID(workspace.UID) {
		ID = workspace.ID
	}

	return ID
}
