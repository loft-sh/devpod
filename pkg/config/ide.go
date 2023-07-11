package config

type IDE string

const (
	IDENone            IDE = "none"
	IDEVSCode          IDE = "vscode"
	IDEOpenVSCode      IDE = "openvscode"
	IDEIntellij        IDE = "intellij"
	IDEGoland          IDE = "goland"
	IDEPyCharm         IDE = "pycharm"
	IDEPhpStorm        IDE = "phpstorm"
	IDECLion           IDE = "clion"
	IDERubyMine        IDE = "rubymine"
	IDERider           IDE = "rider"
	IDEWebStorm        IDE = "webstorm"
	IDEFleet           IDE = "fleet"
	IDEJupyterNotebook IDE = "jupyternotebook"
)
