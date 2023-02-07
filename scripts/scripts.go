package scripts

import _ "embed"

//go:embed install_docker.sh
var InstallDocker string

//go:embed install_devpod.sh.tpl
var InstallDevPodTemplate string
