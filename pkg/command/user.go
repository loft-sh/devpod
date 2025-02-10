package command

import (
	"os/user"

	"github.com/loft-sh/devpod/pkg/util"
)

func GetHome(userName string) (string, error) {
	if userName == "" {
		return util.UserHomeDir()
	}

	u, err := user.Lookup(userName)
	if err != nil {
		return "", err
	}

	return u.HomeDir, nil
}
