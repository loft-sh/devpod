package licenseapi

// Invoice provides details about an invoice
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Invoice struct {
	// Date contains the unix timestamp marking the date this invoices was or will be created
	// +optional
	Date int64 `json:"date,omitempty"`

	// Total is the total of the invoice
	// +optional
	Total int64 `json:"total,omitempty"`

	// Currency specifies the currency of Total in 3-character ISO 4217 code
	// Default is: "" (representing USD)
	// +optional
	Currency string `json:"currency,omitempty"`
}
