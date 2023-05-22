package framework

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
)

const agentServerPort = "9191"

func StartAgentServer() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", ":"+agentServerPort)
	if err != nil {
		return err
	}

	err = os.Setenv("DEVPOD_AGENT_URL", "http://localhost:"+agentServerPort)
	if err != nil {
		return err
	}

	if err := http.Serve(listener, http.FileServer(http.Dir(filepath.Join(wd, "bin")))); err != nil {
		return err
	}

	return nil
}
