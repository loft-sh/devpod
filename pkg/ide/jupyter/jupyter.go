package jupyter

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/loft-sh/log"
)

const (
	OpenOption        = "OPEN"
	BindAddressOption = "BIND_ADDRESS"
)

var Options = ide.Options{
	BindAddressOption: {
		Name:        BindAddressOption,
		Description: "The address to bind the server to locally. E.g. 0.0.0.0:12345",
		Default:     "",
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

const DefaultServerPort = 10700

func NewJupyterNotebookServer(workspaceFolder string, userName string, values map[string]config.OptionValue, log log.Logger) *JupyterNotbookServer {
	return &JupyterNotbookServer{
		values:          values,
		workspaceFolder: workspaceFolder,
		userName:        userName,
		log:             log,
	}
}

type JupyterNotbookServer struct {
	values          map[string]config.OptionValue
	workspaceFolder string
	userName        string
	log             log.Logger
}

func (o *JupyterNotbookServer) Install() error {
	err := o.installNotebook()
	if err != nil {
		return err
	}

	return o.Start()
}

func (o *JupyterNotbookServer) installNotebook() error {
	if command.ExistsForUser("jupyter", o.userName) {
		return nil
	}

	// check if pip3 exists
	baseCommand := ""
	if command.ExistsForUser("pip3", o.userName) {
		baseCommand = "pip3"
	} else if command.ExistsForUser("pip", o.userName) {
		baseCommand = "pip"
	} else {
		return fmt.Errorf("seems like neither pip3 nor pip exists, please make sure to install python correctly")
	}

	// install notebook command
	runCommand := fmt.Sprintf("%s install notebook", baseCommand)
	args := []string{}
	if o.userName != "" {
		args = append(args, "su", o.userName, "-c", runCommand)
	} else {
		args = append(args, "sh", "-c", runCommand)
	}

	// install
	o.log.Infof("Installing jupyter notebook...")
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error installing jupyter notebook: %w", command.WrapCommandError(out, err))
	}

	o.log.Info("Successfully installed jupyter notebook")
	return nil
}

func (o *JupyterNotbookServer) Start() error {
	return single.Single("jupyter.pid", func() (*exec.Cmd, error) {
		o.log.Infof("Starting jupyter notebook in background...")
		runCommand := fmt.Sprintf("jupyter notebook --ip='*' --NotebookApp.notebook_dir='%s' --NotebookApp.token='' --NotebookApp.password='' --no-browser --port '%s' --allow-root", o.workspaceFolder, strconv.Itoa(DefaultServerPort))
		args := []string{}
		if o.userName != "" {
			args = append(args, "su", o.userName, "-w", "SSH_AUTH_SOCK", "-l", "-c", runCommand)
		} else {
			args = append(args, "sh", "-l", "-c", runCommand)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = o.workspaceFolder
		return cmd, nil
	})
}
