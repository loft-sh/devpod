package licenseapi

// Announcement contains an announcement that should be shown within the Loft instance.
// This information is sent to Loft instances when they check in with the license server.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Announcement struct {
	// Name contains the resource name of the announcement
	Name string `json:"name,omitempty"`
	// Title contains the title of the announcement in HTML format.
	Title string `json:"title,omitempty"`
	// Body contains the main message of the announcement in HTML format.
	Body string `json:"body,omitempty"`
	// Buttons to show alongside the announcement
	Buttons []*Button `json:"buttons,omitempty"`
}
