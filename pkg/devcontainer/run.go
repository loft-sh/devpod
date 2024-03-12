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

type Runner interface {
	Up(ctx context.Context, options UpOptions) (*config.Result, error)

	Build(ctx context.Context, options provider2.BuildOptions) (string, error)

	Find(ctx context.Context) (*config.ContainerDetails, error)

	Command(
		ctx context.Context,
		user string,
		command string,
		stdin io.Reader,
		stdout io.Writer,
		stderr io.Writer,
	) error

	Stop(ctx context.Context) error

	Delete(ctx context.Context) error

	Logs(ctx context.Context, writer io.Writer) error
}

func NewRunner(
	agentPath, agentDownloadURL string,
	workspaceConfig *provider2.AgentWorkspaceInfo,
	log log.Logger,
) (Runner, error) {
	driver, err := drivercreate.NewDriver(workspaceConfig, log)
	if err != nil {
		return nil, err
	}

	// we use the workspace uid as id to avoid conflicts between container names

	return &runner{
		Driver: driver,

		AgentPath:            agentPath,
		AgentDownloadURL:     agentDownloadURL,
		LocalWorkspaceFolder: workspaceConfig.ContentFolder,
		ID:                   GetRunnerIDFromWorkspace(workspaceConfig.Workspace),
		WorkspaceConfig:      workspaceConfig,
		Log:                  log,
	}, nil
}

type runner struct {
	Driver driver.Driver

	WorkspaceConfig  *provider2.AgentWorkspaceInfo
	AgentPath        string
	AgentDownloadURL string

	LocalWorkspaceFolder string

	ID string

	Log log.Logger
}

type UpOptions struct {
	provider2.CLIOptions

	NoBuild    bool
	ForceBuild bool
}

func (r *runner) Up(ctx context.Context, options UpOptions) (*config.Result, error) {
	// download workspace source before recreating container
	_, isDockerDriver := r.Driver.(driver.DockerDriver)
	if options.Recreate && !isDockerDriver {
		return r.recreateCustomDriver(ctx, options)
	}

	// prepare config
	substitutedConfig, substitutionContext, err := r.prepare(options.CLIOptions)
	if err != nil {
		return nil, err
	}

	// remove build information
	defer func() {
		contextPath := config.GetContextPath(substitutedConfig.Config)
		_ = os.RemoveAll(filepath.Join(contextPath, config.DevPodContextFeatureFolder))
	}()

	// run initializeCommand
	err = runInitializeCommand(r.LocalWorkspaceFolder, substitutedConfig.Config, r.Log)
	if err != nil {
		return nil, err
	}

	// check if its a compose devcontainer.json
	var result *config.Result
	if isDockerFileConfig(substitutedConfig.Config) || substitutedConfig.Config.Image != "" || substitutedConfig.Config.ContainerID != "" {
		result, err = r.runSingleContainer(
			ctx,
			substitutedConfig,
			substitutionContext,
			options,
		)
		if err != nil {
			return nil, err
		}
	} else if isDockerComposeConfig(substitutedConfig.Config) {
		result, err = r.runDockerCompose(ctx, substitutedConfig, substitutionContext, options)
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
		result, err = r.runSingleContainer(ctx, substitutedConfig, substitutionContext, options)
		if err != nil {
			return nil, err
		}
	}

	// return result
	return result, nil
}

func (r *runner) prepare(
	options provider2.CLIOptions,
) (*config.SubstitutedConfig, *config.SubstitutionContext, error) {
	var rawParsedConfig *config.DevContainerConfig
	if r.WorkspaceConfig.Workspace.DevContainerConfig != nil {
		rawParsedConfig = config.CloneDevContainerConfig(r.WorkspaceConfig.Workspace.DevContainerConfig)
		if r.WorkspaceConfig.Workspace.DevContainerPath != "" {
			rawParsedConfig.Origin = path.Join(filepath.ToSlash(r.LocalWorkspaceFolder), r.WorkspaceConfig.Workspace.DevContainerPath)
		} else {
			rawParsedConfig.Origin = path.Join(filepath.ToSlash(r.LocalWorkspaceFolder), ".devcontainer.devpod.json")
		}
	} else if r.WorkspaceConfig.Workspace.Source.Container != "" {
		rawParsedConfig = &config.DevContainerConfig{
			DevContainerConfigBase: config.DevContainerConfigBase{
				// Default workspace directory for containers
				// Upon inspecting the container, this would be updated to the correct folder, if found set
				WorkspaceFolder: "/",
			},
			RunningContainer: config.RunningContainer{
				ContainerID: r.WorkspaceConfig.Workspace.Source.Container,
			},
			Origin: "",
		}
	} else {
		var err error

		localWorkspaceFolder := r.LocalWorkspaceFolder
		// if a subpath is specified, let's move to it

		if r.WorkspaceConfig.Workspace.Source.GitSubPath != "" {
			localWorkspaceFolder = filepath.Join(r.LocalWorkspaceFolder, r.WorkspaceConfig.Workspace.Source.GitSubPath)
		}

		// parse the devcontainer json
		rawParsedConfig, err = config.ParseDevContainerJSON(
			localWorkspaceFolder,
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
	}
	configFile := rawParsedConfig.Origin

	// get workspace folder within container
	workspaceMount, containerWorkspaceFolder := getWorkspace(
		r.LocalWorkspaceFolder,
		r.WorkspaceConfig.Workspace.ID,
		rawParsedConfig,
	)
	substitutionContext := &config.SubstitutionContext{
		DevContainerID:           r.ID,
		LocalWorkspaceFolder:     r.LocalWorkspaceFolder,
		ContainerWorkspaceFolder: containerWorkspaceFolder,
		Env:                      config.ListToObject(os.Environ()),

		WorkspaceMount: workspaceMount,
	}

	// substitute & load
	parsedConfig := &config.DevContainerConfig{}
	err := config.Substitute(substitutionContext, rawParsedConfig, parsedConfig)
	if err != nil {
		return nil, nil, err
	}
	if parsedConfig.WorkspaceFolder != "" {
		substitutionContext.ContainerWorkspaceFolder = parsedConfig.WorkspaceFolder
	}
	if parsedConfig.WorkspaceMount != "" {
		substitutionContext.WorkspaceMount = parsedConfig.WorkspaceMount
	}

	if options.DevContainerImage != "" {
		parsedConfig.Build = nil
		parsedConfig.Dockerfile = ""
		parsedConfig.DockerfileContainer = config.DockerfileContainer{}
		parsedConfig.ImageContainer = config.ImageContainer{Image: options.DevContainerImage}
	}

	parsedConfig.Origin = configFile
	return &config.SubstitutedConfig{
		Config: parsedConfig,
		Raw:    rawParsedConfig,
	}, substitutionContext, nil
}

func (r *runner) Command(
	ctx context.Context,
	user string,
	command string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) error {
	return r.Driver.CommandDevContainer(ctx, r.ID, user, command, stdin, stdout, stderr)
}

func (r *runner) Find(ctx context.Context) (*config.ContainerDetails, error) {
	containerDetails, err := r.Driver.FindDevContainer(ctx, r.ID)
	if err != nil {
		return nil, errors.Wrap(err, "find dev container")
	}

	return containerDetails, nil
}

func (r *runner) Logs(ctx context.Context, writer io.Writer) error {
	return r.Driver.GetDevContainerLogs(ctx, r.ID, writer, writer)
}

func (r *runner) recreateCustomDriver(ctx context.Context, options UpOptions) (*config.Result, error) {
	err := r.Driver.StopDevContainer(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	// relaunch Up without recreate now
	options.Recreate = false
	return r.Up(ctx, options)
}

func isDockerFileConfig(config *config.DevContainerConfig) bool {
	return config.GetDockerfile() != ""
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
		errwriter := log.Writer(logrus.ErrorLevel, false)
		defer writer.Close()
		defer errwriter.Close()

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = writer
		cmd.Stderr = errwriter
		cmd.Dir = workspaceFolder
		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

func getWorkspace(
	workspaceFolder, workspaceID string,
	conf *config.DevContainerConfig,
) (string, string) {
	if conf.WorkspaceMount != "" {
		mount := config.ParseMount(conf.WorkspaceMount)
		return conf.WorkspaceMount, mount.Target
	}

	containerMountFolder := conf.WorkspaceFolder
	if containerMountFolder == "" {
		containerMountFolder = "/workspaces/" + workspaceID
	}

	consistency := ""
	if runtime.GOOS != "linux" {
		consistency = ",consistency='consistent'"
	}

	return fmt.Sprintf(
		"type=bind,source=%s,target=%s%s",
		workspaceFolder,
		containerMountFolder,
		consistency,
	), containerMountFolder
}

func GetRunnerIDFromWorkspace(workspace *provider2.Workspace) string {
	ID := workspace.UID
	if encoding.IsLegacyUID(workspace.UID) {
		ID = workspace.ID
	}

	return ID
}
