package licenseapi

// Plan definition
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Plan struct {
	// ID of the plan
	ID string `json:"id,omitempty"`

	// DisplayName is the display name of the plan
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Status is the status of the plan
	// There should only be 1 active plan at the top-level (not including AddOns)
	// The respective price in Prices will have the active status as well
	// +optional
	Status PlanStatus `json:"status,omitempty"`

	// Period provides information about the plan's current period
	// This is nil unless this is the active plan
	// +optional
	Period *PlanPeriod `json:"period,omitempty"`

	// Trial provides details about a planned, ongoing or expired trial
	// +optional
	Trial *Trial `json:"trial,omitempty"`

	// UpcomingInvoice provides a preview of the next invoice that will be created for this Plan
	// +optional
	UpcomingInvoice *Invoice `json:"invoice,omitempty"`

	// Features is a list of features included in the plan
	// +optional
	Features []string `json:"features,omitempty"`

	// Limits is a list of resources included in the plan and their limits
	// +optional
	Limits []Limit `json:"limits,omitempty"`

	// Prices provides details about the available prices (depending on the interval, for example)
	// +optional
	Prices []PlanPrice `json:"prices,omitempty"`

	// AddOns are plans that can be added to this plan
	// +optional
	AddOns []Plan `json:"addons,omitempty"`
}

// PlanPeriod provides details about the period of the plan
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type PlanPeriod struct {
	// CurrentPeriodStart contains the unix timestamp marking the start of the current period
	// +optional
	CurrentPeriodStart int64 `json:"start,omitempty"`

	// CurrentPeriodEnd contains the unix timestamp marking the end of the current period
	// +optional
	CurrentPeriodEnd int64 `json:"end,omitempty"`
}

// PlanExpiration provides details about the expiration of a plan
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type PlanExpiration struct {
	// ExpiresAt is the unix timestamp of when the plan expires
	// +optional
	ExpiresAt int64 `json:"expiresAt,omitempty"`

	// UpgradesTo states the name of the plan that is replacing the current one upon its expiration
	// If this is nil, then this plan just expires (i.e. the subscription may be canceled, paused, etc.)
	// +optional
	UpgradesTo *string `json:"upgradesTo,omitempty"`
}

// PlanPrice defines a price for the plan
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type PlanPrice struct {
	// ID of the price
	ID string `json:"id,omitempty"`

	// Status is the status of the price (PlanStatus)
	// If the plan is active, one of its prices must be active as well
	// +optional
	Status PlanStatus `json:"status,omitempty"`

	// Interval contains the time span of each period (e.g. month, year)
	// +optional
	Interval PlanInterval `json:"interval,omitempty"`

	// IntervalCount specifies if the number of intervals (e.g. 3 [months])
	// +optional
	IntervalCount float64 `json:"intervalCount,omitempty"`

	// Expiration provides information about when this plan expires
	// +optional
	Expiration *PlanExpiration `json:"exp,omitempty"`

	// Quantity sets the quantity the TierResource is supposed to be at
	// If this is the active price, then this is the subscription quantity (currently purchased quantity)
	// +optional
	Quantity float64 `json:"quantity,omitempty"`

	// TierResource provides details about the main resource the tier quantity relates to
	// This may be nil for plans that don't have their quantity tied to a resource
	// +optional
	TierResource *TierResource `json:"resource,omitempty"`

	// TierMode defines how tiers should be used
	// +optional
	TierMode TierMode `json:"tierMode,omitempty"`

	// Tiers is a list of tiers in this plan
	// +optional
	Tiers []PriceTier `json:"tiers,omitempty"`
}

// TierResource provides details about the main resource the tier quantity relates to
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type TierResource struct {
	// Name of the resource (ResourceName)
	Name string `json:"name,omitempty"`

	// Status defines which resources will be counted towards the limit (e.g. active, total, total created etc.)
	// +optional
	Status ResourceStatus `json:"status,omitempty"`
}

// PriceTier defines a tier within a plan
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type PriceTier struct {
	// MinQuantity is the quantity included in this plan
	// +optional
	MinQuantity float64 `json:"min,omitempty"`

	// MaxQuantity is the max quantity that can be purchased
	// +optional
	MaxQuantity float64 `json:"max,omitempty"`

	// UnitPrice is the price per unit in this tier
	// +optional
	UnitPrice float64 `json:"unitPrice,omitempty"`

	// FlatFee is the flat fee for this tier
	// +optional
	FlatFee float64 `json:"flatFee,omitempty"`

	// Currency specifies the currency of UnitPrice and FlatFee in 3-character ISO 4217 code
	// Default is: "" (representing USD)
	// +optional
	Currency string `json:"currency,omitempty"`
}
