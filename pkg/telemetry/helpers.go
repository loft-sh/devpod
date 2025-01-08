package telemetry

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"github.com/denisbrodbeck/machineid"
	"github.com/loft-sh/devpod/pkg/util"
)

// GetMachineID retrieves machine ID and encodes it together with users $HOME path and
// extra key to protect privacy. Returns a hex-encoded string.
func GetMachineID() string {
	id, err := machineid.ID()
	if err != nil {
		id = "error"
	}

	// get $HOME to distinguish two users on the same machine
	// will be hashed later together with the ID
	home, err := util.UserHomeDir()
	if err != nil {
		home = "error"
	}

	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(home))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
