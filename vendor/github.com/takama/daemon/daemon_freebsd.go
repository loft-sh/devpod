// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

package daemon

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// systemVRecord - standard record (struct) for linux systemV version of daemon package
type bsdRecord struct {
	name         string
	description  string
	kind         Kind
	dependencies []string
}

// Standard service path for systemV daemons
func (bsd *bsdRecord) servicePath() string {
	return "/usr/local/etc/rc.d/" + bsd.name
}

// Is a service installed
func (bsd *bsdRecord) isInstalled() bool {

	if _, err := os.Stat(bsd.servicePath()); err == nil {
		return true
	}

	return false
}

// Is a service is enabled
func (bsd *bsdRecord) isEnabled() (bool, error) {
	rcConf, err := os.Open("/etc/rc.conf")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return false, err
	}
	defer rcConf.Close()
	rcData, _ := ioutil.ReadAll(rcConf)
	r, _ := regexp.Compile(`.*` + bsd.name + `_enable="YES".*`)
	v := string(r.Find(rcData))
	var chrFound, sharpFound bool
	for _, c := range v {
		if c == '#' && !chrFound {
			sharpFound = true
			break
		} else if !sharpFound && c != ' ' {
			chrFound = true
			break
		}
	}
	return chrFound, nil
}

func (bsd *bsdRecord) getCmd(cmd string) string {
	if ok, err := bsd.isEnabled(); !ok || err != nil {
		fmt.Println("Service is not enabled, using one" + cmd + " instead")
		cmd = "one" + cmd
	}
	return cmd
}

// Get the daemon properly
func newDaemon(name, description string, kind Kind, dependencies []string) (Daemon, error) {
	return &bsdRecord{name, description, kind, dependencies}, nil
}

func execPath() (name string, err error) {
	name = os.Args[0]
	if name[0] == '.' {
		name, err = filepath.Abs(name)
		if err == nil {
			name = filepath.Clean(name)
		}
	} else {
		name, err = exec.LookPath(filepath.Clean(name))
	}
	return name, err
}

// Check service is running
func (bsd *bsdRecord) checkRunning() (string, bool) {
	output, err := exec.Command("service", bsd.name, bsd.getCmd("status")).Output()
	if err == nil {
		if matched, err := regexp.MatchString(bsd.name, string(output)); err == nil && matched {
			reg := regexp.MustCompile("pid  ([0-9]+)")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				return "Service (pid  " + data[1] + ") is running...", true
			}
			return "Service is running...", true
		}
	}

	return "Service is stopped", false
}

// Install the service
func (bsd *bsdRecord) Install(args ...string) (string, error) {
	installAction := "Install " + bsd.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return installAction + failed, err
	}

	srvPath := bsd.servicePath()

	if bsd.isInstalled() {
		return installAction + failed, ErrAlreadyInstalled
	}

	file, err := os.Create(srvPath)
	if err != nil {
		return installAction + failed, err
	}
	defer file.Close()

	execPatch, err := executablePath(bsd.name)
	if err != nil {
		return installAction + failed, err
	}

	templ, err := template.New("bsdConfig").Parse(bsdConfig)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Path, Args string
		}{bsd.name, bsd.description, execPatch, strings.Join(args, " ")},
	); err != nil {
		return installAction + failed, err
	}

	if err := os.Chmod(srvPath, 0755); err != nil {
		return installAction + failed, err
	}

	return installAction + success, nil
}

// Remove the service
func (bsd *bsdRecord) Remove() (string, error) {
	removeAction := "Removing " + bsd.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return removeAction + failed, err
	}

	if !bsd.isInstalled() {
		return removeAction + failed, ErrNotInstalled
	}

	if err := os.Remove(bsd.servicePath()); err != nil {
		return removeAction + failed, err
	}

	return removeAction + success, nil
}

// Start the service
func (bsd *bsdRecord) Start() (string, error) {
	startAction := "Starting " + bsd.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return startAction + failed, err
	}

	if !bsd.isInstalled() {
		return startAction + failed, ErrNotInstalled
	}

	if _, ok := bsd.checkRunning(); ok {
		return startAction + failed, ErrAlreadyRunning
	}

	if err := exec.Command("service", bsd.name, bsd.getCmd("start")).Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (bsd *bsdRecord) Stop() (string, error) {
	stopAction := "Stopping " + bsd.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return stopAction + failed, err
	}

	if !bsd.isInstalled() {
		return stopAction + failed, ErrNotInstalled
	}

	if _, ok := bsd.checkRunning(); !ok {
		return stopAction + failed, ErrAlreadyStopped
	}

	if err := exec.Command("service", bsd.name, bsd.getCmd("stop")).Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (bsd *bsdRecord) Status() (string, error) {

	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !bsd.isInstalled() {
		return statNotInstalled, ErrNotInstalled
	}

	statusAction, _ := bsd.checkRunning()

	return statusAction, nil
}

// Run - Run service
func (bsd *bsdRecord) Run(e Executable) (string, error) {
	runAction := "Running " + bsd.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

// GetTemplate - gets service config template
func (linux *bsdRecord) GetTemplate() string {
	return bsdConfig
}

// SetTemplate - sets service config template
func (linux *bsdRecord) SetTemplate(tplStr string) error {
	bsdConfig = tplStr
	return nil
}

var bsdConfig = `#!/bin/sh
#
# PROVIDE: {{.Name}}
# REQUIRE: networking syslog
# KEYWORD:

# Add the following lines to /etc/rc.conf to enable the {{.Name}}:
#
# {{.Name}}_enable="YES"
#


. /etc/rc.subr

name="{{.Name}}"
rcvar="{{.Name}}_enable"
command="{{.Path}}"
pidfile="/var/run/$name.pid"

start_cmd="/usr/sbin/daemon -p $pidfile -f $command {{.Args}}"
load_rc_config $name
run_rc_command "$1"
`
