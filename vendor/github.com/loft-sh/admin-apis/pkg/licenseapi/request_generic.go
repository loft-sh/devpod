package licenseapi

// GenericRequestInput defines the payload that needs to be sent to a button's action URL
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type GenericRequestInput struct {
	*InstanceTokenAuth `json:",omitempty" hash:"-"`

	// Payload provides the json encoded payload
	// +optional
	Payload string `json:"payload,omitempty"`

	// ReturnURL is the url from which the request is initiated
	// +optional
	ReturnURL string `json:"returnURL,omitempty"`
}

// GenericRequestOutput specifies the response
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type GenericRequestOutput struct {
	// RedirectURL to redirect the user to.
	// +optional
	RedirectURL string `json:"redirectURL,omitempty"`

	// HTML to display to the user.
	// +optional
	HTML string `json:"html,omitempty"`

	// Buttons to be shown to the user alongside other content (e.g. HTML).
	// +optional
	Buttons []Button `json:"buttons,omitempty"`
}
