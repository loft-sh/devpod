package flags

import (
	flag "github.com/spf13/pflag"
)

type GlobalFlags struct {
	Context  string
	Provider string
	Debug    bool
}

// SetGlobalFlags applies the global flags
func SetGlobalFlags(flags *flag.FlagSet) *GlobalFlags {
	globalFlags := &GlobalFlags{}

	flags.StringVar(&globalFlags.Context, "context", "", "The context to use")
	flags.StringVar(&globalFlags.Provider, "provider", "", "The provider to use. Needs to be configured for the selected context.")
	flags.BoolVar(&globalFlags.Debug, "debug", false, "Prints the stack trace if an error occurs")
	return globalFlags
}
