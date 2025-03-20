package devcontainer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

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
	Up(ctx context.Context, options UpOptions, timeout time.Duration) (*config.Result, error)

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

	NoBuild       bool
	ForceBuild    bool
	RegistryCache string
}

func (r *runner) Up(ctx context.Context, options UpOptions, timeout time.Duration) (*config.Result, error) {
	r.Log.Debugf("Up devcontainer for workspace '%s' with timeout %s", r.WorkspaceConfig.Workspace.ID, timeout)

	substitutedConfig, substitutionContext, err := r.getSubstitutedConfig(options.CLIOptions)
	if err != nil {
		return nil, err
	}
	defer cleanupBuildInformation(substitutedConfig.Config)

	// do not run initialize command in platform mode
	if !options.CLIOptions.Platform.Enabled {
		if err := runInitializeCommand(r.LocalWorkspaceFolder, substitutedConfig.Config, options.InitEnv, r.Log); err != nil {
			return nil, err
		}
	} else if len(substitutedConfig.Config.InitializeCommand) > 0 {
		r.Log.Info("Skipping initializeCommand on platform")
	}

	switch {
	case isDockerFileConfig(substitutedConfig.Config),
		substitutedConfig.Config.Image != "",
		substitutedConfig.Config.ContainerID != "":
		return r.runSingleContainer(
			ctx,
			substitutedConfig,
			substitutionContext,
			options,
			timeout,
		)
	case isDockerComposeConfig(substitutedConfig.Config):
		return r.runDockerCompose(ctx, substitutedConfig, substitutionContext, options, timeout)
	default:
		return r.runDefaultContainer(ctx, options, substitutedConfig, substitutionContext, timeout)
	}
}

func (r *runner) runDefaultContainer(ctx context.Context, options UpOptions, substitutedConfig *config.SubstitutedConfig, substitutionContext *config.SubstitutionContext, timeout time.Duration) (*config.Result, error) {
	if options.FallbackImage != "" {
		r.Log.Warn("dev container config is missing one of \"image\", \"dockerFile\" or \"dockerComposeFile\" properties, using fallback image " + options.FallbackImage)

		substitutedConfig.Config.ImageContainer = config.ImageContainer{
			Image: options.FallbackImage,
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
	}

	return r.runSingleContainer(ctx, substitutedConfig, substitutionContext, options, timeout)
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

func isDockerFileConfig(config *config.DevContainerConfig) bool {
	return config.GetDockerfile() != ""
}

func runInitializeCommand(
	workspaceFolder string,
	config *config.DevContainerConfig,
	extraEnvVars []string,
	log log.Logger,
) error {
	if len(config.InitializeCommand) == 0 {
		return nil
	}

	shellArgs := []string{"sh", "-c"}
	// According to the devcontainer spec, `initializeCommand` needs to be run on the host.
	// On Windows we can't assume everyone has `sh` added to their PATH so we need to use Windows default shell (usually cmd.exe)
	if runtime.GOOS == "windows" {
		comSpec := os.Getenv("COMSPEC")
		if comSpec != "" {
			shellArgs = []string{comSpec, "/c"}
		} else {
			shellArgs = []string{"cmd.exe", "/c"}
		}
	}

	for _, cmd := range config.InitializeCommand {
		// should run in shell?
		var args []string
		if len(cmd) == 1 {
			args = []string{shellArgs[0], shellArgs[1], cmd[0]}
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
		env := cmd.Environ()
		env = append(env, extraEnvVars...)

		cmd.Stdout = writer
		cmd.Stderr = errwriter
		cmd.Dir = workspaceFolder
		cmd.Env = env
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
