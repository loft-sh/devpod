package config

import (
	"encoding/json"
	"math/big"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/loft-sh/log/hash"
)

type ReplaceFunction func(match, variable string, args []string) string

var VariableRegExp = regexp.MustCompile(`\${(.*?)}`)

type SubstitutedConfig struct {
	Config *DevContainerConfig
	Raw    *DevContainerConfig
}

type SubstitutionContext struct {
	DevContainerID           string            `json:"DevContainerID,omitempty"`
	LocalWorkspaceFolder     string            `json:"LocalWorkspaceFolder,omitempty"`
	ContainerWorkspaceFolder string            `json:"ContainerWorkspaceFolder,omitempty"`
	Env                      map[string]string `json:"Env,omitempty"`

	WorkspaceMount string `json:"WorkspaceMount,omitempty"`
}

func Substitute(substitutionCtx *SubstitutionContext, config interface{}, out interface{}) error {
	newVal := map[string]interface{}{}
	err := Convert(config, &newVal)
	if err != nil {
		return err
	}

	// if windows adjust env
	isWindows := runtime.GOOS == "windows"
	if isWindows {
		newEnv := map[string]string{}
		for k, v := range substitutionCtx.Env {
			newEnv[strings.ToLower(k)] = v
		}
		substitutionCtx.Env = newEnv
	}

	if substitutionCtx.ContainerWorkspaceFolder != "" {
		substitutionCtx.ContainerWorkspaceFolder = ResolveString(substitutionCtx.ContainerWorkspaceFolder, func(match, variable string, args []string) string {
			return replaceWithContext(isWindows, substitutionCtx, match, variable, args)
		})
	}
	retVal := substitute0(newVal, func(match, variable string, args []string) string {
		return replaceWithContext(isWindows, substitutionCtx, match, variable, args)
	})

	err = Convert(retVal, out)
	if err != nil {
		return err
	}

	return nil
}

func SubstituteContainerEnv(containerEnv map[string]string, config interface{}, out interface{}) error {
	newVal := map[string]interface{}{}
	err := Convert(config, &newVal)
	if err != nil {
		return err
	}

	// if windows adjust env
	retVal := substitute0(newVal, func(match, variable string, args []string) string {
		return replaceWithContainerEnv(containerEnv, match, variable, args)
	})

	err = Convert(retVal, out)
	if err != nil {
		return err
	}

	return nil
}

func replaceWithContainerEnv(containerEnv map[string]string, match, variable string, args []string) string {
	switch variable {
	case "containerEnv":
		return lookupValue(false, containerEnv, args, match)
	default:
		return match
	}
}

func replaceWithContext(isWindows bool, substitutionCtx *SubstitutionContext, match, variable string, args []string) string {
	switch variable {
	case "devcontainerId":
		if substitutionCtx.DevContainerID != "" {
			return substitutionCtx.DevContainerID
		}
		return match
	case "env":
		fallthrough
	case "localEnv":
		return lookupValue(isWindows, substitutionCtx.Env, args, match)
	case "localWorkspaceFolder":
		if substitutionCtx.LocalWorkspaceFolder != "" {
			return substitutionCtx.LocalWorkspaceFolder
		}
		return match
	case "localWorkspaceFolderBasename":
		if substitutionCtx.LocalWorkspaceFolder != "" {
			return filepath.Base(substitutionCtx.LocalWorkspaceFolder)
		}
		return match
	case "containerWorkspaceFolder":
		if substitutionCtx.ContainerWorkspaceFolder != "" {
			return substitutionCtx.ContainerWorkspaceFolder
		}
		return match
	case "containerWorkspaceFolderBasename":
		if substitutionCtx.ContainerWorkspaceFolder != "" {
			return filepath.Base(substitutionCtx.ContainerWorkspaceFolder)
		}
		return match
	default:
		return match
	}
}

func lookupValue(isWindows bool, env map[string]string, args []string, match string) string {
	if len(args) > 0 {
		envVariableName := args[0]
		if isWindows {
			envVariableName = strings.ToLower(envVariableName)
		}

		foundEnv, ok := env[envVariableName]
		if ok {
			return foundEnv
		}

		if len(args) > 1 {
			defaultValue := args[1]
			return defaultValue
		}

		// For `env` we should do the same as a normal shell does - evaluates missing envs to an empty string #46436
		return ""
	}

	return match
}

func substitute0(val interface{}, replace ReplaceFunction) interface{} {
	switch t := val.(type) {
	case string:
		return ResolveString(t, replace)
	case []interface{}:
		for i, v := range t {
			t[i] = substitute0(v, replace)
		}
		return t
	case map[string]interface{}:
		for k, v := range t {
			t[k] = substitute0(v, replace)
		}
		return t
	default:
		return t
	}
}

func ResolveString(val string, replace ReplaceFunction) string {
	return string(VariableRegExp.ReplaceAllFunc([]byte(val), func(match []byte) []byte {
		variable := string(match[2 : len(match)-1])

		// try to separate variable arguments from variable name
		args := []string{}
		parts := strings.Split(variable, ":")
		if len(parts) > 1 {
			variable = parts[0]
			args = parts[1:]
		}

		return []byte(replace(string(match), variable, args))
	}))
}

func ObjectToList(object map[string]string) []string {
	ret := []string{}
	for k, v := range object {
		ret = append(ret, k+"="+v)
	}

	return ret
}

func ListToObject(list []string) map[string]string {
	ret := map[string]string{}
	for _, l := range list {
		splitted := strings.Split(l, "=")
		if len(splitted) == 1 {
			continue
		}

		ret[splitted[0]] = strings.Join(splitted[1:], "=")
	}

	return ret
}

func GetDevContainerID(labels map[string]string) string {
	labelsBytes, _ := json.Marshal(labels)
	hashedLabels := hash.String(string(labelsBytes))
	bigInt := big.Int{}
	bigInt.SetString(hashedLabels, 16)
	return bigInt.Text(32)
}
