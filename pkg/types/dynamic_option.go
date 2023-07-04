package types

type Option struct {
	// A description of the option displayed to the user by a supporting tool.
	Description string `json:"description,omitempty"`

	// If required is true and the user doesn't supply a value, devpod will ask the user
	Required bool `json:"required,omitempty"`

	// If true, will not show the value to the user
	Password bool `json:"password,omitempty"`

	// Type is the provider option type. Can be one of: string, duration, number or boolean. Defaults to string
	Type string `json:"type,omitempty"`

	// ValidationPattern is a regex pattern to validate the value
	ValidationPattern string `json:"validationPattern,omitempty"`

	// ValidationMessage is the message that appears if the user enters an invalid option
	ValidationMessage string `json:"validationMessage,omitempty"`

	// Suggestions are suggestions to show in the DevPod UI for this option
	Suggestions []string `json:"suggestions,omitempty"`

	// Allowed values for this option.
	Enum []string `json:"enum,omitempty"`

	// Hidden specifies if the option should be hidden
	Hidden bool `json:"hidden,omitempty"`

	// Local means the variable is not resolved immediately and instead later when the workspace / machine was created.
	Local bool `json:"local,omitempty"`

	// Global means the variable is stored globally. By default, option values will be
	// saved per machine or workspace instead.
	Global bool `json:"global,omitempty"`

	// Default value if the user omits this option from their configuration.
	Default string `json:"default,omitempty"`

	// Cache is the duration to cache the value before rerunning the command
	Cache string `json:"cache,omitempty"`

	// Command is the command to run to specify an option
	Command string `json:"command,omitempty"`

	// SubOptionsCommand is the command to run to fetch sub options
	SubOptionsCommand string `json:"subOptionsCommand,omitempty"`
}
