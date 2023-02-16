package command

import (
	"os/user"
)

func GetHome(userName string) (string, error) {
	u, err := user.Lookup(userName)
	if err != nil {
		return "", err
	}

	return u.HomeDir, nil
}
