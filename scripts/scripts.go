package scripts

import _ "embed"

//go:embed install_docker.sh
var installDocker string

func InstallDocker() (string, error) {
	script, err := WrapScript(installDocker)
	if err != nil {
		return "", err
	}

	return script, nil
}
