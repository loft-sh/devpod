package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	uiv1 "github.com/loft-sh/api/v4/pkg/apis/ui/v1"
	"github.com/loft-sh/jspolicy/pkg/apis/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This file is just used as a collector for kiosk objects we want to generate stuff for

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Kiosk holds the kiosk types
// +k8s:openapi-gen=true
// +resource:path=kiosk,rest=KioskREST
type Kiosk struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KioskSpec   `json:"spec,omitempty"`
	Status KioskStatus `json:"status,omitempty"`
}

type KioskSpec struct {
	// policy.loft.sh
	JsPolicy           v1beta1.JsPolicy           `json:"jsPolicy,omitempty"`
	JsPolicyBundle     v1beta1.JsPolicyBundle     `json:"jsPolicyBundle,omitempty"`
	JsPolicyViolations v1beta1.JsPolicyViolations `json:"jsPolicyViolations,omitempty"`

	// cluster.loft.sh
	HelmRelease        clusterv1.HelmRelease        `json:"helmRelease,omitempty"`
	SleepModeConfig    clusterv1.SleepModeConfig    `json:"sleepModeConfig,omitempty"`
	Space              clusterv1.Space              `json:"space,omitempty"`
	VirtualCluster     clusterv1.VirtualCluster     `json:"virtualCluster,omitempty"`
	LocalClusterAccess clusterv1.LocalClusterAccess `json:"localClusterAccess,omitempty"`
	ClusterQuota       clusterv1.ClusterQuota       `json:"clusterQuota,omitempty"`
	ChartInfo          clusterv1.ChartInfo          `json:"chartInfo,omitempty"`

	// storage.loft.sh
	StorageClusterAccess  agentstoragev1.LocalClusterAccess `json:"localStorageClusterAccess,omitempty"`
	StorageClusterQuota   agentstoragev1.ClusterQuota       `json:"storageClusterQuota,omitempty"`
	StorageVirtualCluster agentstoragev1.VirtualCluster     `json:"storageVirtualCluster,omitempty"`
	LocalUser             agentstoragev1.LocalUser          `json:"localUser,omitempty"`
	LocalTeam             agentstoragev1.LocalTeam          `json:"localTeam,omitempty"`

	// ui.loft.sh
	UISettings uiv1.UISettings `json:"UISettings,omitempty"`

	License License `json:"license,omitempty"`
}

type KioskStatus struct {
}
