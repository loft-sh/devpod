package licenseapi

var Limits = map[ResourceName]*Limit{
	ConnectedClusterLimit: {
		DisplayName: "Connected Clusters",
		Name:        string(ConnectedClusterLimit),
	},
	VirtualClusterInstanceLimit: {
		DisplayName: "Virtual Clusters",
		Name:        string(VirtualClusterInstanceLimit),
	},
	DevPodWorkspaceInstanceLimit: {
		DisplayName: "Dev Environments",
		Name:        string(DevPodWorkspaceInstanceLimit),
	},
	UserLimit: {
		DisplayName: "Users",
		Name:        string(UserLimit),
	},
	InstanceLimit: {
		DisplayName: "Instances",
		Name:        string(InstanceLimit),
	},
}

// Limit defines a limit set in the license
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Limit struct {
	// Name is the name of the resource (ResourceName)
	// +optional
	Name string `json:"name,omitempty"`

	// DisplayName is for display purposes.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Limit specifies the limit for this resource.
	// +optional
	Quantity *ResourceCount `json:"quantity,omitempty"`
}
