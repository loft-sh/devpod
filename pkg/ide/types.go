package ide

import "github.com/loft-sh/devpod/pkg/config"

type IDE interface {
	Install() error
}

type Options map[string]Option

type Option struct {
	// Name is the name of the IDE option
	Name string `json:"name,omitempty"`

	// Description is the description of the IDE option
	Description string `json:"description,omitempty"`

	// Default is the default value for this option
	Default string `json:"default,omitempty"`

	// Enum is the possible values for this option
	Enum []string `json:"enum,omitempty"`

	// ValidationPattern to use to validate this option
	ValidationPattern string `json:"validationPattern,omitempty"`

	// ValidationMessage to print if validation fails
	ValidationMessage string `json:"validationMessage,omitempty"`
}

func (o Options) GetValue(values map[string]config.OptionValue, key string) string {
	if values != nil && values[key].Value != "" {
		return values[key].Value
	} else if o[key].Default != "" {
		return o[key].Default
	}

	return ""
}

// ReusesAuthSock determines if the --reuse-ssh-auth-sock flag should be passed to the ssh server helper based on the IDE.
// Browser based IDEs use a browser tunnel to communicate with the remote server instead of an independent ssh connection
func ReusesAuthSock(ide string) bool {
	return ide == "openvscode" || ide == "marimo" || ide == "jupyternotebook" || ide == "jlab"
}
