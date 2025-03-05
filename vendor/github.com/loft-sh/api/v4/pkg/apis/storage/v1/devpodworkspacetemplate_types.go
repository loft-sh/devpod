package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceTemplate holds the DevPodWorkspaceTemplate information
// +k8s:openapi-gen=true
type DevPodWorkspaceTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspaceTemplateSpec   `json:"spec,omitempty"`
	Status DevPodWorkspaceTemplateStatus `json:"status,omitempty"`
}

func (a *DevPodWorkspaceTemplate) GetVersions() []VersionAccessor {
	var retVersions []VersionAccessor
	for _, v := range a.Spec.Versions {
		b := v
		retVersions = append(retVersions, &b)
	}

	return retVersions
}

func (a *DevPodWorkspaceTemplateVersion) GetVersion() string {
	return a.Version
}

func (a *DevPodWorkspaceTemplate) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodWorkspaceTemplate) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodWorkspaceTemplate) GetAccess() []Access {
	return a.Spec.Access
}

func (a *DevPodWorkspaceTemplate) SetAccess(access []Access) {
	a.Spec.Access = access
}

// DevPodWorkspaceTemplateSpec holds the specification
type DevPodWorkspaceTemplateSpec struct {
	// DisplayName is the name that is shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the virtual cluster template
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Parameters define additional app parameters that will set provider values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Template holds the DevPod workspace template
	Template DevPodWorkspaceTemplateDefinition `json:"template,omitempty"`

	// Versions are different versions of the template that can be referenced as well
	// +optional
	Versions []DevPodWorkspaceTemplateVersion `json:"versions,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type DevPodWorkspaceTemplateDefinition struct {
	// Kubernetes holds the definition for kubernetes based workspaces
	Kubernetes *DevPodWorkspaceKubernetesSpec `json:"kubernetes,omitempty"`

	// WorkspaceEnv are environment variables that should be available within the created workspace.
	// +optional
	WorkspaceEnv map[string]DevPodProviderOption `json:"workspaceEnv,omitempty"`

	// InitEnv are environment variables that should be available during the initialization phase of the created workspace.
	// +optional
	InitEnv map[string]DevPodProviderOption `json:"initEnv,omitempty"`

	// InstanceTemplate holds the workspace instance template
	// +optional
	InstanceTemplate DevPodWorkspaceInstanceTemplateDefinition `json:"instanceTemplate,omitempty"`

	// GitCloneStrategy specifies how git based workspace are being cloned. Can be "" (full, default), treeless, blobless or shallow
	// +optional
	GitCloneStrategy GitCloneStrategy `json:"gitCloneStrategy,omitempty"`

	// PreventWakeUpOnConnection is used to prevent workspace that uses sleep mode from waking up on incomming ssh connection.
	// +optional
	PreventWakeUpOnConnection bool `json:"preventWakeUpOnConnection,omitempty"`

	// Provider holds the legacy VM provider configuration
	//
	// Deprecated: use fields on template instead
	// +optional
	Provider *DevPodWorkspaceProvider `json:"provider,omitempty"`
}

type DevPodWorkspaceKubernetesSpec struct {
	// Pod holds the definition for workspace pod.
	//
	// Defaults will be applied for fields that aren't specified.
	// +optional
	Pod *DevPodWorkspacePodTemplate `json:"pod,omitempty"`

	// VolumeClaim holds the definition for the main workspace persistent volume.
	// This volume is guaranteed to exist for the lifespan of the workspace.
	//
	// Defaults will be applied for fields that aren't specified.
	// +optional
	VolumeClaim *DevPodWorkspaceVolumeClaimTemplate `json:"volumeClaim,omitempty"`

	// PodTimeout specifies a maximum duration to wait for the workspace pod to start up before failing.
	// Default: 10m
	// +optional
	PodTimeout string `json:"podTimeout,omitempty"`

	// NodeArchitecture specifies the node architecture the workspace image will be built for.
	// Only necessary if you need to build workspace images on the fly in the kubernetes cluster and your cluster is mixed architecture.
	// +optional
	NodeArchitecure string `json:"nodeArchitecture,omitempty"`

	// SpaceTemplateRef is a reference to the space that should get created for this DevPod.
	// If this is specified, the kubernetes provider will be selected automatically.
	// +optional
	SpaceTemplateRef *TemplateRef `json:"spaceTemplateRef,omitempty"`

	// SpaceTemplate is the inline template for a space that should get created for this DevPod.
	// If this is specified, the kubernetes provider will be selected automatically.
	// +optional
	SpaceTemplate *SpaceTemplateDefinition `json:"spaceTemplate,omitempty"`

	// VirtualClusterTemplateRef is a reference to the virtual cluster that should get created for this DevPod.
	// If this is specified, the kubernetes provider will be selected automatically.
	// +optional
	VirtualClusterTemplateRef *TemplateRef `json:"virtualClusterTemplateRef,omitempty"`

	// VirtualClusterTemplate is the inline template for a virtual cluster that should get created for this DevPod.
	// If this is specified, the kubernetes provider will be selected automatically.
	// +optional
	VirtualClusterTemplate *VirtualClusterTemplateDefinition `json:"virtualClusterTemplate,omitempty"`
}

// DevPodWorkspacePodTemplate is a less restrictive PodTemplate
type DevPodWorkspacePodTemplate struct {
	// The pods metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`

	Spec DevPodWorkspacePodTemplateSpec `json:"spec,omitempty"`
}

// DevPodWorkspacePodTemplateSpec is a less restrictive PodSpec
type DevPodWorkspacePodTemplateSpec struct {
	// List of volumes that can be mounted by containers belonging to the pod.
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// List of initialization containers belonging to the pod.
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// List of containers belonging to the pod.
	// +optional
	Containers []corev1.Container `json:"containers,omitempty"`

	// Restart policy for all containers within the pod.
	// +optional
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"`

	// Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// Optional duration in seconds the pod may be active on the node relative to
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	// Set DNS policy for the pod.
	// +optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// AutomountServiceAccountToken indicates whether a service account token should be automatically mounted.
	// +optional
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty"`

	// NodeName indicates in which node this pod is scheduled.
	// +optional
	NodeName string `json:"nodeName,omitempty"`

	// Host networking requested for this pod. Use the host's network namespace.
	// +optional
	HostNetwork bool `json:"hostNetwork,omitempty"`

	// Use the host's pid namespace.
	// +optional
	HostPID bool `json:"hostPID,omitempty"`

	// Use the host's ipc namespace.
	// +optional
	HostIPC bool `json:"hostIPC,omitempty"`

	// Share a single process namespace between all of the containers in a pod.
	// +optional
	ShareProcessNamespace *bool `json:"shareProcessNamespace,omitempty"`

	// SecurityContext holds pod-level security attributes and common container settings.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Specifies the hostname of the Pod
	// +optional
	Hostname string `json:"hostname,omitempty"`

	// If specified, the fully qualified Pod hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>".
	// +optional
	Subdomain string `json:"subdomain,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// If specified, the pod will be dispatched by specified scheduler.
	// If not specified, the pod will be dispatched by default scheduler.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts
	// +optional
	HostAliases []corev1.HostAlias `json:"hostAliases,omitempty"`

	// If specified, indicates the pod's priority.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// Specifies the DNS parameters of a pod.
	// +optional
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`

	// If specified, all readiness gates will be evaluated for pod readiness.
	// +optional
	ReadinessGates []corev1.PodReadinessGate `json:"readinessGates,omitempty"`

	// RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used to run this pod
	// +optional
	RuntimeClassName *string `json:"runtimeClassName,omitempty"`

	// EnableServiceLinks indicates whether information about services should be injected into pod's
	// environment variables, matching the syntax of Docker links.
	// +optional
	EnableServiceLinks *bool `json:"enableServiceLinks,omitempty"`

	// PreemptionPolicy is the Policy for preempting pods with lower priority.
	// +optional
	PreemptionPolicy *corev1.PreemptionPolicy `json:"preemptionPolicy,omitempty"`

	// Overhead represents the resource overhead associated with running a pod for a given RuntimeClass.
	// +optional
	Overhead corev1.ResourceList `json:"overhead,omitempty"`

	// TopologySpreadConstraints describes how a group of pods ought to spread across topology domains.
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// If true the pod's hostname will be configured as the pod's FQDN, rather than the leaf name (the default).
	// In Linux containers, this means setting the FQDN in the hostname field of the kernel (the nodename field of struct utsname).
	// In Windows containers, this means setting the registry value of hostname for the registry key HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\Tcpip\\Parameters to FQDN.
	// +optional
	SetHostnameAsFQDN *bool `json:"setHostnameAsFQDN,omitempty"`

	// Specifies the OS of the containers in the pod.
	// +optional
	OS *corev1.PodOS `json:"os,omitempty"`

	// Use the host's user namespace.
	// +optional
	HostUsers *bool `json:"hostUsers,omitempty"`

	// SchedulingGates is an opaque list of values that if specified will block scheduling the pod.
	// If schedulingGates is not empty, the pod will stay in the SchedulingGated state and the
	// scheduler will not attempt to schedule the pod.
	// +optional
	SchedulingGates []corev1.PodSchedulingGate `json:"schedulingGates,omitempty"`

	// ResourceClaims defines which ResourceClaims must be allocated
	// and reserved before the Pod is allowed to start. The resources
	// will be made available to those containers which consume them
	// by name.
	// +optional
	ResourceClaims []corev1.PodResourceClaim `json:"resourceClaims,omitempty"`

	// Resources is the total amount of CPU and Memory resources required by all
	// containers in the pod. It supports specifying Requests and Limits for
	// "cpu" and "memory" resource names only. ResourceClaims are not supported.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type DevPodWorkspaceVolumeClaimTemplate struct {
	// The pods metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`

	Spec corev1.PersistentVolumeClaimSpec `json:"spec,omitempty"`
}

// +enum
type GitCloneStrategy string

// WARN: Need to match https://github.com/loft-sh/devpod/pkg/git/clone.go
const (
	FullCloneStrategy     GitCloneStrategy = ""
	BloblessCloneStrategy GitCloneStrategy = "blobless"
	TreelessCloneStrategy GitCloneStrategy = "treeless"
	ShallowCloneStrategy  GitCloneStrategy = "shallow"
)

type DevPodWorkspaceProvider struct {
	// Name is the name of the provider. This can also be an url.
	// +optional
	Name string `json:"name,omitempty"`

	// Options are the provider option values
	// +optional
	Options map[string]DevPodProviderOption `json:"options,omitempty"`

	// Env are environment options to set when using the provider.
	// +optional
	Env map[string]DevPodProviderOption `json:"env,omitempty"`
}

type DevPodWorkspaceInstanceTemplateDefinition struct {
	// The workspace instance metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`
}

type DevPodProviderOption struct {
	// Value of this option.
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom specifies a secret where this value should be taken from.
	// +optional
	ValueFrom *DevPodProviderOptionFrom `json:"valueFrom,omitempty"`
}

type DevPodProviderOptionFrom struct {
	// ProjectSecretRef is the project secret to use for this value.
	// +optional
	ProjectSecretRef *corev1.SecretKeySelector `json:"projectSecretRef,omitempty"`

	// SharedSecretRef is the shared secret to use for this value.
	// +optional
	SharedSecretRef *corev1.SecretKeySelector `json:"sharedSecretRef,omitempty"`
}

type DevPodProviderSource struct {
	// Github source for the provider
	Github string `json:"github,omitempty"`

	// File source for the provider
	File string `json:"file,omitempty"`

	// URL where the provider was downloaded from
	URL string `json:"url,omitempty"`
}

type DevPodWorkspaceTemplateVersion struct {
	// Template holds the DevPod template
	// +optional
	Template DevPodWorkspaceTemplateDefinition `json:"template,omitempty"`

	// Parameters define additional app parameters that will set provider values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Version is the version. Needs to be in X.X.X format.
	// +optional
	Version string `json:"version,omitempty"`
}

// DevPodWorkspaceTemplateStatus holds the status
type DevPodWorkspaceTemplateStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceTemplateList contains a list of DevPodWorkspaceTemplate
type DevPodWorkspaceTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevPodWorkspaceTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DevPodWorkspaceTemplate{}, &DevPodWorkspaceTemplateList{})
}
