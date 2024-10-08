package flags

import (
	flag "github.com/spf13/pflag"
)

type GlobalFlags struct {
	Context   string
	Provider  string
	LogOutput string

	Debug  bool
	Silent bool

	AgentDir string

	DevPodHome string

	UID string
}

// SetGlobalFlags applies the global flags
func SetGlobalFlags(flags *flag.FlagSet) *GlobalFlags {
	globalFlags := &GlobalFlags{}

	flags.StringVar(&globalFlags.DevPodHome, "devpod-home", "", "If defined will override the default devpod home. You can also use DEVPOD_HOME to set this")
	flags.StringVar(&globalFlags.LogOutput, "log-output", "plain", "The log format to use. Can be either plain, raw or json")
	flags.StringVar(&globalFlags.Context, "context", "", "The context to use")
	flags.StringVar(&globalFlags.Provider, "provider", "", "The provider to use. Needs to be configured for the selected context.")
	flags.BoolVar(&globalFlags.Debug, "debug", false, "Prints the stack trace if an error occurs")
	flags.BoolVar(&globalFlags.Silent, "silent", false, "Run in silent mode and prevents any devpod log output except panics & fatals")

	_ = flags.MarkHidden("git-username")
	_ = flags.MarkHidden("git-token")
	flags.StringVar(&globalFlags.UID, "uid", "", "Set UID for workspace")
	_ = flags.MarkHidden("uid")

	flags.StringVar(&globalFlags.AgentDir, "agent-dir", "", "The data folder where agent data is stored.")
	_ = flags.MarkHidden("agent-dir")
	return globalFlags
}
