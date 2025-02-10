package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type BackupApply struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BackupApplySpec `json:"spec,omitempty"`
}

// BackupApplySpec defines the desired state of BackupApply
type BackupApplySpec struct {
	// Raw is the raw backup to apply
	Raw string `json:"raw,omitempty"`
}
