package config

import (
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

type Config struct {
	// DefaultContext is the default context to use. Defaults to "default"
	DefaultContext string `json:"defaultContext,omitempty"`

	// Contexts holds the config contexts
	Contexts map[string]*ConfigContext `json:"contexts,omitempty"`
}

type ConfigContext struct {
	// DefaultProvider is the default provider to use
	DefaultProvider string `json:"defaultProvider,omitempty"`

	// Providers holds the provider configuration
	Providers map[string]*ConfigProvider `json:"providers,omitempty"`
}

type ConfigProvider struct {
	// Options are the configured provider options
	Options map[string]provider.OptionValue `json:"options,omitempty"`
}

var ConfigFile = "config.yaml"

const DefaultContext = "default"

func LoadConfig(contextOverride string) (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	configBytes, err := os.ReadFile(filepath.Join(configDir, ConfigFile))
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
		}, nil
	}

	config := &Config{}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		return nil, err
	}
	if contextOverride != "" {
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

	return config, nil
}

func SaveConfig(config *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(configDir, ConfigFile), out, 0666)
	if err != nil {
		return err
	}

	return nil
}
