package ideparse

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"reflect"
	"regexp"
	"strings"
)

type AllowedIDE struct {
	// Name of the IDE
	Name config.IDE `json:"name,omitempty"`
	// Options of the IDE
	Options ide.Options `json:"options,omitempty"`
}

var AllowedIDEs = []AllowedIDE{
	{
		Name:    config.IDENone,
		Options: map[string]ide.Option{},
	},
	{
		Name:    config.IDEVSCode,
		Options: vscode.Options,
	},
	{
		Name:    config.IDEOpenVSCode,
		Options: openvscode.Options,
	},
	{
		Name:    config.IDEGoland,
		Options: jetbrains.GolandOptions,
	},
	{
		Name:    config.IDEPyCharm,
		Options: jetbrains.PyCharmOptions,
	},
	{
		Name:    config.IDEPhpStorm,
		Options: jetbrains.PhpStormOptions,
	},
	{
		Name:    config.IDEIntellij,
		Options: jetbrains.IntellijOptions,
	},
	{
		Name:    config.IDECLion,
		Options: jetbrains.CLionOptions,
	},
	{
		Name:    config.IDERider,
		Options: jetbrains.RiderOptions,
	},
	{
		Name:    config.IDERubyMine,
		Options: jetbrains.RubyMineOptions,
	},
	{
		Name:    config.IDEWebStorm,
		Options: jetbrains.WebStormOptions,
	},
}

func RefreshIDEOptions(devPodConfig *config.Config, workspace *provider.Workspace, ide string, options []string) (*provider.Workspace, error) {
	ide = strings.ToLower(ide)
	if ide == "" {
		if workspace.IDE.Name != "" {
			ide = workspace.IDE.Name
		} else if devPodConfig.Current().DefaultIDE != "" {
			ide = devPodConfig.Current().DefaultIDE
		} else {
			ide = detect()
		}
	}

	// get ide options
	ideOptions, err := GetIDEOptions(ide)
	if err != nil {
		return nil, err
	}

	// get global options and set them as non user
	// provided.
	retValues := devPodConfig.IDEOptions(ide)
	for k, v := range retValues {
		retValues[k] = config.OptionValue{
			Value: v.Value,
		}
	}

	// get existing options
	if ide == workspace.IDE.Name {
		for k, v := range workspace.IDE.Options {
			if !v.UserProvided {
				continue
			}

			retValues[k] = v
		}
	}

	// get user options
	values, err := ParseOptions(options, ideOptions)
	if err != nil {
		return nil, errors.Wrap(err, "parse options")
	}
	for k, v := range values {
		retValues[k] = v
	}

	// check if we need to modify workspace
	if workspace.IDE.Name != ide || !reflect.DeepEqual(workspace.IDE.Options, retValues) {
		workspace = provider.CloneWorkspace(workspace)
		workspace.IDE.Name = ide
		workspace.IDE.Options = retValues
		err = provider.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, errors.Wrap(err, "save workspace")
		}
	}

	return workspace, nil
}

func GetIDEOptions(ide string) (ide.Options, error) {
	var match *AllowedIDE
	for _, m := range AllowedIDEs {
		if string(m.Name) == ide {
			match = &m
			break
		}
	}
	if match == nil {
		allowedIDEArray := []string{}
		for _, a := range AllowedIDEs {
			allowedIDEArray = append(allowedIDEArray, string(a.Name))
		}

		return nil, fmt.Errorf("unrecognized ide '%s', please use one of: %v", ide, allowedIDEArray)
	}

	return match.Options, nil
}

func ParseOptions(options []string, ideOptions ide.Options) (map[string]config.OptionValue, error) {
	if ideOptions == nil {
		ideOptions = ide.Options{}
	}

	allowedOptions := []string{}
	for optionName := range ideOptions {
		allowedOptions = append(allowedOptions, optionName)
	}

	retMap := map[string]config.OptionValue{}
	for _, option := range options {
		splitted := strings.Split(option, "=")
		if len(splitted) == 1 {
			return nil, fmt.Errorf("invalid option '%s', expected format KEY=VALUE", option)
		}

		key := strings.ToUpper(strings.TrimSpace(splitted[0]))
		value := strings.Join(splitted[1:], "=")
		ideOption, ok := ideOptions[key]
		if !ok {
			return nil, fmt.Errorf("invalid option '%s', allowed options are: %v", key, allowedOptions)
		}

		if ideOption.ValidationPattern != "" {
			matcher, err := regexp.Compile(ideOption.ValidationPattern)
			if err != nil {
				return nil, err
			}

			if !matcher.MatchString(value) {
				if ideOption.ValidationMessage != "" {
					return nil, fmt.Errorf(ideOption.ValidationMessage)
				}

				return nil, fmt.Errorf("invalid value '%s' for option '%s', has to match the following regEx: %s", value, key, ideOption.ValidationPattern)
			}
		}

		if len(ideOption.Enum) > 0 {
			found := false
			for _, e := range ideOption.Enum {
				if value == e {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("invalid value '%s' for option '%s', has to match one of the following values: %v", value, key, ideOption.Enum)
			}
		}

		retMap[key] = config.OptionValue{
			Value:        value,
			UserProvided: true,
		}
	}

	return retMap, nil
}

func detect() string {
	if command.Exists("code") {
		return string(config.IDEVSCode)
	}

	return string(config.IDEOpenVSCode)
}
