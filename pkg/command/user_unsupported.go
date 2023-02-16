//go:build windows
// +build windows

package command

import (
	"fmt"
	"os/exec"
)

func getHome(uid int) (string, error) {
	return "", fmt.Errorf("unsupported")
}

func setUser(userName string, cmd *exec.Cmd) {

}
