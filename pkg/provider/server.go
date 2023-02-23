package provider

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/types"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	// ID is the server id to use
	ID string `json:"id,omitempty"`

	// Folder is the local folder where server related contents will be stored
	Folder string `json:"folder,omitempty"`

	// Provider is the provider used to create this workspace
	Provider ServerProviderConfig `json:"provider,omitempty"`

	// CreationTimestamp is the timestamp when this workspace was created
	CreationTimestamp types.Time `json:"creationTimestamp,omitempty"`

	// Context is the context where this config file was loaded from
	Context string `json:"context,omitempty"`

	// Origin is the place where this config file was loaded from
	Origin string `json:"-"`
}

type ServerProviderConfig struct {
	// Name is the provider name used to deploy this server
	Name string `json:"name,omitempty"`

	// Options are the provider options used to create this server
	Options map[string]config.OptionValue `json:"options,omitempty"`
}

func ToOptionsServer(s *Server) map[string]string {
	retVars := map[string]string{}
	if s == nil {
		return retVars
	}
	for optionName, optionValue := range s.Provider.Options {
		retVars[strings.ToUpper(optionName)] = optionValue.Value
	}
	if s.ID != "" {
		retVars[SERVER_ID] = s.ID
	}
	if s.Folder != "" {
		retVars[SERVER_FOLDER] = filepath.ToSlash(s.Folder)
	}
	if s.Context != "" {
		retVars[SERVER_CONTEXT] = s.Context
	}
	if s.Provider.Name != "" {
		retVars[SERVER_PROVIDER] = s.Provider.Name
	}

	// devpod binary
	devPodBinary, _ := os.Executable()
	retVars[DEVPOD] = filepath.ToSlash(devPodBinary)
	return retVars
}
