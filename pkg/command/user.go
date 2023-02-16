package command

import (
	"os/exec"
	"os/user"
)

func AsUser(user string, cmd *exec.Cmd) {
	setUser(user, cmd)
}

func GetHome(userName string) (string, error) {
	u, err := user.Lookup(userName)
	if err != nil {
		return "", err
	}

	return u.HomeDir, nil
}
