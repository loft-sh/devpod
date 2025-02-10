package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Backup holds the Backup information
// +k8s:openapi-gen=true
// +resource:path=backups,rest=BackupREST
// +subresource:request=BackupApply,path=apply,kind=BackupApply,rest=BackupApplyREST
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec   `json:"spec,omitempty"`
	Status BackupStatus `json:"status,omitempty"`
}

// BackupSpec holds the spec
type BackupSpec struct{}

// BackupStatus holds the status
type BackupStatus struct {
	RawBackup string `json:"rawBackup,omitempty"`
}
