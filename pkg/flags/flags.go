package flags

import (
	flag "github.com/spf13/pflag"
)

type GitCredentialsFlags struct {
	GitUsername string `json:"gitUsername,omitempty"`
	GitToken    string `json:"gitToken,omitempty"`
}

func SetGitCredentialsFlags(flags *flag.FlagSet, o *GitCredentialsFlags) {
	flags.StringVar(&o.GitUsername, "git-username", "", "The username to use for git operations")
	flags.StringVar(&o.GitToken, "git-token", "", "The token to use for git operations")
	_ = flags.MarkHidden("git-username")
	_ = flags.MarkHidden("git-token")
}
