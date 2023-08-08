package config

const (
	ContextOptionSSHAddPrivateKeys          = "SSH_ADD_PRIVATE_KEYS"
	ContextOptionSSHAgentForwarding         = "SSH_AGENT_FORWARDING"
	ContextOptionSSHInjectDockerCredentials = "SSH_INJECT_DOCKER_CREDENTIALS"
	ContextOptionSSHInjectGitCredentials    = "SSH_INJECT_GIT_CREDENTIALS"
	ContextOptionExitAfterTimeout           = "EXIT_AFTER_TIMEOUT"
	ContextOptionTelemetry                  = "TELEMETRY"
	ContextOptionAgentURL                   = "AGENT_URL"
	ContextOptionDotfilesURL                = "DOTFILES_URL"
	ContextOptionDotfilesScript             = "DOTFILES_SCRIPT"
)

var ContextOptions = []ContextOption{
	{
		Name:        ContextOptionSSHAddPrivateKeys,
		Description: "Specifies if DevPod should automatically add ssh-keys to the ssh-agent",
		Default:     "true",
		Enum:        []string{"true", "false"},
	},
	{
		Name:        ContextOptionExitAfterTimeout,
		Description: "Specifies if DevPod should exit the process after the browser has been idle for a minute",
		Default:     "true",
		Enum:        []string{"true", "false"},
	},
	{
		Name:        ContextOptionSSHAgentForwarding,
		Description: "Specifies if DevPod should do agent forwarding by default for ssh",
		Default:     "true",
		Enum:        []string{"true", "false"},
	},
	{
		Name:        ContextOptionSSHInjectDockerCredentials,
		Description: "Specifies if DevPod should inject docker credentials into the workspace",
		Default:     "true",
		Enum:        []string{"true", "false"},
	},
	{
		Name:        ContextOptionSSHInjectGitCredentials,
		Description: "Specifies if DevPod should inject git credentials into the workspace",
		Default:     "true",
		Enum:        []string{"true", "false"},
	},
	{
		Name:        ContextOptionTelemetry,
		Description: "Specifies if DevPod should send telemetry information",
		Default:     "true",
		Enum:        []string{"true", "false"},
	},
	{
		Name:        ContextOptionAgentURL,
		Description: "Specifies the agent url to use for DevPod",
	},
	{
		Name:        ContextOptionDotfilesURL,
		Description: "Specifies the dotfiles repo url to use for DevPod",
	},
	{
		Name:        ContextOptionDotfilesScript,
		Description: "Specifies the script to run after cloning dotfiles repo to install them",
	},
}
