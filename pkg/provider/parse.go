package provider

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
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
	"multiline",
	"duration",
	"number",
	"boolean",
}

func ParseProvider(reader io.Reader) (*ProviderConfig, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	parsedConfig := &ProviderConfig{}
	err = yaml.Unmarshal(payload, parsedConfig)
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
		return fmt.Errorf("provider name can only include lowercase letters, numbers or dashes")
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

		if optionValue.Global && optionValue.Mutable {
			return fmt.Errorf("global and mutable cannot be used together in option '%s'", optionName)
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

	// validate provider binaries
	err := validateBinaries("binaries", config.Binaries)
	if err != nil {
		return err
	}
	err = validateProviderType(config)
	if err != nil {
		return err
	}

	err = validateOptionGroups(config)
	if err != nil {
		return err
	}

	return nil
}

func validateProviderType(config *ProviderConfig) error {
	if config.IsProxyProvider() {
		if !reflect.DeepEqual(config.Agent, ProviderAgentConfig{}) {
			return fmt.Errorf("agent config is not allowed for proxy providers")
		}
		if len(config.Exec.Command) > 0 {
			return fmt.Errorf("exec.command is not allowed in proxy providers")
		}
		if len(config.Exec.Create) > 0 {
			return fmt.Errorf("exec.create is not allowed in proxy providers")
		}
		if len(config.Exec.Start) > 0 {
			return fmt.Errorf("exec.create is not allowed in proxy providers")
		}
		if len(config.Exec.Stop) > 0 {
			return fmt.Errorf("exec.create is not allowed in proxy providers")
		}
		if len(config.Exec.Status) > 0 {
			return fmt.Errorf("exec.create is not allowed in proxy providers")
		}
		if len(config.Exec.Delete) > 0 {
			return fmt.Errorf("exec.create is not allowed in proxy providers")
		}
		if len(config.Exec.Proxy.Status) == 0 {
			return fmt.Errorf("exec.proxy.status is required for proxy providers")
		}
		if len(config.Exec.Proxy.Stop) == 0 {
			return fmt.Errorf("exec.proxy.stop is required for proxy providers")
		}
		if len(config.Exec.Proxy.Delete) == 0 {
			return fmt.Errorf("exec.proxy.delete is required for proxy providers")
		}
		if len(config.Exec.Proxy.Ssh) == 0 {
			return fmt.Errorf("exec.proxy.ssh is required for proxy providers")
		}
		if len(config.Exec.Proxy.Up) == 0 {
			return fmt.Errorf("exec.proxy.up is required for proxy providers")
		}

		return nil
	}

	// daemon provider
	if config.IsDaemonProvider() {
		if !reflect.DeepEqual(config.Agent, ProviderAgentConfig{}) {
			return fmt.Errorf("agent config is not allowed for daemon providers")
		}
		if len(config.Exec.Command) > 0 {
			return fmt.Errorf("exec.command is not allowed in daemon providers")
		}
		if len(config.Exec.Create) > 0 {
			return fmt.Errorf("exec.create is not allowed in daemon providers")
		}
		if len(config.Exec.Start) > 0 {
			return fmt.Errorf("exec.create is not allowed in daemon providers")
		}
		if len(config.Exec.Stop) > 0 {
			return fmt.Errorf("exec.create is not allowed in daemon providers")
		}
		if len(config.Exec.Status) > 0 {
			return fmt.Errorf("exec.create is not allowed in daemon providers")
		}
		if len(config.Exec.Delete) > 0 {
			return fmt.Errorf("exec.create is not allowed in daemon providers")
		}
		if len(config.Exec.Daemon.Start) == 0 {
			return fmt.Errorf("exec.daemon.start is required for daemon providers")
		}

		return nil
	}

	// validate driver
	if config.Agent.Driver != "" && config.Agent.Driver != CustomDriver && config.Agent.Driver != DockerDriver && config.Agent.Driver != KubernetesDriver {
		return fmt.Errorf("agent.driver can only be docker, kubernetes or custom")
	}

	// validate custom driver
	if config.Agent.Driver == CustomDriver {
		if len(config.Agent.Custom.TargetArchitecture) == 0 {
			return fmt.Errorf("agent.custom.targetArchitecture is required")
		}
		if len(config.Agent.Custom.StartDevContainer) == 0 {
			return fmt.Errorf("agent.custom.startDevContainer is required")
		}
		if len(config.Agent.Custom.StopDevContainer) == 0 {
			return fmt.Errorf("agent.custom.stopDevContainer is required")
		}
		if len(config.Agent.Custom.RunDevContainer) == 0 {
			return fmt.Errorf("agent.custom.runDevContainer is required")
		}
		if len(config.Agent.Custom.DeleteDevContainer) == 0 {
			return fmt.Errorf("agent.custom.deleteDevContainer is required")
		}
		if len(config.Agent.Custom.FindDevContainer) == 0 {
			return fmt.Errorf("agent.custom.findDevContainer is required")
		}
		if len(config.Agent.Custom.CommandDevContainer) == 0 {
			return fmt.Errorf("agent.custom.commandDevContainer is required")
		}
		// TODO: Add config.Agent.Custom.GetDevContainerLogs validation
		// after we released a new version of the kubernetes provider and gave folks a chance to update
	}

	// agent binaries
	err := validateBinaries("agent.binaries", config.Agent.Binaries)
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

	return nil
}

func validateOptionGroups(config *ProviderConfig) error {
	for idx, group := range config.OptionGroups {
		if group.Name == "" {
			return fmt.Errorf("optionGroups[%d].name cannot be empty", idx)
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

func ParseOptions(options []string) (map[string]string, error) {
	retMap := map[string]string{}
	for _, option := range options {
		splitted := strings.Split(option, "=")
		if len(splitted) == 1 {
			return nil, fmt.Errorf("invalid option '%s', expected format KEY=VALUE", option)
		}

		key := strings.ToUpper(strings.TrimSpace(splitted[0]))
		value := strings.Join(splitted[1:], "=")

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
