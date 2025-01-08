package encoding

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/log"
)

const (
	// hashingKey is a random string used for hashing the UID.
	// It shouldn't be changed after the release.
	hashingKey = "2f1uR7n8ryzFEaAm87Ec"
)

const (
	WorkspaceUIDLength = 16
	MachineUIDLength   = 40
)

func CreateNewUID(context, id string) string {
	// this returns always a UID with length 16
	uid := strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")
	args := []string{}
	if context != "" {
		args = append(args, context)
	}
	if id != "" {
		args = append(args, id)
	}
	args = append(args, uid)
	return SafeConcatNameMax(args, WorkspaceUIDLength)
}

func CreateNewUIDShort(id string) string {
	// this returns always a UID with length 16
	uid := strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")[0:5]
	return SafeConcatNameMax([]string{id, uid}, WorkspaceUIDLength)
}

func IsLegacyUID(uid string) bool {
	return len(uid) != WorkspaceUIDLength && len(uid) != MachineUIDLength
}

func SafeConcatNameMax(name []string, max int) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > max {
		digest := sha256.Sum256([]byte(fullPath))
		digestEncoded := hex.EncodeToString(digest[0:])
		trimmedPath := fullPath[0 : max-6]
		if strings.HasSuffix(trimmedPath, "-") {
			trimmedPath += digestEncoded[0:6]
		} else {
			trimmedPath += "-" + digestEncoded[0:5]
		}

		return trimmedPath
	}
	return fullPath
}

func GetMachineUIDShort(log log.Logger) string {
	return GetMachineUID(log)[0:5]
}

// Gets machine ID and encodes it together with users $HOME path and extra key to protect privacy.
// Returns a hex-encoded string.
func GetMachineUID(log log.Logger) string {
	id, err := machineid.ID()
	if err != nil {
		id = "error"
		if log != nil {
			log.Debugf("Error retrieving machine uid: %v", err)
		}
	}
	// get $HOME to distinguish two users on the same machine
	// will be hashed later together with the ID
	home, err := util.UserHomeDir()
	if err != nil {
		home = "error"
		if log != nil {
			log.Debugf("Error retrieving machine home: %v", err)
		}
	}
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(hashingKey))
	mac.Write([]byte(home))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
