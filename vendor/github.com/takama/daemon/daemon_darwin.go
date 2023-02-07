// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

// Package daemon darwin (mac os x) version
package daemon

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"text/template"
)

// darwinRecord - standard record (struct) for darwin version of daemon package
type darwinRecord struct {
	name         string
	description  string
	kind         Kind
	dependencies []string
}

func newDaemon(name, description string, kind Kind, dependencies []string) (Daemon, error) {

	return &darwinRecord{name, description, kind, dependencies}, nil
}

// Standard service path for system daemons
func (darwin *darwinRecord) servicePath() string {
	var path string

	switch darwin.kind {
	case UserAgent:
		usr, _ := user.Current()
		path = usr.HomeDir + "/Library/LaunchAgents/" + darwin.name + ".plist"
	case GlobalAgent:
		path = "/Library/LaunchAgents/" + darwin.name + ".plist"
	case GlobalDaemon:
		path = "/Library/LaunchDaemons/" + darwin.name + ".plist"
	}

	return path
}

// Is a service installed
func (darwin *darwinRecord) isInstalled() bool {

	if _, err := os.Stat(darwin.servicePath()); err == nil {
		return true
	}

	return false
}

// Get executable path
func execPath() (string, error) {
	return filepath.Abs(os.Args[0])
}

// Check service is running
func (darwin *darwinRecord) checkRunning() (string, bool) {
	output, err := exec.Command("launchctl", "list", darwin.name).Output()
	if err == nil {
		if matched, err := regexp.MatchString(darwin.name, string(output)); err == nil && matched {
			reg := regexp.MustCompile("PID\" = ([0-9]+);")
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
func (darwin *darwinRecord) Install(args ...string) (string, error) {
	installAction := "Install " + darwin.description + ":"

	ok, err := checkPrivileges()
	if !ok && darwin.kind != UserAgent {
		return installAction + failed, err
	}

	srvPath := darwin.servicePath()

	if darwin.isInstalled() {
		return installAction + failed, ErrAlreadyInstalled
	}

	file, err := os.Create(srvPath)
	if err != nil {
		return installAction + failed, err
	}
	defer file.Close()

	execPatch, err := executablePath(darwin.name)
	if err != nil {
		return installAction + failed, err
	}

	templ, err := template.New("propertyList").Parse(propertyList)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Path string
			Args       []string
		}{darwin.name, execPatch, args},
	); err != nil {
		return installAction + failed, err
	}

	return installAction + success, nil
}

// Remove the service
func (darwin *darwinRecord) Remove() (string, error) {
	removeAction := "Removing " + darwin.description + ":"

	ok, err := checkPrivileges()
	if !ok && darwin.kind != UserAgent {
		return removeAction + failed, err
	}

	if !darwin.isInstalled() {
		return removeAction + failed, ErrNotInstalled
	}

	if err := os.Remove(darwin.servicePath()); err != nil {
		return removeAction + failed, err
	}

	return removeAction + success, nil
}

// Start the service
func (darwin *darwinRecord) Start() (string, error) {
	startAction := "Starting " + darwin.description + ":"

	ok, err := checkPrivileges()
	if !ok && darwin.kind != UserAgent {
		return startAction + failed, err
	}

	if !darwin.isInstalled() {
		return startAction + failed, ErrNotInstalled
	}

	if _, ok := darwin.checkRunning(); ok {
		return startAction + failed, ErrAlreadyRunning
	}

	if err := exec.Command("launchctl", "load", darwin.servicePath()).Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (darwin *darwinRecord) Stop() (string, error) {
	stopAction := "Stopping " + darwin.description + ":"

	ok, err := checkPrivileges()
	if !ok && darwin.kind != UserAgent {
		return stopAction + failed, err
	}

	if !darwin.isInstalled() {
		return stopAction + failed, ErrNotInstalled
	}

	if _, ok := darwin.checkRunning(); !ok {
		return stopAction + failed, ErrAlreadyStopped
	}

	if err := exec.Command("launchctl", "unload", darwin.servicePath()).Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (darwin *darwinRecord) Status() (string, error) {

	ok, err := checkPrivileges()
	if !ok && darwin.kind != UserAgent {
		return "", err
	}

	if !darwin.isInstalled() {
		return statNotInstalled, ErrNotInstalled
	}

	statusAction, _ := darwin.checkRunning()

	return statusAction, nil
}

// Run - Run service
func (darwin *darwinRecord) Run(e Executable) (string, error) {
	runAction := "Running " + darwin.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

// GetTemplate - gets service config template
func (linux *darwinRecord) GetTemplate() string {
	return propertyList
}

// SetTemplate - sets service config template
func (linux *darwinRecord) SetTemplate(tplStr string) error {
	propertyList = tplStr
	return nil
}

var propertyList = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>KeepAlive</key>
	<true/>
	<key>Label</key>
	<string>{{.Name}}</string>
	<key>ProgramArguments</key>
	<array>
	    <string>{{.Path}}</string>
		{{range .Args}}<string>{{.}}</string>
		{{end}}
	</array>
	<key>RunAtLoad</key>
	<true/>
    <key>WorkingDirectory</key>
    <string>/usr/local/var</string>
    <key>StandardErrorPath</key>
    <string>/usr/local/var/log/{{.Name}}.err</string>
    <key>StandardOutPath</key>
    <string>/usr/local/var/log/{{.Name}}.log</string>
</dict>
</plist>
`
