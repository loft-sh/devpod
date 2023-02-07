package flags

import (
	flag "github.com/spf13/pflag"
)

type GlobalFlags struct {
	Context  string
	Provider string
	Debug    bool
	Silent   bool
}

// SetGlobalFlags applies the global flags
func SetGlobalFlags(flags *flag.FlagSet) *GlobalFlags {
	globalFlags := &GlobalFlags{}

	flags.StringVar(&globalFlags.Context, "context", "", "The context to use")
	flags.StringVar(&globalFlags.Provider, "provider", "", "The provider to use. Needs to be configured for the selected context.")
	flags.BoolVar(&globalFlags.Debug, "debug", false, "Prints the stack trace if an error occurs")
	flags.BoolVar(&globalFlags.Silent, "silent", false, "Run in silent mode and prevents any devpod log output except panics & fatals")
	return globalFlags
}
