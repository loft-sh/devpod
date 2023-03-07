package config

import (
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

type Config struct {
	// DefaultContext is the default context to use. Defaults to "default"
	DefaultContext string `json:"defaultContext,omitempty"`

	// Contexts holds the config contexts
	Contexts map[string]*ConfigContext `json:"contexts,omitempty"`

	// Origin holds the path where this config was loaded from
	Origin string `json:"-"`

	// OriginalContext is the original default context
	OriginalContext string `json:"-"`
}

type ConfigContext struct {
	// DefaultProvider is the default provider to use
	DefaultProvider string `json:"defaultProvider,omitempty"`

	// Providers holds the provider configuration
	Providers map[string]*ConfigProvider `json:"providers,omitempty"`
}

type ConfigProvider struct {
	// Options are the configured provider options
	Options map[string]OptionValue `json:"options,omitempty"`
}

type OptionValue struct {
	// Value is the value of the option
	Value string `json:"value,omitempty"`

	// UserProvided signals that this value was user provided
	UserProvided bool `json:"userProvided,omitempty"`

	// Filled is the time when this value was filled
	Filled *types.Time `json:"filled,omitempty"`
}

func (c *Config) Current() *ConfigContext {
	return c.Contexts[c.DefaultContext]
}

func (c *Config) ProviderOptions(provider string) map[string]OptionValue {
	return c.Current().ProviderOptions(provider)
}

func (c *ConfigContext) ProviderOptions(provider string) map[string]OptionValue {
	retOptions := map[string]OptionValue{}
	if c.Providers == nil || c.Providers[provider] == nil {
		return retOptions
	}

	for k, v := range c.Providers[provider].Options {
		retOptions[k] = v
	}
	return retOptions
}

var ConfigFile = "config.yaml"

const DefaultContext = "default"

func CloneConfig(config *Config) *Config {
	out, _ := json.Marshal(config)
	ret := &Config{}
	_ = json.Unmarshal(out, ret)
	ret.Origin = config.Origin
	ret.OriginalContext = config.OriginalContext
	return ret
}

func LoadConfig(contextOverride string) (*Config, error) {
	configOrigin, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	configBytes, err := os.ReadFile(configOrigin)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "read config")
		}

		context := contextOverride
		if context == "" {
			context = DefaultContext
		}

		return &Config{
			DefaultContext: context,
			Contexts: map[string]*ConfigContext{
				context: {
					Providers: map[string]*ConfigProvider{},
				},
			},
			Origin: configOrigin,
		}, nil
	}

	config := &Config{}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		return nil, err
	}
	if contextOverride != "" {
		config.OriginalContext = config.DefaultContext
		config.DefaultContext = contextOverride
	} else if config.DefaultContext == "" {
		config.DefaultContext = DefaultContext
	}
	if config.Contexts == nil {
		config.Contexts = map[string]*ConfigContext{}
	}
	if config.Contexts[config.DefaultContext] == nil {
		config.Contexts[config.DefaultContext] = &ConfigContext{}
	}
	if config.Contexts[config.DefaultContext].Providers == nil {
		config.Contexts[config.DefaultContext].Providers = map[string]*ConfigProvider{}
	}

	config.Origin = configOrigin
	return config, nil
}

func SaveConfig(config *Config) error {
	configOrigin, err := GetConfigPath()
	if err != nil {
		return err
	}

	config = CloneConfig(config)
	if config.OriginalContext != "" {
		config.DefaultContext = config.OriginalContext
	}

	out, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(configOrigin), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(configOrigin, out, 0666)
	if err != nil {
		return err
	}

	return nil
}
