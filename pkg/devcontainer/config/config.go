package config

import (
	"encoding/json"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/pkg/types"
)

type MergedDevContainerConfig struct {
	DevContainerConfigBase  `json:",inline"`
	UpdatedConfigProperties `json:",inline"`
	NonComposeBase          `json:",inline"`
	ImageContainer          `json:",inline"`
	ComposeContainer        `json:",inline"`
	DockerfileContainer     `json:",inline"`
	RunningContainer        `json:",inline"`

	// Origin is the origin from where this config was loaded
	Origin string `json:"-"`
}

type DevContainerConfig struct {
	DevContainerConfigBase `json:",inline"`
	DevContainerActions    `json:",inline"`
	NonComposeBase         `json:",inline"`
	ImageContainer         `json:",inline"`
	ComposeContainer       `json:",inline"`
	DockerfileContainer    `json:",inline"`
	RunningContainer       `json:",inline"`

	// Origin is the origin from where this config was loaded
	Origin string `json:"-"`
}

func CloneDevContainerConfig(config *DevContainerConfig) *DevContainerConfig {
	out := &DevContainerConfig{}
	_ = Convert(config, out)
	out.Origin = config.Origin
	return out
}

type DevContainerConfigBase struct {
	// A name for the dev container which can be displayed to the user.
	Name string `json:"name,omitempty"`

	// Features to add to the dev container.
	Features map[string]interface{} `json:"features,omitempty"`

	// Array consisting of the Feature id (without the semantic version) of Features in the order the user wants them to be installed.
	OverrideFeatureInstallOrder []string `json:"overrideFeatureInstallOrder,omitempty"`

	// Ports that are forwarded from the container to the local machine. Can be an integer port number, or a string of the format "host:port_number".
	ForwardPorts types.StrIntArray `json:"forwardPorts,omitempty"`

	// Set default properties that are applied when a specific port number is forwarded.
	PortsAttributes map[string]PortAttribute `json:"portAttributes,omitempty"`

	// Set default properties that are applied to all ports that don't get properties from the setting `remote.portsAttributes`.
	OtherPortsAttributes *PortAttribute `json:"otherPortsAttributes,omitempty"`

	// Controls whether on Linux the container's user should be updated with the local user's UID and GID. On by default when opening from a local folder.
	UpdateRemoteUserUID *bool `json:"updateRemoteUserUID,omitempty"`

	// Remote environment variables to set for processes spawned in the container including lifecycle scripts and any remote editor/IDE server process.
	RemoteEnv map[string]string `json:"remoteEnv,omitempty"`

	// The username to use for spawning processes in the container including lifecycle scripts and any remote editor/IDE server process. The default is the same user as the container.
	RemoteUser string `json:"remoteUser,omitempty"`

	// A command to run locally before anything else. This command is run before "onCreateCommand". If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	InitializeCommand types.LifecycleHook `json:"initializeCommand,omitempty"`

	// Action to take when the user disconnects from the container in their editor. The default is to stop the container.
	ShutdownAction string `json:"shutdownAction,omitempty"`

	// The user command to wait for before continuing execution in the background while the UI is starting up. The default is "updateContentCommand".
	WaitFor string `json:"waitFor,omitempty"`

	// User environment probe to run. The default is "loginInteractiveShell".
	UserEnvProbe string `json:"userEnvProbe,omitempty"`

	// Host hardware requirements.
	HostRequirements *HostRequirements `json:"hostRequirements,omitempty"`

	// Whether to overwrite the command specified in the image. The default is true.
	OverrideCommand *bool `json:"overrideCommand,omitempty"`

	// The path of the workspace folder inside the container.
	WorkspaceFolder string `json:"workspaceFolder,omitempty"`

	// DEPRECATED: Use 'customizations/vscode/settings' instead
	// Machine specific settings that should be copied into the container. These are only copied when connecting to the container for the first time, rebuilding the container then triggers it again.
	Settings map[string]interface{} `json:"settings,omitempty"`

	// DEPRECATED: Use 'customizations/vscode/extensions' instead
	// An array of extensions that should be installed into the container.
	Extensions []string `json:"extensions,omitempty"`

	// DEPRECATED: Use 'customizations/vscode/devPort' instead
	// The port VS Code can use to connect to its backend.
	DevPort int `json:"devPort,omitempty"`
}

type DevContainerActions struct {
	// A command to run when creating the container. This command is run after "initializeCommand" and before "updateContentCommand". If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	OnCreateCommand types.LifecycleHook `json:"onCreateCommand,omitempty"`

	// A command to run when creating the container and rerun when the workspace content was updated while creating the container.
	// This command is run after "onCreateCommand" and before "postCreateCommand". If this is a single string, it will be run in a shell.
	// If this is an array of strings, it will be run as a single command without shell.
	UpdateContentCommand types.LifecycleHook `json:"updateContentCommand,omitempty"`

	// A command to run after creating the container. This command is run after "updateContentCommand" and before "postStartCommand".
	// If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	PostCreateCommand types.LifecycleHook `json:"postCreateCommand,omitempty"`

	// A command to run after starting the container. This command is run after "postCreateCommand" and before "postAttachCommand".
	// If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	PostStartCommand types.LifecycleHook `json:"postStartCommand,omitempty"`

	// A command to run when attaching to the container. This command is run after "postStartCommand".
	// If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	PostAttachCommand types.LifecycleHook `json:"postAttachCommand,omitempty"`

	// Tool-specific configuration. Each tool should use a JSON object subproperty with a unique name to group its customizations.
	Customizations map[string]interface{} `json:"customizations,omitempty"`
}

type UpdatedConfigProperties struct {
	// Entrypoint script that should fire at container start up.
	Entrypoints []string `json:"entrypoints,omitempty"`

	// A command to run when creating the container. This command is run after "initializeCommand" and before "updateContentCommand". If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	OnCreateCommands []types.LifecycleHook `json:"onCreateCommand,omitempty"`

	// A command to run when creating the container and rerun when the workspace content was updated while creating the container.
	// This command is run after "onCreateCommand" and before "postCreateCommand". If this is a single string, it will be run in a shell.
	// If this is an array of strings, it will be run as a single command without shell.
	UpdateContentCommands []types.LifecycleHook `json:"updateContentCommand,omitempty"`

	// A command to run after creating the container. This command is run after "updateContentCommand" and before "postStartCommand".
	// If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	PostCreateCommands []types.LifecycleHook `json:"postCreateCommand,omitempty"`

	// A command to run after starting the container. This command is run after "postCreateCommand" and before "postAttachCommand".
	// If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	PostStartCommands []types.LifecycleHook `json:"postStartCommand,omitempty"`

	// A command to run when attaching to the container. This command is run after "postStartCommand".
	// If this is a single string, it will be run in a shell. If this is an array of strings, it will be run as a single command without shell.
	PostAttachCommands []types.LifecycleHook `json:"postAttachCommand,omitempty"`

	// Tool-specific configuration. Each tool should use a JSON object subproperty with a unique name to group its customizations.
	Customizations map[string][]interface{} `json:"customizations,omitempty"`
}

type ComposeContainer struct {
	// The name of the docker-compose file(s) used to start the services.
	DockerComposeFile types.StrArray `json:"dockerComposeFile,omitempty"`

	// The service you want to work on. This is considered the primary container for your dev environment which your editor will connect to.
	Service string `json:"service,omitempty"`

	// An array of services that should be started and stopped.
	RunServices []string `json:"runServices,omitempty"`
}

type ImageContainer struct {
	// The docker image that will be used to create the container.
	Image string `json:"image,omitempty"`
}

type NonComposeBase struct {
	// Application ports that are exposed by the container. This can be a single port or an array of ports. Each port can be a number or a string.
	// A number is mapped to the same port on the host. A string is passed to Docker unchanged and can be used to map ports differently,
	// e.g. "8000:8010".
	AppPort types.StrIntArray `json:"appPort,omitempty"`

	// Container environment variables.
	ContainerEnv map[string]string `json:"containerEnv,omitempty"`

	// The user the container will be started with. The default is the user on the Docker image.
	ContainerUser string `json:"containerUser,omitempty"`

	// Mounts points to set up when creating the container. See Docker's documentation for the --mount option for the supported syntax.
	Mounts []*Mount `json:"mounts,omitempty"`

	// Passes the --init flag when creating the dev container.
	Init *bool `json:"init,omitempty"`

	// Passes the --privileged flag when creating the dev container.
	Privileged *bool `json:"privileged,omitempty"`

	// Passes docker capabilities to include when creating the dev container.
	CapAdd []string `json:"capAdd,omitempty"`

	// Passes docker security options to include when creating the dev container.
	SecurityOpt []string `json:"securityOpt,omitempty"`

	// The arguments required when starting in the container.
	RunArgs []string `json:"runArgs,omitempty"`

	// The --mount parameter for docker run. The default is to mount the project folder at /workspaces/$project.
	WorkspaceMount string `json:"workspaceMount,omitempty"`
}

type DockerfileContainer struct {
	// The location of the Dockerfile that defines the contents of the container. The path is relative to the folder containing the `devcontainer.json` file.
	Dockerfile string `json:"dockerFile,omitempty"`

	// The location of the context folder for building the Docker image. The path is relative to the folder containing the `devcontainer.json` file.
	Context string `json:"context,omitempty"`

	// Docker build-related options.
	Build *ConfigBuildOptions `json:"build,omitempty"`
}

type RunningContainer struct {
	ContainerID string `json:"containerID,omitempty"`
}

func (d DockerfileContainer) GetDockerfile() string {
	if d.Dockerfile != "" {
		return d.Dockerfile
	}
	if d.Build != nil && d.Build.Dockerfile != "" {
		return d.Build.Dockerfile
	}
	return ""
}

func (d DockerfileContainer) GetContext() string {
	if d.Context != "" {
		return d.Context
	}
	if d.Build != nil && d.Build.Context != "" {
		return d.Build.Context
	}
	return ""
}

func (d DockerfileContainer) GetTarget() string {
	if d.Build != nil {
		return d.Build.Target
	}
	return ""
}

func (d DockerfileContainer) GetArgs() map[string]string {
	if d.Build != nil {
		return d.Build.Args
	}
	return nil
}

func (d DockerfileContainer) GetOptions() []string {
	if d.Build != nil {
		return d.Build.Options
	}
	return nil
}

func (d DockerfileContainer) GetCacheFrom() types.StrArray {
	if d.Build != nil {
		return d.Build.CacheFrom
	}
	return nil
}

type ConfigBuildOptions struct {
	// The location of the Dockerfile that defines the contents of the container. The path is relative to the folder containing the `devcontainer.json` file.
	Dockerfile string `json:"dockerfile,omitempty"`

	// The location of the context folder for building the Docker image. The path is relative to the folder containing the `devcontainer.json` file.
	Context string `json:"context,omitempty"`

	// Target stage in a multi-stage build.
	Target string `json:"target,omitempty"`

	// Build arguments.
	Args map[string]string `json:"args,omitempty"`

	// The image to consider as a cache. Use an array to specify multiple images.
	CacheFrom types.StrArray `json:"cacheFrom,omitempty"`

	// Build cli options
	Options []string `json:"options,omitempty"`
}

type HostRequirements struct {
	// Number of required CPUs.
	CPUs int `json:"cpus,omitempty"`

	// Amount of required RAM in bytes. Supports units tb, gb, mb and kb.
	Memory string `json:"memory,omitempty"`

	// Amount of required disk space in bytes. Supports units tb, gb, mb and kb.
	Storage string `json:"storage,omitempty"`

	// If GPU support should be enabled
	GPU types.StrBool `json:"gpu,omitempty"`
}

type PortAttribute struct {
	// Defines the action that occurs when the port is discovered for automatic forwarding
	// default=notify
	OnAutoForward string `json:"onAutoForward,omitempty"`

	// Automatically prompt for elevation (if needed) when this port is forwarded. Elevate is required if the local port is a privileged port.
	ElevateIfNeeded bool `json:"elevateIfNeeded,omitempty"`

	// Label that will be shown in the UI for this port.
	// default=Application
	Label string `json:"label,omitempty"`

	// When true, a modal dialog will show if the chosen local port isn't used for forwarding.
	RequireLocalPort bool `json:"requireLocalPort,omitempty"`

	// The protocol to use when forwarding this port.
	Protocol string `json:"protocol,omitempty"`
}

type DevPodCustomizations struct {
	PrebuildRepository         types.StrArray    `json:"prebuildRepository,omitempty"`
	FeatureDownloadHTTPHeaders map[string]string `json:"featureDownloadHTTPHeaders,omitempty"`
}

type VSCodeCustomizations struct {
	Settings   map[string]interface{} `json:"settings,omitempty"`
	Extensions []string               `json:"extensions,omitempty"`
	DevPort    int                    `json:"devPort,omitempty"`
}

type Mount struct {
	Type     string   `json:"type,omitempty"`
	Source   string   `json:"source,omitempty"`
	Target   string   `json:"target,omitempty"`
	External bool     `json:"external,omitempty"`
	Other    []string `json:"other,omitempty"`
}

func (m *Mount) String() string {
	components := []string{}
	if m.Type != "" {
		components = append(components, "type="+m.Type)
	}
	if m.Source != "" {
		components = append(components, "src="+m.Source)
	}
	if m.Target != "" {
		components = append(components, "dst="+m.Target)
	}
	if m.External {
		components = append(components, "external="+strconv.FormatBool(m.External))
	}
	components = append(components, m.Other...)
	return strings.Join(components, ",")
}

func GetContextPath(parsedConfig *DevContainerConfig) string {
	context := parsedConfig.GetContext()
	dockerfilePath := parsedConfig.GetDockerfile()

	configDir := path.Dir(filepath.ToSlash(parsedConfig.Origin))
	if context != "" {
		return filepath.FromSlash(path.Join(configDir, context))
	} else if dockerfilePath != "" {
		return filepath.FromSlash(path.Join(configDir, path.Dir(dockerfilePath)))
	}

	return configDir
}

func ParseMount(str string) Mount {
	retMount := Mount{}
	splitted := strings.Split(str, ",")
	for _, split := range splitted {
		splitted2 := strings.Split(split, "=")
		key := splitted2[0]
		if key == "src" || key == "source" {
			retMount.Source = splitted2[1]
		} else if key == "dst" || key == "destination" || key == "target" {
			retMount.Target = splitted2[1]
		} else if key == "type" {
			retMount.Type = splitted2[1]
		} else if key == "external" {
			retMount.External, _ = strconv.ParseBool(splitted2[1])
		} else {
			retMount.Other = append(retMount.Other, split)
		}
	}

	return retMount
}

func (m *Mount) UnmarshalJSON(data []byte) error {
	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return err
	}
	switch obj := jsonObj.(type) {
	case string:
		*m = ParseMount(obj)
		return nil
	case map[string]interface{}:
		sourceStr, ok := obj["source"].(string)
		if ok {
			m.Source = sourceStr
		}
		targetStr, ok := obj["target"].(string)
		if ok {
			m.Target = targetStr
		}
		typeStr, ok := obj["type"].(string)
		if ok {
			m.Type = typeStr
		}
		externalStr, ok := obj["external"].(bool)
		if ok {
			m.External = externalStr
		}
		otherInterface, ok := obj["other"].([]interface{})
		if ok {
			otherStr := make([]string, len(otherInterface))
			for i := range otherInterface {
				otherStr[i] = otherInterface[i].(string)
			}
			m.Other = otherStr
		}
		return nil
	}
	return types.ErrUnsupportedType
}
