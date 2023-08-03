package provider

import "github.com/loft-sh/devpod/pkg/types"

type Engine struct {
	// ID is the engine id to use
	ID string `json:"id,omitempty"`

	// URL is the Loft DevPod Engine url to use
	URL string `json:"url,omitempty"`

	// CreationTimestamp is the timestamp when this workspace was created
	CreationTimestamp types.Time `json:"creationTimestamp,omitempty"`
}
