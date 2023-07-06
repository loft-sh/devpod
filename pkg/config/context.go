package config

const (
	ContextOptionSSHAgentForwarding         = "SSH_AGENT_FORWARDING"
	ContextOptionSSHInjectDockerCredentials = "SSH_INJECT_DOCKER_CREDENTIALS"
	ContextOptionSSHInjectGitCredentials    = "SSH_INJECT_GIT_CREDENTIALS"
	ContextOptionTelemetry                  = "TELEMETRY"
)

var ContextOptions = []ContextOption{
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
}
