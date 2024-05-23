package licenseapi

// License is a struct representing the license data sent to a Loft instance after checking in with
// the license server.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type License struct {
	// InstanceID contains the instance id of the Loft instance
	InstanceID string `json:"instance,omitempty"`

	// Analytics indicates the analytics endpoints and which requests should be sent to the
	// analytics server.
	// +optional
	Analytics *Analytics `json:"analytics,omitempty"`
	// DomainToken holds the JWT with the URL that the Loft instance is publicly available on.
	// (via Loft router)
	// +optional
	DomainToken string `json:"domainToken"`
	// Buttons is a slice of license server endpoints (buttons) that the Loft instance may need to
	// hit. Each Button contains the display text and link for the front end to work with.
	Buttons []*Button `json:"buttons,omitempty"`
	// Announcements is a map string/string such that we can easily add any additional data without
	// needing to change types. For now, we will use the keys "name" and "content".
	// +optional
	Announcements []*Announcement `json:"announcement,omitempty"`
	// Modules is a list of modules.
	// +optional
	Modules []*Module `json:"modules,omitempty"`
	// BlockRequests specifies which requests the product should block when a limit is exceeded.
	// +optional
	BlockRequests []BlockRequest `json:"block,omitempty"`
	// IsOffline indicates if the license is an offline license or not.
	// +optional
	IsOffline bool `json:"isOffline,omitempty"`

	// +optional
	Routes LicenseAPIRoutes `json:"routes,omitempty"`

	// Plans contains a list of plans
	// +optional
	Plans []Plan `json:"plans,omitempty"`
}
