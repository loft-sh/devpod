// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

/*
Package daemon v1.0.0 for use with Go (golang) services.

Package daemon provides primitives for daemonization of golang services. In the
current implementation the only supported operating systems are macOS, FreeBSD,
Linux and Windows. Also to note, for global daemons one must have root rights to
install or remove the service. The only exception is macOS where there is an
implementation of a user daemon that can installed or removed by the current
user.

Example:

	// Example of a daemon with echo service
	package main

	import (
		"fmt"
		"log"
		"net"
		"os"
		"os/signal"
		"syscall"

		"github.com/takama/daemon"
	)

	const (

		// name of the service
		name        = "myservice"
		description = "My Echo Service"

		// port which daemon should be listen
		port = ":9977"
	)

  // dependencies that are NOT required by the service, but might be used
  var dependencies = []string{"dummy.service"}

	var stdlog, errlog *log.Logger

	// Service has embedded daemon
	type Service struct {
		daemon.Daemon
	}

	// Manage by daemon commands or run the daemon
	func (service *Service) Manage() (string, error) {

		usage := "Usage: myservice install | remove | start | stop | status"

		// if received any kind of command, do it
		if len(os.Args) > 1 {
			command := os.Args[1]
			switch command {
			case "install":
				return service.Install()
			case "remove":
				return service.Remove()
			case "start":
				return service.Start()
			case "stop":
				return service.Stop()
			case "status":
				return service.Status()
			default:
				return usage, nil
			}
		}

		// Do something, call your goroutines, etc

		// Set up channel on which to send signal notifications.
		// We must use a buffered channel or risk missing the signal
		// if we're not ready to receive when the signal is sent.
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

		// Set up listener for defined host and port
		listener, err := net.Listen("tcp", port)
		if err != nil {
			return "Possibly was a problem with the port binding", err
		}

		// set up channel on which to send accepted connections
		listen := make(chan net.Conn, 100)
		go acceptConnection(listener, listen)

		// loop work cycle with accept connections or interrupt
		// by system signal
		for {
			select {
			case conn := <-listen:
				go handleClient(conn)
			case killSignal := <-interrupt:
				stdlog.Println("Got signal:", killSignal)
				stdlog.Println("Stoping listening on ", listener.Addr())
				listener.Close()
				if killSignal == os.Interrupt {
					return "Daemon was interrupted by system signal", nil
				}
				return "Daemon was killed", nil
			}
		}

		// never happen, but need to complete code
		return usage, nil
	}

	// Accept a client connection and collect it in a channel
	func acceptConnection(listener net.Listener, listen chan<- net.Conn) {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			listen <- conn
		}
	}

	func handleClient(client net.Conn) {
		for {
			buf := make([]byte, 4096)
			numbytes, err := client.Read(buf)
			if numbytes == 0 || err != nil {
				return
			}
			client.Write(buf[:numbytes])
		}
	}

	func init() {
		stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
		errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
	}

	func main() {
		srv, err := daemon.New(name, description, daemon.SystemDaemon, dependencies...)
		if err != nil {
			errlog.Println("Error: ", err)
			os.Exit(1)
		}
		service := &Service{srv}
		status, err := service.Manage()
		if err != nil {
			errlog.Println(status, "\nError: ", err)
			os.Exit(1)
		}
		fmt.Println(status)
	}

Go daemon
*/
package daemon

import (
	"errors"
	"runtime"
	"strings"
)

// Status constants.
const (
	statNotInstalled = "Service not installed"
)

// Daemon interface has a standard set of methods/commands
type Daemon interface {
	// GetTemplate - gets service config template
	GetTemplate() string

	// SetTemplate - sets service config template
	SetTemplate(string) error

	// Install the service into the system
	Install(args ...string) (string, error)

	// Remove the service and all corresponding files from the system
	Remove() (string, error)

	// Start the service
	Start() (string, error)

	// Stop the service
	Stop() (string, error)

	// Status - check the service status
	Status() (string, error)

	// Run - run executable service
	Run(e Executable) (string, error)
}

// Executable interface defines controlling methods of executable service
type Executable interface {
	// Start - non-blocking start service
	Start()
	// Stop - non-blocking stop service
	Stop()
	// Run - blocking run service
	Run()
}

// Kind is type of the daemon
type Kind string

const (
	// UserAgent is a user daemon that runs as the currently logged in user and
	// stores its property list in the user’s individual LaunchAgents directory.
	// In other words, per-user agents provided by the user. Valid for macOS only.
	UserAgent Kind = "UserAgent"

	// GlobalAgent is a user daemon that runs as the currently logged in user and
	// stores its property list in the users' global LaunchAgents directory. In
	// other words, per-user agents provided by the administrator. Valid for macOS
	// only.
	GlobalAgent Kind = "GlobalAgent"

	// GlobalDaemon is a system daemon that runs as the root user and stores its
	// property list in the global LaunchDaemons directory. In other words,
	// system-wide daemons provided by the administrator. Valid for macOS only.
	GlobalDaemon Kind = "GlobalDaemon"

	// SystemDaemon is a system daemon that runs as the root user. In other words,
	// system-wide daemons provided by the administrator. Valid for FreeBSD, Linux
	// and Windows only.
	SystemDaemon Kind = "SystemDaemon"
)

// New - Create a new daemon
//
// name: name of the service
//
// description: any explanation, what is the service, its purpose
//
// kind: what kind of daemon to create
func New(name, description string, kind Kind, dependencies ...string) (Daemon, error) {
	switch runtime.GOOS {
	case "darwin":
		if kind == SystemDaemon {
			return nil, errors.New("Invalid daemon kind specified")
		}
	case "freebsd":
		if kind != SystemDaemon {
			return nil, errors.New("Invalid daemon kind specified")
		}
	case "linux":
		if kind != SystemDaemon {
			return nil, errors.New("Invalid daemon kind specified")
		}
	case "windows":
		if kind != SystemDaemon {
			return nil, errors.New("Invalid daemon kind specified")
		}
	}

	return newDaemon(strings.Join(strings.Fields(name), "_"), description, kind, dependencies)
}
