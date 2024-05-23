package v1

import (
	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Announcement holds the announcement information
// +k8s:openapi-gen=true
// +resource:path=announcements,rest=AnnouncementREST
type Announcement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnnouncementSpec   `json:"spec,omitempty"`
	Status AnnouncementStatus `json:"status,omitempty"`
}

type AnnouncementSpec struct {
}

type AnnouncementStatus struct {
	// Announcement is the html announcement that should be displayed in the frontend
	// +optional
	Announcement licenseapi.Announcement `json:"announcement,omitempty"`
}
