//go:build !windows
// +build !windows

package command

import (
	"bytes"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

func getHome(uid int) (string, error) {
	// try to find homedir
	var stdout bytes.Buffer
	cmd := exec.Command("getent", "passwd", strconv.Itoa(uid))
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	if passwd := strings.TrimSpace(stdout.String()); passwd != "" {
		// username:password:uid:gid:gecos:home:shell
		passwdParts := strings.SplitN(passwd, ":", 7)
		if len(passwdParts) > 5 {
			return passwdParts[5], nil
		}
	}

	return "", nil
}

func setUser(userName string, cmd *exec.Cmd) {
	if userName == "" {
		return
	}

	u, err := user.Lookup(userName)
	if err != nil {
		return
	}

	uid, err := strconv.ParseInt(u.Uid, 10, 32)
	if err != nil {
		return
	}

	gid, err := strconv.ParseInt(u.Gid, 10, 32)
	if err != nil {
		return
	}

	if os.Getuid() == int(uid) {
		return
	}

	groups := []uint32{}
	groupIds, err := u.GroupIds()
	if err == nil {
		for _, group := range groupIds {
			gid, err := strconv.ParseInt(group, 10, 32)
			if err != nil {
				continue
			}

			groups = append(groups, uint32(gid))
		}
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid:    uint32(uid),
		Gid:    uint32(gid),
		Groups: groups,
	}

	// replace HOME
	newEnv := []string{}
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "HOME=") {
			continue
		}

		newEnv = append(newEnv, env)
	}
	home, err := getHome(int(uid))
	if err == nil {
		newEnv = append(newEnv, "HOME="+home)
	}
	cmd.Env = newEnv
}
