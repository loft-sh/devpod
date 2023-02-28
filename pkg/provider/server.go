package provider

import (
	"github.com/loft-sh/devpod/pkg/types"
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
}
