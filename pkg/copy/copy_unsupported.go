//go:build windows

package copy

import (
	"os"

	"golang.org/x/sys/windows"
)

func IsUID(info os.FileInfo, uid uint32) bool {
	return true
}

// Lchown implementation for windows
// inspired by: https://gist.github.com/micahyoung/4163bbe0195a18850706e7f34cef3c01
func Lchown(info os.FileInfo, sourcePath, destPath string) error {

	secInfo, err := windows.GetNamedSecurityInfo(
		sourcePath,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION)

	if err != nil {
		return err
	}

	owner, _, err := secInfo.Owner()
	if err != nil {
		return err
	}

	// write a owner SIDs to the file's security descriptor
	return windows.SetNamedSecurityInfo(
		destPath,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION,
		owner,
		nil,
		nil,
		nil,
	)
}
