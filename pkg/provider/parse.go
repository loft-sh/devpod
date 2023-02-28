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
	"strings"
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

var ValidServerStages = map[string]bool{
	"init":    true,
	"command": true,
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

		if optionValue.After != "" && !ValidServerStages[optionValue.After] {
			return fmt.Errorf("invalid after stage in option '%s': %s", optionName, optionValue.After)
		}

		if optionValue.Before != "" && !ValidServerStages[optionValue.Before] {
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

	// validate provider binaries
	err := validateBinaries("binaries", config.Binaries)
	if err != nil {
		return err
	}
	err = validateBinaries("agent.binaries", config.Agent.Binaries)
	if err != nil {
		return err
	}

	// validate provider type
	if config.Type == "" || config.Type == ProviderTypeServer {
		if len(config.Exec.Command) == 0 {
			return fmt.Errorf("exec.command is required")
		}
		if len(config.Exec.Create) > 0 && len(config.Exec.Delete) == 0 {
			return fmt.Errorf("exec.delete is required")
		}
		if len(config.Exec.Create) == 0 && len(config.Exec.Delete) > 0 {
			return fmt.Errorf("exec.create is required")
		}
	} else if config.Type == ProviderTypeDirect {
		if len(config.Exec.Command) == 0 {
			return fmt.Errorf("exec.command is required")
		}
	} else {
		return fmt.Errorf("provider type '%s' unrecognized, either choose '%s' or '%s'", config.Type, ProviderTypeServer, ProviderTypeDirect)
	}

	return nil
}

func validateBinaries(prefix string, binaries map[string][]*ProviderBinary) error {
	for binaryName, binaryArr := range binaries {
		if optionNameRegEx.MatchString(binaryName) {
			return fmt.Errorf("binary name '%s' can only consist of upper case letters, numbers or underscores. E.g. MY_BINARY, KUBECTL", binaryName)
		}

		for _, binary := range binaryArr {
			if binary.OS != "linux" && binary.OS != "darwin" && binary.OS != "windows" {
				return fmt.Errorf("unsupported binary operating system '%s', must be 'linux', 'darwin' or 'windows'", binary.OS)
			}
			if binary.Path == "" {
				return fmt.Errorf("%s.%s.path required binary path, cannot be empty", prefix, binaryName)
			}
			if binary.ArchivePath == "" && (strings.HasSuffix(binary.Path, ".gz") || strings.HasSuffix(binary.Path, ".tar") || strings.HasSuffix(binary.Path, ".tgz") || strings.HasSuffix(binary.Path, ".zip")) {
				return fmt.Errorf("%s.%s.archivePath required because binary path is an archive", prefix, binaryName)
			}
			if binary.Arch == "" {
				return fmt.Errorf("%s.%s.arch required, cannot be empty", prefix, binaryName)
			}
		}
	}

	return nil
}
