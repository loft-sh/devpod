package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

var ProviderNameRegEx = regexp.MustCompile(`[^a-z0-9\-]+`)

var optionNameRegEx = regexp.MustCompile(`[^A-Z0-9_]+`)

var allowedTypes = []string{
	"string",
	"duration",
	"number",
	"boolean",
}

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

func validate(config *ProviderConfig) error {
	// validate name
	if config.Name == "" {
		return fmt.Errorf("name is missing in provider.yaml")
	}
	if ProviderNameRegEx.MatchString(config.Name) {
		return fmt.Errorf("provider name can only include smaller case letters, numbers or dashes")
	} else if len(config.Name) > 32 {
		return fmt.Errorf("provider name cannot be longer than 32 characters")
	}

	// validate version
	if config.Version != "" {
		_, err := semver.Parse(strings.TrimPrefix(config.Version, "v"))
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
				return fmt.Errorf("error parsing validation pattern '%s' for option '%s': %w", optionValue.ValidationPattern, optionName, err)
			}
		}

		if optionValue.Default != "" && optionValue.Command != "" {
			return fmt.Errorf("default and command cannot be used together in option '%s'", optionName)
		}

		if optionValue.Global && optionValue.Cache != "" {
			return fmt.Errorf("global and cache cannot be used together in option '%s'", optionName)
		}

		if optionValue.Cache != "" {
			_, err := time.ParseDuration(optionValue.Cache)
			if err != nil {
				return fmt.Errorf("invalid cache value for option '%s': %w", optionName, err)
			}
		}

		if optionValue.Type != "" && !contains(allowedTypes, optionValue.Type) {
			return fmt.Errorf("type can only be one of in option '%s': %v", optionName, allowedTypes)
		}

		if optionValue.Cache != "" && optionValue.Command == "" {
			return fmt.Errorf("cache can only be used with command in option '%s'", optionName)
		}
	}

	// validate driver
	if config.Agent.Driver != "" && config.Agent.Driver != DockerDriver && config.Agent.Driver != KubernetesDriver {
		return fmt.Errorf("agent.driver can only be docker or kubernetes")
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
	if len(config.Exec.Command) == 0 {
		return fmt.Errorf("exec.command is required")
	}
	if len(config.Exec.Create) > 0 && len(config.Exec.Delete) == 0 {
		return fmt.Errorf("exec.delete is required")
	}
	if len(config.Exec.Create) == 0 && len(config.Exec.Delete) > 0 {
		return fmt.Errorf("exec.create is required")
	}
	if len(config.Exec.Start) == 0 && len(config.Exec.Stop) > 0 {
		return fmt.Errorf("exec.start is required")
	}
	if len(config.Exec.Stop) == 0 && len(config.Exec.Start) > 0 {
		return fmt.Errorf("exec.start is required")
	}
	if len(config.Exec.Status) == 0 && len(config.Exec.Start) > 0 {
		return fmt.Errorf("exec.status is required")
	}
	if len(config.Exec.Create) == 0 && len(config.Exec.Start) > 0 {
		return fmt.Errorf("exec.create is required")
	}

	err = validateOptionGroups(config)
	if err != nil {
		return err
	}

	return nil
}

func validateOptionGroups(config *ProviderConfig) error {
	foundOptions := map[string]bool{}
	for idx, group := range config.OptionGroups {
		if group.Name == "" {
			return fmt.Errorf("optionGroups[%d].name cannot be empty", idx)
		}

		for _, option := range group.Options {
			if config.Options == nil || config.Options[option] == nil {
				return fmt.Errorf("option '%s' in option group '%s' was not found under options", option, group.Name)
			} else if foundOptions[option] {
				return fmt.Errorf("option '%s' is used in multiple option groups", option)
			}

			foundOptions[option] = true
		}
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

func ParseOptions(provider *ProviderConfig, options []string) (map[string]string, error) {
	providerOptions := provider.Options
	if providerOptions == nil {
		providerOptions = map[string]*ProviderOption{}
	}

	allowedOptions := []string{}
	for optionName := range providerOptions {
		allowedOptions = append(allowedOptions, optionName)
	}

	retMap := map[string]string{}
	for _, option := range options {
		splitted := strings.Split(option, "=")
		if len(splitted) == 1 {
			return nil, fmt.Errorf("invalid option '%s', expected format KEY=VALUE", option)
		}

		key := strings.ToUpper(strings.TrimSpace(splitted[0]))
		value := strings.Join(splitted[1:], "=")
		providerOption := providerOptions[key]
		if providerOption == nil {
			return nil, fmt.Errorf("invalid option '%s', allowed options are: %v", key, allowedOptions)
		}

		if providerOption.ValidationPattern != "" {
			matcher, err := regexp.Compile(providerOption.ValidationPattern)
			if err != nil {
				return nil, err
			}

			if !matcher.MatchString(value) {
				if providerOption.ValidationMessage != "" {
					return nil, fmt.Errorf(providerOption.ValidationMessage)
				}

				return nil, fmt.Errorf("invalid value '%s' for option '%s', has to match the following regEx: %s", value, key, providerOption.ValidationPattern)
			}
		}

		if len(providerOption.Enum) > 0 {
			found := false
			for _, e := range providerOption.Enum {
				if value == e {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("invalid value '%s' for option '%s', has to match one of the following values: %v", value, key, providerOption.Enum)
			}
		}

		if providerOption.Type != "" {
			if providerOption.Type == "number" {
				_, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid value '%s' for option '%s', must be a number", value, key)
				}
			} else if providerOption.Type == "boolean" {
				_, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("invalid value '%s' for option '%s', must be a boolean", value, key)
				}
			} else if providerOption.Type == "duration" {
				_, err := time.ParseDuration(value)
				if err != nil {
					return nil, fmt.Errorf("invalid value '%s' for option '%s', must be a duration like 10s, 5m or 24h", value, key)
				}
			}
		}

		retMap[key] = value
	}

	return retMap, nil
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
