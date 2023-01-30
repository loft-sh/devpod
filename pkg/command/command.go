package command

import "os/exec"

func Exists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
