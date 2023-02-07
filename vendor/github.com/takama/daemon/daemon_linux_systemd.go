// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

package daemon

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

// systemDRecord - standard record (struct) for linux systemD version of daemon package
type systemDRecord struct {
	name         string
	description  string
	kind         Kind
	dependencies []string
}

// Standard service path for systemD daemons
func (linux *systemDRecord) servicePath() string {
	return "/etc/systemd/system/" + linux.name + ".service"
}

// Is a service installed
func (linux *systemDRecord) isInstalled() bool {

	if _, err := os.Stat(linux.servicePath()); err == nil {
		return true
	}

	return false
}

// Check service is running
func (linux *systemDRecord) checkRunning() (string, bool) {
	output, err := exec.Command("systemctl", "status", linux.name+".service").Output()
	if err == nil {
		if matched, err := regexp.MatchString("Active: active", string(output)); err == nil && matched {
			reg := regexp.MustCompile("Main PID: ([0-9]+)")
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
func (linux *systemDRecord) Install(args ...string) (string, error) {
	installAction := "Install " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return installAction + failed, err
	}

	srvPath := linux.servicePath()

	if linux.isInstalled() {
		return installAction + failed, ErrAlreadyInstalled
	}

	file, err := os.Create(srvPath)
	if err != nil {
		return installAction + failed, err
	}
	defer file.Close()

	execPatch, err := executablePath(linux.name)
	if err != nil {
		return installAction + failed, err
	}

	templ, err := template.New("systemDConfig").Parse(systemDConfig)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Dependencies, Path, Args string
		}{
			linux.name,
			linux.description,
			strings.Join(linux.dependencies, " "),
			execPatch,
			strings.Join(args, " "),
		},
	); err != nil {
		return installAction + failed, err
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return installAction + failed, err
	}

	if err := exec.Command("systemctl", "enable", linux.name+".service").Run(); err != nil {
		return installAction + failed, err
	}

	return installAction + success, nil
}

// Remove the service
func (linux *systemDRecord) Remove() (string, error) {
	removeAction := "Removing " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return removeAction + failed, err
	}

	if !linux.isInstalled() {
		return removeAction + failed, ErrNotInstalled
	}

	if err := exec.Command("systemctl", "disable", linux.name+".service").Run(); err != nil {
		return removeAction + failed, err
	}

	if err := os.Remove(linux.servicePath()); err != nil {
		return removeAction + failed, err
	}

	return removeAction + success, nil
}

// Start the service
func (linux *systemDRecord) Start() (string, error) {
	startAction := "Starting " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return startAction + failed, err
	}

	if !linux.isInstalled() {
		return startAction + failed, ErrNotInstalled
	}

	if _, ok := linux.checkRunning(); ok {
		return startAction + failed, ErrAlreadyRunning
	}

	if err := exec.Command("systemctl", "start", linux.name+".service").Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (linux *systemDRecord) Stop() (string, error) {
	stopAction := "Stopping " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return stopAction + failed, err
	}

	if !linux.isInstalled() {
		return stopAction + failed, ErrNotInstalled
	}

	if _, ok := linux.checkRunning(); !ok {
		return stopAction + failed, ErrAlreadyStopped
	}

	if err := exec.Command("systemctl", "stop", linux.name+".service").Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (linux *systemDRecord) Status() (string, error) {

	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !linux.isInstalled() {
		return statNotInstalled, ErrNotInstalled
	}

	statusAction, _ := linux.checkRunning()

	return statusAction, nil
}

// Run - Run service
func (linux *systemDRecord) Run(e Executable) (string, error) {
	runAction := "Running " + linux.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

// GetTemplate - gets service config template
func (linux *systemDRecord) GetTemplate() string {
	return systemDConfig
}

// SetTemplate - sets service config template
func (linux *systemDRecord) SetTemplate(tplStr string) error {
	systemDConfig = tplStr
	return nil
}

var systemDConfig = `[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
PIDFile=/var/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /var/run/{{.Name}}.pid
ExecStart={{.Path}} {{.Args}}
Restart=on-failure

[Install]
WantedBy=multi-user.target
`
