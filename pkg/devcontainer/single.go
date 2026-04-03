package devcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/skevetter/devpod/pkg/command"
	"github.com/skevetter/devpod/pkg/daemon/agent"
	"github.com/skevetter/devpod/pkg/devcontainer/config"
	"github.com/skevetter/devpod/pkg/devcontainer/metadata"
	"github.com/skevetter/devpod/pkg/driver"
	provider2 "github.com/skevetter/devpod/pkg/provider"
)

var dockerlessImage = "ghcr.io/loft-sh/dockerless:0.2.0"

const (
	DevPodExtraEnvVar           = "DEVPOD"
	RemoteContainersExtraEnvVar = "REMOTE_CONTAINERS"
	WorkspaceIDExtraEnvVar      = "DEVPOD_WORKSPACE_ID"
	WorkspaceUIDExtraEnvVar     = "DEVPOD_WORKSPACE_UID"

	DefaultEntrypoint = `
while ! command -v /usr/local/bin/devpod >/dev/null 2>&1; do
  echo "waiting for devpod agent to be available"
  sleep 1
done
exec /usr/local/bin/devpod agent container daemon
`
)

// resolvedContainer holds the outputs that every code path through
// runSingleContainer must produce before handing off to setupContainer.
// Using a struct makes it impossible to forget either field.
type resolvedContainer struct {
	details      *config.ContainerDetails
	mergedConfig *config.MergedDevContainerConfig
}

// resolveParams bundles the arguments shared by the container resolution methods.
type resolveParams struct {
	parsedConfig        *config.SubstitutedConfig
	substitutionContext *config.SubstitutionContext
	options             UpOptions
}

func (r *runner) runSingleContainer(
	ctx context.Context,
	parsedConfig *config.SubstitutedConfig,
	substitutionContext *config.SubstitutionContext,
	options UpOptions,
	timeout time.Duration,
) (*config.Result, error) {
	r.Log.Debugf("starting devcontainer for workspace %s", r.ID)

	substitutionContext.Userns = options.Userns
	substitutionContext.UidMap = options.UidMap
	substitutionContext.GidMap = options.GidMap

	// Check if Docker exists before trying to find containers
	var containerDetails *config.ContainerDetails
	var err error
	dockerCmd := "docker"
	if r.WorkspaceConfig.Agent.Docker.Path != "" {
		dockerCmd = r.WorkspaceConfig.Agent.Docker.Path
	}
	if command.Exists(dockerCmd) {
		containerDetails, err = r.Driver.FindDevContainer(ctx, r.ID)
		if err != nil {
			return nil, fmt.Errorf("find dev container: %w", err)
		}
	}

	// Resolve container: ensure we have a running container with merged config.
	var resolved *resolvedContainer

	params := &resolveParams{
		parsedConfig:        parsedConfig,
		substitutionContext: substitutionContext,
		options:             options,
	}

	if options.Recreate && parsedConfig.Config.ContainerID != "" {
		return nil, fmt.Errorf("cannot recreate container not created by DevPod")
	} else if !options.Recreate && containerDetails != nil {
		resolved, err = r.resolveExistingContainer(ctx, containerDetails, params)
	} else {
		resolved, err = r.resolveNewContainer(ctx, params)
	}
	if err != nil {
		return nil, err
	}

	return r.setupContainer(ctx, &setupContainerParams{
		rawConfig:           parsedConfig.Raw,
		containerDetails:    resolved.details,
		mergedConfig:        resolved.mergedConfig,
		substitutionContext: substitutionContext,
		timeout:             timeout,
	})
}

// resolveExistingContainer handles the case where a container already exists.
// It starts the container if stopped, merges configuration from container
// metadata, and optionally reprovisions. Returns fresh container details.
func (r *runner) resolveExistingContainer(
	ctx context.Context,
	containerDetails *config.ContainerDetails,
	p *resolveParams,
) (*resolvedContainer, error) {
	containerDetails, err := r.ensureRunning(ctx, containerDetails)
	if err != nil {
		return nil, err
	}

	// For non-managed containers with a workingDir, use it as the workspace folder.
	if p.parsedConfig.Config.ContainerID != "" && containerDetails.Config.WorkingDir != "" {
		p.substitutionContext.ContainerWorkspaceFolder = containerDetails.Config.WorkingDir
	}

	mergedConfig, err := r.mergeExistingContainerConfig(containerDetails, p)
	if err != nil {
		return nil, err
	}

	containerDetails, err = r.reprovisionIfNeeded(ctx, containerDetails)
	if err != nil {
		return nil, err
	}

	return &resolvedContainer{
		details:      containerDetails,
		mergedConfig: mergedConfig,
	}, nil
}

// ensureRunning starts the container if it is not running and returns
// fresh container details.
func (r *runner) ensureRunning(
	ctx context.Context,
	containerDetails *config.ContainerDetails,
) (*config.ContainerDetails, error) {
	if strings.ToLower(containerDetails.State.Status) == "running" {
		return containerDetails, nil
	}

	if err := r.Driver.StartDevContainer(ctx, r.ID); err != nil {
		return nil, err
	}
	return r.findRunningContainerOrFail(ctx, "start")
}

// mergeExistingContainerConfig extracts image metadata from the running
// container and merges it with the parsed devcontainer configuration.
func (r *runner) mergeExistingContainerConfig(
	containerDetails *config.ContainerDetails,
	p *resolveParams,
) (*config.MergedDevContainerConfig, error) {
	imageMetadataConfig, err := metadata.GetImageMetadataFromContainer(
		containerDetails,
		p.substitutionContext,
		r.Log,
	)
	if err != nil {
		return nil, err
	}

	if p.options.ExtraDevContainerPath != "" {
		if imageMetadataConfig == nil {
			imageMetadataConfig = &config.ImageMetadataConfig{}
		}
		extraConfig, err := config.ParseDevContainerJSONFile(p.options.ExtraDevContainerPath)
		if err != nil {
			return nil, err
		}
		config.AddConfigToImageMetadata(extraConfig, imageMetadataConfig)
	}

	mergedConfig, err := config.MergeConfiguration(
		p.parsedConfig.Config,
		imageMetadataConfig.Config,
	)
	if err != nil {
		return nil, fmt.Errorf("merge config: %w", err)
	}
	return mergedConfig, nil
}

// reprovisionIfNeeded re-runs the devcontainer via the driver if the driver
// supports reprovisioning, and returns fresh container details.
func (r *runner) reprovisionIfNeeded(
	ctx context.Context,
	containerDetails *config.ContainerDetails,
) (*config.ContainerDetails, error) {
	d, ok := r.Driver.(driver.ReprovisioningDriver)
	if !ok || !d.CanReprovision() {
		return containerDetails, nil
	}

	if err := r.Driver.RunDevContainer(ctx, r.ID, nil); err != nil {
		return nil, fmt.Errorf("runner driver run dev container: %w", err)
	}
	return r.findRunningContainerOrFail(ctx, "reprovision")
}

// resolveNewContainer handles the case where no container exists (or recreate
// is requested). It builds the image, runs a new container, and returns fresh
// container details.
func (r *runner) resolveNewContainer(
	ctx context.Context,
	p *resolveParams,
) (*resolvedContainer, error) {
	buildInfo, err := r.build(ctx, p.parsedConfig, p.substitutionContext, provider2.BuildOptions{
		CLIOptions: provider2.CLIOptions{
			PrebuildRepositories:  p.options.PrebuildRepositories,
			ForceDockerless:       p.options.ForceDockerless,
			Platform:              p.options.Platform,
			ExtraDevContainerPath: p.options.ExtraDevContainerPath,
		},
		NoBuild:       p.options.NoBuild,
		RegistryCache: p.options.RegistryCache,
		ExportCache:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("build image: %w", err)
	}

	if p.options.Recreate {
		if err := r.deleteForRecreate(ctx); err != nil {
			return nil, err
		}
	}

	mergedConfig, err := config.MergeConfiguration(
		p.parsedConfig.Config,
		buildInfo.ImageMetadata.Config,
	)
	if err != nil {
		return nil, fmt.Errorf("merge config: %w", err)
	}

	r.injectDaemonEntrypoint(p, mergedConfig)

	err = r.runContainer(ctx, p.parsedConfig, p.substitutionContext, mergedConfig, buildInfo)
	if err != nil {
		return nil, fmt.Errorf("runner run container: %w", err)
	}

	containerDetails, err := r.findRunningContainerOrFail(ctx, "creation")
	if err != nil {
		return nil, err
	}

	return &resolvedContainer{
		details:      containerDetails,
		mergedConfig: mergedConfig,
	}, nil
}

// deleteForRecreate removes the existing container before recreating it.
// Docker containers are fully deleted; other drivers stop the container.
func (r *runner) deleteForRecreate(ctx context.Context) error {
	if _, ok := r.Driver.(driver.DockerDriver); ok {
		if err := r.Delete(ctx); err != nil {
			return fmt.Errorf("delete devcontainer: %w", err)
		}
		return nil
	}

	if err := r.Driver.StopDevContainer(ctx, r.ID); err != nil {
		return fmt.Errorf("stop devcontainer: %w", err)
	}
	return nil
}

// injectDaemonEntrypoint adds the workspace daemon config to the container
// environment when platform configuration is provided.
func (r *runner) injectDaemonEntrypoint(
	p *resolveParams,
	mergedConfig *config.MergedDevContainerConfig,
) {
	if p.options.Platform.AccessKey == "" {
		return
	}
	r.Log.Debugf("Platform config detected, injecting DevPod daemon entrypoint.")

	data, err := agent.GetEncodedWorkspaceDaemonConfig(
		p.options.Platform,
		r.WorkspaceConfig.Workspace,
		p.substitutionContext,
		mergedConfig,
	)
	if err != nil {
		r.Log.Errorf("Failed to marshal daemon config: %v", err)
		return
	}
	mergedConfig.ContainerEnv[config.WorkspaceDaemonConfigExtraEnvVar] = data
}

// findRunningContainerOrFail re-fetches container details from the driver and
// returns an error if the container cannot be found. Use after any operation
// that changes container state (start, create, reprovision) to ensure callers
// always receive current, non-nil details.
func (r *runner) findRunningContainerOrFail(
	ctx context.Context,
	operation string,
) (*config.ContainerDetails, error) {
	details, err := r.Driver.FindDevContainer(ctx, r.ID)
	if err != nil {
		return nil, fmt.Errorf("find dev container after %s: %w", operation, err)
	}
	if details == nil {
		return nil, fmt.Errorf("dev container %s not found after %s", r.ID, operation)
	}
	return details, nil
}

func (r *runner) runContainer(
	ctx context.Context,
	parsedConfig *config.SubstitutedConfig,
	substitutionContext *config.SubstitutionContext,
	mergedConfig *config.MergedDevContainerConfig,
	buildInfo *config.BuildInfo,
) error {
	var err error

	// build run options for dockerless mode
	var runOptions *driver.RunOptions
	if buildInfo.Dockerless != nil {
		runOptions, err = r.getDockerlessRunOptions(mergedConfig, substitutionContext, buildInfo)
		if err != nil {
			return fmt.Errorf("build dockerless run options: %w", err)
		}
	} else {
		// build run options
		runOptions, err = r.getRunOptions(mergedConfig, substitutionContext, buildInfo)
		if err != nil {
			return fmt.Errorf("build run options: %w", err)
		}
	}

	runOptions.Env = r.addExtraEnvVars(runOptions.Env)

	// check if docker
	dockerDriver, ok := r.Driver.(driver.DockerDriver)
	if ok {
		return dockerDriver.RunDockerDevContainer(ctx, &driver.RunDockerDevContainerParams{
			WorkspaceID:          r.ID,
			Options:              runOptions,
			ParsedConfig:         parsedConfig.Config,
			IDE:                  r.WorkspaceConfig.Workspace.IDE.Name,
			IDEOptions:           r.WorkspaceConfig.Workspace.IDE.Options,
			LocalWorkspaceFolder: r.LocalWorkspaceFolder,
		})
	}

	// build run options for regular driver
	return r.Driver.RunDevContainer(ctx, r.ID, runOptions)
}

func (r *runner) getDockerlessRunOptions(
	mergedConfig *config.MergedDevContainerConfig,
	substitutionContext *config.SubstitutionContext,
	buildInfo *config.BuildInfo,
) (*driver.RunOptions, error) {
	// parse workspace mount
	workspaceMountParsed := config.ParseMount(substitutionContext.WorkspaceMount)

	// add metadata as label here
	marshalled, err := json.Marshal(buildInfo.ImageMetadata.Raw)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	env := map[string]string{
		"DOCKERLESS":            "true",
		"DOCKERLESS_CONTEXT":    buildInfo.Dockerless.Context,
		"DOCKERLESS_DOCKERFILE": buildInfo.Dockerless.Dockerfile,
		"GODEBUG":               "http2client=0", // https://github.com/GoogleContainerTools/kaniko/issues/875
	}
	maps.Copy(env, mergedConfig.ContainerEnv)
	if buildInfo.Dockerless.Target != "" {
		env["DOCKERLESS_TARGET"] = buildInfo.Dockerless.Target
	}
	if len(buildInfo.Dockerless.BuildArgs) > 0 {
		out, err := json.Marshal(config.ObjectToList(buildInfo.Dockerless.BuildArgs))
		if err != nil {
			return nil, fmt.Errorf("marshal build args: %w", err)
		}

		env["DOCKERLESS_BUILD_ARGS"] = string(out)
	}

	image := dockerlessImage
	if r.WorkspaceConfig != nil && r.WorkspaceConfig.Agent.Dockerless.Image != "" {
		image = r.WorkspaceConfig.Agent.Dockerless.Image
	}

	// we need to add an extra mount here, because otherwise the build config might get lost
	mounts := mergedConfig.Mounts
	mounts = append(mounts, &config.Mount{
		Type:   "volume",
		Source: "dockerless-" + r.ID,
		Target: "/workspaces/.dockerless",
	})

	uid := ""
	if r.WorkspaceConfig != nil && r.WorkspaceConfig.Workspace != nil {
		uid = r.WorkspaceConfig.Workspace.UID
	}

	// build run options
	return &driver.RunOptions{
		UID:        uid,
		Image:      image,
		User:       "root",
		Entrypoint: "/.dockerless/dockerless",
		Cmd: []string{
			"start",
			"--wait",
			"--entrypoint", "/.dockerless/bin/sh",
			"--cmd", "-c",
			"--cmd", GetStartScript(mergedConfig),
			"--user", buildInfo.Dockerless.User,
		},
		Env:    env,
		CapAdd: mergedConfig.CapAdd,
		Labels: []string{
			metadata.ImageMetadataLabel + "=" + string(marshalled),
			config.UserLabel + "=" + buildInfo.Dockerless.User,
		},
		Privileged:     mergedConfig.Privileged,
		Init:           mergedConfig.Init,
		WorkspaceMount: &workspaceMountParsed,
		Mounts:         mounts,
		Userns:         substitutionContext.Userns,
		UidMap:         substitutionContext.UidMap,
		GidMap:         substitutionContext.GidMap,
	}, nil
}

func (r *runner) getRunOptions(
	mergedConfig *config.MergedDevContainerConfig,
	substitutionContext *config.SubstitutionContext,
	buildInfo *config.BuildInfo,
) (*driver.RunOptions, error) {
	// parse workspace mount
	workspaceMountParsed := config.ParseMount(substitutionContext.WorkspaceMount)

	// add metadata as label here
	marshalled, err := json.Marshal(buildInfo.ImageMetadata.Raw)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	// build labels & entrypoint
	entrypoint, cmd := GetContainerEntrypointAndArgs(mergedConfig, buildInfo.ImageDetails)

	// Get user from image details if available, otherwise use empty string
	imageUser := ""
	if buildInfo.ImageDetails != nil {
		imageUser = buildInfo.ImageDetails.Config.User
	}

	labels := []string{
		metadata.ImageMetadataLabel + "=" + string(marshalled),
		config.UserLabel + "=" + imageUser,
	}

	user := imageUser
	if mergedConfig.ContainerUser != "" {
		user = mergedConfig.ContainerUser
	}

	uid := ""
	if r.WorkspaceConfig != nil && r.WorkspaceConfig.Workspace != nil {
		uid = r.WorkspaceConfig.Workspace.UID
	}

	return &driver.RunOptions{
		UID:            uid,
		Image:          buildInfo.ImageName,
		User:           user,
		Entrypoint:     entrypoint,
		Cmd:            cmd,
		Env:            mergedConfig.ContainerEnv,
		CapAdd:         mergedConfig.CapAdd,
		Labels:         labels,
		Privileged:     mergedConfig.Privileged,
		Init:           mergedConfig.Init,
		WorkspaceMount: &workspaceMountParsed,
		SecurityOpt:    mergedConfig.SecurityOpt,
		Mounts:         mergedConfig.Mounts,
		Userns:         substitutionContext.Userns,
		UidMap:         substitutionContext.UidMap,
		GidMap:         substitutionContext.GidMap,
	}, nil
}

// add environment variables that signals that we are in a remote container
// (vscode compatibility) and specifically that we are using devpod.
func (r *runner) addExtraEnvVars(env map[string]string) map[string]string {
	if env == nil {
		env = make(map[string]string)
	}

	env[DevPodExtraEnvVar] = "true"
	env[RemoteContainersExtraEnvVar] = "true"
	if r.WorkspaceConfig != nil && r.WorkspaceConfig.Workspace != nil &&
		r.WorkspaceConfig.Workspace.ID != "" {
		env[WorkspaceIDExtraEnvVar] = r.WorkspaceConfig.Workspace.ID
	}
	if r.WorkspaceConfig != nil && r.WorkspaceConfig.Workspace != nil &&
		r.WorkspaceConfig.Workspace.UID != "" {
		env[WorkspaceUIDExtraEnvVar] = r.WorkspaceConfig.Workspace.UID
	}

	return env
}

func GetStartScript(mergedConfig *config.MergedDevContainerConfig) string {
	customEntrypoints := mergedConfig.Entrypoints
	return `echo Container started
trap "exit 0" 15
exec "$@"
` + strings.Join(customEntrypoints, "\n") + DefaultEntrypoint
}

func GetContainerEntrypointAndArgs(
	mergedConfig *config.MergedDevContainerConfig,
	imageDetails *config.ImageDetails,
) (string, []string) {
	cmd := []string{
		"-c",
		GetStartScript(mergedConfig),
		"-",
	} // `wait $!` allows for the `trap` to run (synchronous `sleep` would not).
	if imageDetails != nil && mergedConfig.OverrideCommand != nil &&
		!*mergedConfig.OverrideCommand {
		cmd = append(cmd, imageDetails.Config.Entrypoint...)
		cmd = append(cmd, imageDetails.Config.Cmd...)
	}
	return "/bin/sh", cmd
}
