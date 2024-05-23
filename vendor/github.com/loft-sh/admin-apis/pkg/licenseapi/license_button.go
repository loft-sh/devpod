package licenseapi

// Button is an object that represents a button in the Loft UI that links to some external service
// for handling operations for licensing for example.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Button struct {
	// Name is the name of the button (ButtonName). Optional.
	// +optional
	Name string `json:"name,omitempty"`

	// URL is the link at the other end of the button.
	ActionURL string `json:"url"`
	// DisplayText is the text to display on the button. If display text is unset the button will
	// never be shown in the loft UI.
	// +optional
	DisplayText string `json:"displayText,omitempty"`
	// Direct indicates if the Loft front end should directly hit this endpoint. If false, it means
	// that the Loft front end will be hitting the license server first to generate a one time token
	// for the operation; this also means that there will be a redirect URL in the response to the
	// request for this and that link should be followed by the front end.
	// +optional
	Direct bool `json:"direct,omitempty"`
}
