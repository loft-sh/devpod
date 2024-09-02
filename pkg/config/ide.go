package config

type IDE string

const (
	IDENone            IDE = "none"
	IDEVSCode          IDE = "vscode"
	IDEVSCodeInsiders  IDE = "vscode-insiders"
	IDEOpenVSCode      IDE = "openvscode"
	IDEIntellij        IDE = "intellij"
	IDEGoland          IDE = "goland"
	IDERustRover       IDE = "rustrover"
	IDEPyCharm         IDE = "pycharm"
	IDEPhpStorm        IDE = "phpstorm"
	IDECLion           IDE = "clion"
	IDERubyMine        IDE = "rubymine"
	IDERider           IDE = "rider"
	IDEWebStorm        IDE = "webstorm"
	IDEFleet           IDE = "fleet"
	IDEJupyterNotebook IDE = "jupyternotebook"
	IDECursor          IDE = "cursor"
)
