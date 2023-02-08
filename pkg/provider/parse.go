package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"io"
	"regexp"
	"time"
)

var providerNameRegEx = regexp.MustCompile(`[^a-z0-9\-]+`)

var optionNameRegEx = regexp.MustCompile(`[^A-Z0-9_]+`)

func ParseProvider(reader io.Reader) (*ProviderConfig, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := yaml.YAMLToJSON(payload)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.DisallowUnknownFields()

	parsedConfig := &ProviderConfig{}
	err = decoder.Decode(parsedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "parse provider config")
	}

	err = validate(parsedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "validate")
	}

	return parsedConfig, nil
}

var validServerStages = map[string]bool{
	"init":    true,
	"command": true,
	"status":  true,
	"create":  true,
	"delete":  true,
	"start":   true,
	"stop":    true,
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

		if optionValue.Default != "" && optionValue.Command != "" {
			return fmt.Errorf("default and command cannot be used together in option '%s'", optionName)
		}

		if optionValue.After != "" && optionValue.Before != "" {
			return fmt.Errorf("after and before cannot be used together in option '%s'", optionName)
		}

		if optionValue.After != "" && !validServerStages[optionValue.After] {
			return fmt.Errorf("invalid after stage in option '%s': %s", optionName, optionValue.After)
		}

		if optionValue.Before != "" && !validServerStages[optionValue.Before] {
			return fmt.Errorf("invalid before stage in option '%s': %s", optionName, optionValue.Before)
		}

		if optionValue.Cache != "" {
			_, err := time.ParseDuration(optionValue.Cache)
			if err != nil {
				return fmt.Errorf("invalid cache value for option '%s': %v", optionName, err)
			}
		}

		if optionValue.Required && (optionValue.Before != "" || optionValue.After != "") {
			return fmt.Errorf("required cannot be used together with before or afte in option '%s'", optionName)
		}

		if optionValue.Cache != "" && optionValue.Command == "" {
			return fmt.Errorf("cache can only be used with command in option '%s'", optionName)
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
