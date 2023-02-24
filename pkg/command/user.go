package command

import (
	"github.com/mitchellh/go-homedir"
	"os/user"
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
