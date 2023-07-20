//go:build windows
// +build windows

package setup

import (
	"fmt"
	"io/fs"
)

func GetUserInfo(info fs.FileInfo) (string, string, error) {
	return "", "", fmt.Errorf("userinfo is currently not supported on windows")
}
