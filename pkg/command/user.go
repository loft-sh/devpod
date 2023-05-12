package command

import (
	"os/user"

	"github.com/mitchellh/go-homedir"
)

func GetHome(userName string) (string, error) {
	if userName == "" {
		return homedir.Dir()
	}

	u, err := user.Lookup(userName)
	if err != nil {
		return "", err
	}

	return u.HomeDir, nil
}
