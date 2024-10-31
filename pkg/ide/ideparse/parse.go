package ideparse

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/ide/fleet"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/jupyter"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
)

type AllowedIDE struct {
	// Name of the IDE
	Name config.IDE `json:"name,omitempty"`
	// DisplayName is the name to show to the user
	DisplayName string `json:"displayName,omitempty"`
	// Options of the IDE
	Options ide.Options `json:"options,omitempty"`
	// Icon holds an image URL that will be displayed
	Icon string `json:"icon,omitempty"`
	// IconDark holds an image URL that will be displayed in dark mode
	IconDark string `json:"iconDark,omitempty"`
	// Experimental indicates that this IDE is experimental
	Experimental bool `json:"experimental,omitempty"`
}

var AllowedIDEs = []AllowedIDE{
	{
		Name:        config.IDENone,
		DisplayName: "None",
		Options:     map[string]ide.Option{},
		Icon:        "https://devpod.sh/assets/none.svg",
		IconDark:    "https://devpod.sh/assets/none_dark.svg",
	},
	{
		Name:        config.IDEVSCode,
		DisplayName: "VSCode",
		Options:     vscode.Options,
		Icon:        "https://devpod.sh/assets/vscode.svg",
	},
	{
		Name:        config.IDEOpenVSCode,
		DisplayName: "VSCode Browser",
		Options:     openvscode.Options,
		Icon:        "https://devpod.sh/assets/vscodebrowser.svg",
	},
	{
		Name:        config.IDEGoland,
		DisplayName: "Goland",
		Options:     jetbrains.GolandOptions,
		Icon:        "https://devpod.sh/assets/goland.svg",
	},
	{
		Name:        config.IDERustRover,
		DisplayName: "RustRover",
		Options:     jetbrains.RustRoverOptions,
		Icon:        "https://devpod.sh/assets/rustrover.svg",
	},
	{
		Name:        config.IDEPyCharm,
		DisplayName: "PyCharm",
		Options:     jetbrains.PyCharmOptions,
		Icon:        "https://devpod.sh/assets/pycharm.svg",
	},
	{
		Name:        config.IDEPhpStorm,
		DisplayName: "PhpStorm",
		Options:     jetbrains.PhpStormOptions,
		Icon:        "https://devpod.sh/assets/phpstorm.svg",
	},
	{
		Name:        config.IDEIntellij,
		DisplayName: "Intellij",
		Options:     jetbrains.IntellijOptions,
		Icon:        "https://devpod.sh/assets/intellij.svg",
	},
	{
		Name:        config.IDECLion,
		DisplayName: "CLion",
		Options:     jetbrains.CLionOptions,
		Icon:        "https://devpod.sh/assets/clion.svg",
	},
	{
		Name:        config.IDERider,
		DisplayName: "Rider",
		Options:     jetbrains.RiderOptions,
		Icon:        "https://devpod.sh/assets/rider.svg",
	},
	{
		Name:        config.IDERubyMine,
		DisplayName: "RubyMine",
		Options:     jetbrains.RubyMineOptions,
		Icon:        "https://devpod.sh/assets/rubymine.svg",
	},
	{
		Name:        config.IDEWebStorm,
		DisplayName: "WebStorm",
		Options:     jetbrains.WebStormOptions,
		Icon:        "https://devpod.sh/assets/webstorm.svg",
	},
	{
		Name:         config.IDEFleet,
		DisplayName:  "Fleet",
		Options:      fleet.Options,
		Icon:         "https://devpod.sh/assets/fleet.svg",
		Experimental: true,
	},
	{
		Name:         config.IDEJupyterNotebook,
		DisplayName:  "Jupyter Notebook",
		Options:      jupyter.Options,
		Icon:         "https://devpod.sh/assets/jupyter.svg",
		IconDark:     "https://devpod.sh/assets/jupyter_dark.svg",
		Experimental: true,
	},
	{
		Name:         config.IDEJupyterDesktop,
		DisplayName:  "Jupyter Desktop",
		Options:      jupyter.Options,
		Icon:         "https://devpod.sh/assets/jupyter.svg",
		IconDark:     "https://devpod.sh/assets/jupyter_dark.svg",
		Experimental: true,
	},
	{
		Name:         config.IDEVSCodeInsiders,
		DisplayName:  "VSCode Insiders",
		Options:      vscode.Options,
		Icon:         "https://devpod.sh/assets/vscode_insiders.svg",
		Experimental: true,
	},
	{
		Name:         config.IDECursor,
		DisplayName:  "Cursor",
		Options:      vscode.Options,
		Icon:         "https://devpod.sh/assets/cursor.svg",
		Experimental: true,
	},
	{
		Name:         config.IDEPositron,
		DisplayName:  "Positron",
		Options:      vscode.Options,
		Icon:         "https://devpod.sh/assets/positron.svg",
		Experimental: true,
	},
	{
		Name:         config.IDEMarimo,
		DisplayName:  "Marimo",
		Options:      vscode.Options,
		Icon:         "https://devpod.sh/assets/marimo.svg",
		Experimental: true,
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
		m := m
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
					return nil, fmt.Errorf("%s", ideOption.ValidationMessage)
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
