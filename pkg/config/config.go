package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
)

type Config struct {
	// DefaultContext is the default context to use. Defaults to "default"
	DefaultContext string `json:"defaultContext,omitempty"`

	// Contexts holds the config contexts
	Contexts map[string]*ContextConfig `json:"contexts,omitempty"`

	// Origin holds the path where this config was loaded from
	Origin string `json:"-"`

	// OriginalContext is the original default context
	OriginalContext string `json:"-"`
}

type ContextConfig struct {
	// DefaultProvider is the default provider to use
	DefaultProvider string `json:"defaultProvider,omitempty"`

	// DefaultIDE holds default ide configuration
	DefaultIDE string `json:"defaultIde,omitempty"`

	// IDEs holds the ide configuration
	IDEs map[string]*IDEConfig `json:"ides,omitempty"`

	// Providers holds the provider configuration
	Providers map[string]*ProviderConfig `json:"providers,omitempty"`

	// OriginalProvider is the original default provider
	OriginalProvider string `json:"-"`
}

type IDEConfig struct {
	// Options are additional ide options
	Options map[string]OptionValue `json:"options,omitempty"`
}

type IDE string

const (
	IDENone       IDE = "none"
	IDEVSCode     IDE = "vscode"
	IDEOpenVSCode IDE = "openvscode"
	IDEIntellij   IDE = "intellij"
	IDEGoland     IDE = "goland"
	IDEPyCharm    IDE = "pycharm"
	IDEPhpStorm   IDE = "phpstorm"
	IDECLion      IDE = "clion"
	IDERubyMine   IDE = "rubymine"
	IDERider      IDE = "rider"
	IDEWebStorm   IDE = "webstorm"
)

type ProviderConfig struct {
	// Initialized holds if the provider was initialized correctly.
	Initialized bool `json:"initialized,omitempty"`

	// SingleMachine signals DevPod if a single machine should be used for this provider.
	SingleMachine bool `json:"singleMachine,omitempty"`

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

func (c *Config) Current() *ContextConfig {
	return c.Contexts[c.DefaultContext]
}

func (c *Config) ProviderOptions(provider string) map[string]OptionValue {
	return c.Current().ProviderOptions(provider)
}

func (c *Config) IDEOptions(ide string) map[string]OptionValue {
	return c.Current().IDEOptions(ide)
}

func (c *ContextConfig) IsSingleMachine(provider string) bool {
	if c.Providers == nil || c.Providers[provider] == nil {
		return false
	}
	return c.Providers[provider].SingleMachine
}

func (c *ContextConfig) IDEOptions(ide string) map[string]OptionValue {
	retOptions := map[string]OptionValue{}
	if c.IDEs == nil || c.IDEs[ide] == nil {
		return retOptions
	}

	for k, v := range c.IDEs[ide].Options {
		retOptions[k] = v
	}
	return retOptions
}

func (c *ContextConfig) ProviderOptions(provider string) map[string]OptionValue {
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
	err := json.Unmarshal(out, ret)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %v", err))
	}
	for ctxName, ctx := range ret.Contexts {
		if ctx.Providers == nil {
			ctx.Providers = map[string]*ProviderConfig{}
		}
		if ctx.IDEs == nil {
			ctx.IDEs = map[string]*IDEConfig{}
		}
		ctx.OriginalProvider = config.Contexts[ctxName].OriginalProvider
	}
	ret.Origin = config.Origin
	ret.OriginalContext = config.OriginalContext
	return ret
}

func LoadConfig(contextOverride string, providerOverride string) (*Config, error) {
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
			Contexts: map[string]*ContextConfig{
				context: {
					DefaultProvider: providerOverride,
					Providers:       map[string]*ProviderConfig{},
					IDEs:            map[string]*IDEConfig{},
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
		config.Contexts = map[string]*ContextConfig{}
	}
	if config.Contexts[config.DefaultContext] == nil {
		config.Contexts[config.DefaultContext] = &ContextConfig{}
	}
	if config.Contexts[config.DefaultContext].Providers == nil {
		config.Contexts[config.DefaultContext].Providers = map[string]*ProviderConfig{}
	}
	if config.Contexts[config.DefaultContext].IDEs == nil {
		config.Contexts[config.DefaultContext].IDEs = map[string]*IDEConfig{}
	}
	if providerOverride != "" {
		config.Contexts[config.DefaultContext].OriginalProvider = config.Contexts[config.DefaultContext].DefaultProvider
		config.Contexts[config.DefaultContext].DefaultProvider = providerOverride
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
	if config.Contexts[config.DefaultContext].OriginalProvider != "" {
		config.Contexts[config.DefaultContext].DefaultProvider = config.Contexts[config.DefaultContext].OriginalProvider
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
