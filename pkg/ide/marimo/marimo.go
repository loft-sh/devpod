package marimo

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/loft-sh/log"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/single"
)

const DefaultServerPort = 10710
const (
	OpenOption        = "OPEN"
	AccessToken       = "ACCESS_TOKEN"
	BindAddressOption = "BIND_ADDRESS"
)

var Options = ide.Options{
	BindAddressOption: {
		Name:        BindAddressOption,
		Description: "The address to bind the server to locally. E.g. 0.0.0.0:12345",
		Default:     "",
	},
	AccessToken: {
		Name:        AccessToken,
		Description: "Access token to authenticate with the server",
		Default:     "NhLpVl4re5PFd3QRFxvQ",
	},
	OpenOption: {
		Name:        OpenOption,
		Description: "If DevPod should automatically open the browser",
		Default:     "true",
		Enum: []string{
			"true",
			"false",
		},
	},
}

type Server struct {
	opts            map[string]config.OptionValue
	userName        string
	workspaceFolder string
	log             log.Logger
}

func NewServer(workspaceFolder, userName string, opts map[string]config.OptionValue, log log.Logger) *Server {
	return &Server{
		opts:            opts,
		workspaceFolder: workspaceFolder,
		userName:        userName,
		log:             log,
	}
}

func (s *Server) Install() error {
	if command.ExistsForUser("marimo", s.userName) {
		return nil
	}

	// check if pip3 exists
	baseCommand := ""
	if command.ExistsForUser("pip3", s.userName) {
		baseCommand = "pip3"
	} else if command.ExistsForUser("pip", s.userName) {
		baseCommand = "pip"
	} else {
		return fmt.Errorf("seems like neither pip3 nor pip exists, please make sure to install python correctly")
	}

	// install notebook command
	runCommand := fmt.Sprintf("%s install marimo", baseCommand)
	args := []string{}
	if s.userName != "" {
		args = append(args, "su", s.userName, "-c", runCommand)
	} else {
		args = append(args, "sh", "-c", runCommand)
	}

	// install
	s.log.Infof("Installing marimo...")
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error installing marimo: %w", command.WrapCommandError(out, err))
	}
	return s.start()
}

func (s *Server) start() error {
	return single.Single("marimo.pid", func() (*exec.Cmd, error) {
		s.log.Infof("Starting marimo in background...")
		token := Options.GetValue(s.opts, AccessToken)
		runCommand := fmt.Sprintf("marimo edit --headless --host 0.0.0.0 --port %s --token-password %s", strconv.Itoa(DefaultServerPort), token)
		args := []string{}
		if s.userName != "" {
			args = append(args, "su", s.userName, "-l", "-c", runCommand)
		} else {
			args = append(args, "sh", "-l", "-c", runCommand)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = s.workspaceFolder
		return cmd, nil
	})
}
