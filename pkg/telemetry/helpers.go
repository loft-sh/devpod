package telemetry

import (
	"os"
	"runtime"

	"github.com/loft-sh/devpod/pkg/encoding"
	"github.com/loft-sh/devpod/pkg/telemetry/types"
	"github.com/loft-sh/devpod/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	UIEnvVar = "DEVPOD_UI"
)

func (d *DefaultCollector) getInstanceProperties(command *cobra.Command, executionID string, ts int64) types.InstanceProperties {
	p := types.InstanceProperties{
		Timestamp:   ts,
		ExecutionID: executionID,
		UID:         encoding.GetMachineUID(nil),
		Arch:        runtime.GOARCH,
		OS:          runtime.GOOS,
		Version:     getVersion(),
		Flags:       getFlags(command),
		UI:          isUIEvent(),
	}

	return p
}

func getVersion() types.Version {
	return types.Version{
		Major:      version.GetMajorVersion(),
		Minor:      version.GetMinorVersion(),
		Patch:      version.GetPatchVersion(),
		PreRelease: version.GetPrerelease(),
		Build:      version.GetBuild(),
	}
}

func getFlags(command *cobra.Command) types.Flags {
	if command == nil {
		return types.Flags{}
	}

	setFlags := []string{}
	command.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			setFlags = append(setFlags, f.Name)
		}
	})

	return types.Flags{SetFlags: setFlags}
}

func shouldSkipCommand(cmd string) bool {
	if isUIEvent() {
		for _, exception := range UIEventsExceptions {
			if cmd == exception {
				return true
			}
		}
	}
	return false
}

func isUIEvent() bool {
	return os.Getenv(UIEnvVar) == "true"
}
