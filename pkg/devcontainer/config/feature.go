package config

import "github.com/loft-sh/devpod/pkg/types"

type FeatureSet struct {
	ConfigID string
	Folder   string
	Config   *FeatureConfig
	Options  interface{}
}

type FeatureConfig struct {
	// ID of the Feature. The id should be unique in the context of the repository/published package where the feature exists and must match the name of the directory where the devcontainer-feature.json resides.
	ID string `json:"id,omitempty"`

	// Display name of the Feature.
	Name string `json:"name,omitempty"`

	// The version of the Feature. Follows the semanatic versioning (semver) specification.
	Version string `json:"version,omitempty"`

	// Description of the Feature. For the best appearance in an implementing tool, refrain from including markdown or HTML in the description.
	Description string `json:"description,omitempty"`

	// Entrypoint script that should fire at container start up.
	Entrypoint string `json:"entrypoint,omitempty"`

	// Indicates that the Feature is deprecated, and will not receive any further updates/support. This property is intended to be used by the supporting tools for highlighting Feature deprecation.
	Deprecated bool `json:"deprecated,omitempty"`

	// Array of old IDs used to publish this Feature. The property is useful for renaming a currently published Feature within a single namespace.
	LegacyIds []string `json:"legacyIds,omitempty"`

	// Possible user-configurable options for this Feature. The selected options will be passed as environment variables when installing the Feature into the container.
	Options map[string]FeatureConfigOption `json:"options,omitempty"`

	// URL to documentation for the Feature.
	DocumentationURL string `json:"documentationURL,omitempty"`

	// URL to the license for the Feature.
	LicenseURL string `json:"licenseURL,omitempty"`

	// Passes docker capabilities to include when creating the dev container.
	CapAdd []string `json:"capAdd,omitempty"`

	// Adds the tiny init process to the container (--init) when the Feature is used.
	Init *bool `json:"init,omitempty"`

	// Sets privileged mode (--privileged) for the container.
	Privileged *bool `json:"privileged,omitempty"`

	// Sets container security options to include when creating the container.
	SecurityOpt []string `json:"securityOpt,omitempty"`

	// Mounts a volume or bind mount into the container.
	Mounts []*Mount `json:"mounts,omitempty"`

	// Array of ID's of Features that should execute before this one. Allows control for feature authors on soft dependencies between different Features.
	InstallsAfter []string `json:"installsAfter,omitempty"`

	// Container environment variables.
	ContainerEnv map[string]string `json:"containerEnv,omitempty"`

	// Tool-specific configuration. Each tool should use a JSON object subproperty with a unique name to group its customizations.
	Customizations map[string]interface{} `json:"customizations,omitempty"`

	// Origin is the path where the feature was loaded from
	Origin string `json:"-"`
}

type FeatureConfigOption struct {
	// Default value if the user omits this option from their configuration.
	Default types.StrBool `json:"default,omitempty"`

	// A description of the option displayed to the user by a supporting tool.
	Description string `json:"description,omitempty"`

	// The type of the option. Can be 'boolean' or 'string'.  Options of type 'string' should use the 'enum' or 'proposals' property to provide a list of allowed values.
	Type string `json:"type,omitempty"`

	// Allowed values for this option.  Unlike 'proposals', the user cannot provide a custom value not included in the 'enum' array.
	Enum []string `json:"enum,omitempty"`

	// Suggested values for this option.  Unlike 'enum', the 'proposals' attribute indicates the installation script can handle arbitrary values provided by the user.
	Proposals []string `json:"proposals,omitempty"`
}
