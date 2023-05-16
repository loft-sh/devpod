package telemetry

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"

	"github.com/denisbrodbeck/machineid"
	"github.com/loft-sh/devpod/pkg/telemetry/types"
	"github.com/loft-sh/devpod/pkg/version"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	// hashingKey is a random string used for hashing the UID.
	// It shouldn't be changed after the release.
	hashingKey = "2f1uR7n8ryzFEaAm87Ec"
	UIEnvVar   = "DEVPOD_UI"
)

func (d *DefaultCollector) getInstanceProperties(command *cobra.Command, executionID string, ts int64) types.InstanceProperties {
	p := types.InstanceProperties{
		Timestamp:   ts,
		ExecutionID: executionID,
		UID:         getUID(),
		Arch:        runtime.GOARCH,
		OS:          runtime.GOOS,
		Version:     getVersion(),
		Flags:       getFlags(command),
		UI:          isUIEvent(),
	}

	return p
}

// Gets machine ID and encodes it together with users $HOME path and extra key to protect privacy.
// Returns a hex-encoded string.
func getUID() string {
	id, err := machineid.ID()
	if err != nil {
		id = "error"
	}
	// get $HOME to distinguish two users on the same machine
	// will be hashed later together with the ID
	home, err := homedir.Dir()
	if err != nil {
		home = "error"
	}
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(hashingKey))
	mac.Write([]byte(home))
	return fmt.Sprintf("%x", mac.Sum(nil))
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
