package provider

import "github.com/loft-sh/devpod/pkg/types"

type ProInstance struct {
	// ID is the pro id to use
	ID string `json:"id,omitempty"`

	// URL is the Loft DevPod Pro url to use
	URL string `json:"url,omitempty"`

	// CreationTimestamp is the timestamp when this pro instance was created
	CreationTimestamp types.Time `json:"creationTimestamp,omitempty"`
}
