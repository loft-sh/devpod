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
	IDEDataSpell       IDE = "dataspell"
	IDEFleet           IDE = "fleet"
	IDEJupyterNotebook IDE = "jupyternotebook"
	IDECursor          IDE = "cursor"
	IDEPositron        IDE = "positron"
	IDECodium          IDE = "codium"
	IDEZed             IDE = "zed"
	IDERStudio         IDE = "rstudio"
	IDEWindsurf        IDE = "windsurf"
)

type IDEGroup string

const (
	IDEGroupPrimary   IDEGroup = "Primary"
	IDEGroupJetBrains IDEGroup = "JetBrains"
	IDEGroupOther     IDEGroup = "Other"
)
