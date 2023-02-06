package provider

import (
	"fmt"
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	"regexp"
)

var providerNameRegEx = regexp.MustCompile(`[^a-z0-9\-]+`)

var optionNameRegEx = regexp.MustCompile(`[^A-Z0-9_]+`)

func ParseProvider(reader io.Reader) (*ProviderConfig, error) {
	decoder := yaml.NewDecoder(reader)
	decoder.SetStrict(true)

	parsedConfig := &ProviderConfig{}
	err := decoder.Decode(parsedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "parse provider config")
	}

	err = validate(parsedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "validate")
	}

	return parsedConfig, nil
}

func validate(config *ProviderConfig) error {
	// validate name
	if config.Name == "" {
		return fmt.Errorf("name is missing in provider.yaml")
	}
	if providerNameRegEx.MatchString(config.Name) {
		return fmt.Errorf("provider name can only include smaller case letters, numbers or dashes")
	}

	// validate version
	if config.Version != "" {
		_, err := semver.Parse(config.Version)
		if err != nil {
			return errors.Wrap(err, "parse provider version")
		}
	}

	// validate option names
	for optionName, optionValue := range config.Options {
		if optionNameRegEx.MatchString(optionName) {
			return fmt.Errorf("provider option '%s' can only consist of upper case letters, numbers or underscores. E.g. MY_OPTION, MY_OTHER_OPTION", optionName)
		}

		// validate option validation
		if optionValue.ValidationPattern != "" {
			_, err := regexp.Compile(optionValue.ValidationPattern)
			if err != nil {
				return fmt.Errorf("error parsing validation pattern '%s' for option '%s': %v", optionValue.ValidationPattern, optionName, err)
			}
		}
	}

	// validate provider type
	if config.Type == "" || config.Type == ProviderTypeServer {
		if len(config.Exec.Command) == 0 {
			return fmt.Errorf("exec.command is required if provider is 'Server' type")
		}
		if len(config.Exec.Tunnel) > 0 {
			return fmt.Errorf("exec.tunnel is forbidden if provider is 'Server' type")
		}
	} else if config.Type == ProviderTypeWorkspace {
		if len(config.Exec.Tunnel) == 0 {
			return fmt.Errorf("exec.tunnel is required if provider is 'Workspace' type")
		}
		if len(config.Exec.Command) > 0 {
			return fmt.Errorf("exec.command is forbidden if provider is 'Workspace' type")
		}
	} else {
		return fmt.Errorf("provider type '%s' unrecognized, either choose '%s' or '%s'", config.Type, ProviderTypeServer, ProviderTypeWorkspace)
	}

	return nil
}
