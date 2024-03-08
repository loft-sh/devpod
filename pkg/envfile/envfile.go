package envfile

import (
	"encoding/json"
	"os"

	"github.com/loft-sh/log"
)

var location = "/etc/envfile.json"

type EnvFile struct {
	// Env holds the environment variables to set
	Env map[string]string `json:"env,omitempty"`
}

func Apply(log log.Logger) {
	out, err := os.ReadFile(location)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Debugf("Error reading envfile: %v", err)
		}

		return
	}

	envFile := &EnvFile{}
	err = json.Unmarshal(out, envFile)
	if err != nil {
		log.Debugf("Error parsing envfile: %v", err)
		return
	}

	for k, v := range envFile.Env {
		_ = os.Setenv(k, v)
	}
}

func MergeAndApply(env map[string]string, log log.Logger) {
	if len(env) == 0 {
		return
	}

	envFile := &EnvFile{}
	out, err := os.ReadFile(location)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Debugf("Error reading envfile: %v", err)
			return
		}
	} else {
		err = json.Unmarshal(out, envFile)
		if err != nil {
			log.Debugf("Error parsing envfile: %v", err)
			return
		}
	}

	if envFile.Env == nil {
		envFile.Env = map[string]string{}
	}
	for k, v := range env {
		envFile.Env[k] = v
	}

	out, err = json.Marshal(envFile)
	if err != nil {
		log.Debugf("Error marshalling envfile: %v", err)
		return
	}

	err = os.WriteFile(location, out, 0600)
	if err != nil {
		log.Debugf("Error writing envfile: %v", err)
		return
	}

	// apply
	for k, v := range envFile.Env {
		_ = os.Setenv(k, v)
	}
}
