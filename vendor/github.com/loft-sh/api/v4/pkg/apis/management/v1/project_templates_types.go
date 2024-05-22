package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ProjectTemplates struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// DefaultVirtualClusterTemplate is the default template for the project
	DefaultVirtualClusterTemplate string `json:"defaultVirtualClusterTemplate,omitempty"`

	// VirtualClusterTemplates holds all the allowed virtual cluster templates
	VirtualClusterTemplates []VirtualClusterTemplate `json:"virtualClusterTemplates,omitempty"`

	// DefaultSpaceTemplate
	DefaultSpaceTemplate string `json:"defaultSpaceTemplate,omitempty"`

	// SpaceTemplates holds all the allowed space templates
	SpaceTemplates []SpaceTemplate `json:"spaceTemplates,omitempty"`

	// DefaultDevPodWorkspaceTemplate
	DefaultDevPodWorkspaceTemplate string `json:"defaultDevPodWorkspaceTemplate,omitempty"`

	// DevPodWorkspaceTemplates holds all the allowed space templates
	DevPodWorkspaceTemplates []DevPodWorkspaceTemplate `json:"devPodWorkspaceTemplates,omitempty"`
}
