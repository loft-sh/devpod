package config

const (
	ContextOptionSSHAddPrivateKeys          = "SSH_ADD_PRIVATE_KEYS"
	ContextOptionGPGAgentForwarding         = "GPG_AGENT_FORWARDING"
	ContextOptionGitSSHSignatureForwarding  = "GIT_SSH_SIGNATURE_FORWARDING"
	ContextOptionSSHInjectDockerCredentials = "SSH_INJECT_DOCKER_CREDENTIALS"
	ContextOptionSSHInjectGitCredentials    = "SSH_INJECT_GIT_CREDENTIALS"
	ContextOptionExitAfterTimeout           = "EXIT_AFTER_TIMEOUT"
	ContextOptionTelemetry                  = "TELEMETRY"
	ContextOptionAgentURL                   = "AGENT_URL"
	ContextOptionDotfilesURL                = "DOTFILES_URL"
	ContextOptionDotfilesScript             = "DOTFILES_SCRIPT"
	ContextOptionSSHAgentForwarding         = "SSH_AGENT_FORWARDING"
	ContextOptionSSHConfigPath              = "SSH_CONFIG_PATH"
	ContextOptionAgentInjectTimeout         = "AGENT_INJECT_TIMEOUT"
	ContextOptionRegistryCache              = "REGISTRY_CACHE"
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
		Name:        ContextOptionGPGAgentForwarding,
		Description: "Specifies if DevPod should do gpg-agent forwarding by default for ssh",
		Default:     "false",
		Enum:        []string{"true", "false"},
	},
	{
		Name:        ContextOptionGitSSHSignatureForwarding,
		Description: "Specifies if DevPod should automatically detect ssh signature git setting and inject ssh signature helper",
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
		Name:        ContextOptionSSHAgentForwarding,
		Description: "Specifies if DevPod should do agent forwarding by default into the workspace",
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
	{
		Name:        ContextOptionSSHConfigPath,
		Description: "Specifies the path where the ssh config should be written to",
	},
	{
		Name:        ContextOptionAgentInjectTimeout,
		Description: "Specifies the timeout to inject the agent",
		Default:     "20",
	},
	{
		Name:        ContextOptionRegistryCache,
		Description: "Specifies the registry to use as a build cache",
		Default:     "",
	},
}
