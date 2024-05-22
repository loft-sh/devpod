package licenseapi

import "net/http"

const InstanceCreateRoute = "/instance"
const InstanceCreateMethod = http.MethodPost

// InstanceCreateInput is the required input data for "instance create" operations, that is, the
// primary endpoint that Loft instances will hit to register to the license server as well as get
// information about the instance's current license.
// +k8s:deepcopy-gen=true
type InstanceCreateInput struct {
	*InstanceTokenAuth `hash:"-"`

	// Product is the product that is being used. Can be empty, loft, devpod-pro or vcluster-pro.
	// This should NOT be a ProductName but a string to allow for downward compatibility
	Product string `json:"product,omitempty" form:"product"`

	// Email is the admin email. Can be empty if no email is specified.
	Email string `json:"email,omitempty" form:"email"`

	LoftVersion string `json:"version"     form:"version"     validate:"required"`
	KubeVersion string `json:"kubeVersion" form:"kubeVersion" validate:"required"`

	KubeSystemNamespaceUID string `json:"kubeSystemNamespace" form:"kubeSystemNamespaceUID" validate:"required"`

	// ResourceUsage contains information about the number of resources used
	ResourceUsage map[string]ResourceCount `json:"resources,omitempty" form:"resources"`

	// FeatureUse contains information about what features are used
	FeatureUsage map[string]bool `json:"features,omitempty" form:"features"`

	// DebugInstanceID is the ID of the instance. This is only used for testing purposes.
	// Should never be sent from production instances.
	// Requires authentication via an access key.
	DebugInstanceID *string `json:"debugInstanceID,omitempty" form:"debugInstanceID" hash:"-"`
}

// InstanceCreateOutput is the struct holding all information returned from "instance create"
// requests.
// +k8s:deepcopy-gen=true
type InstanceCreateOutput struct {
	// License is the license data for the requested Loft instance.
	License     *License `json:"license,omitempty"`
	CurrentTime int64    `json:"currentTime"`
}
