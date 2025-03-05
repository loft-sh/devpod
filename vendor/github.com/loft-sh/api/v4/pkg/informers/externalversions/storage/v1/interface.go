// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/loft-sh/api/v4/pkg/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// AccessKeys returns a AccessKeyInformer.
	AccessKeys() AccessKeyInformer
	// Apps returns a AppInformer.
	Apps() AppInformer
	// Clusters returns a ClusterInformer.
	Clusters() ClusterInformer
	// ClusterAccesses returns a ClusterAccessInformer.
	ClusterAccesses() ClusterAccessInformer
	// ClusterRoleTemplates returns a ClusterRoleTemplateInformer.
	ClusterRoleTemplates() ClusterRoleTemplateInformer
	// DevPodEnvironmentTemplates returns a DevPodEnvironmentTemplateInformer.
	DevPodEnvironmentTemplates() DevPodEnvironmentTemplateInformer
	// DevPodWorkspaceInstances returns a DevPodWorkspaceInstanceInformer.
	DevPodWorkspaceInstances() DevPodWorkspaceInstanceInformer
	// DevPodWorkspacePresets returns a DevPodWorkspacePresetInformer.
	DevPodWorkspacePresets() DevPodWorkspacePresetInformer
	// DevPodWorkspaceTemplates returns a DevPodWorkspaceTemplateInformer.
	DevPodWorkspaceTemplates() DevPodWorkspaceTemplateInformer
	// NetworkPeers returns a NetworkPeerInformer.
	NetworkPeers() NetworkPeerInformer
	// Projects returns a ProjectInformer.
	Projects() ProjectInformer
	// SharedSecrets returns a SharedSecretInformer.
	SharedSecrets() SharedSecretInformer
	// SpaceInstances returns a SpaceInstanceInformer.
	SpaceInstances() SpaceInstanceInformer
	// SpaceTemplates returns a SpaceTemplateInformer.
	SpaceTemplates() SpaceTemplateInformer
	// Tasks returns a TaskInformer.
	Tasks() TaskInformer
	// Teams returns a TeamInformer.
	Teams() TeamInformer
	// Users returns a UserInformer.
	Users() UserInformer
	// VirtualClusterInstances returns a VirtualClusterInstanceInformer.
	VirtualClusterInstances() VirtualClusterInstanceInformer
	// VirtualClusterTemplates returns a VirtualClusterTemplateInformer.
	VirtualClusterTemplates() VirtualClusterTemplateInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// AccessKeys returns a AccessKeyInformer.
func (v *version) AccessKeys() AccessKeyInformer {
	return &accessKeyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Apps returns a AppInformer.
func (v *version) Apps() AppInformer {
	return &appInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Clusters returns a ClusterInformer.
func (v *version) Clusters() ClusterInformer {
	return &clusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterAccesses returns a ClusterAccessInformer.
func (v *version) ClusterAccesses() ClusterAccessInformer {
	return &clusterAccessInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterRoleTemplates returns a ClusterRoleTemplateInformer.
func (v *version) ClusterRoleTemplates() ClusterRoleTemplateInformer {
	return &clusterRoleTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// DevPodEnvironmentTemplates returns a DevPodEnvironmentTemplateInformer.
func (v *version) DevPodEnvironmentTemplates() DevPodEnvironmentTemplateInformer {
	return &devPodEnvironmentTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// DevPodWorkspaceInstances returns a DevPodWorkspaceInstanceInformer.
func (v *version) DevPodWorkspaceInstances() DevPodWorkspaceInstanceInformer {
	return &devPodWorkspaceInstanceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// DevPodWorkspacePresets returns a DevPodWorkspacePresetInformer.
func (v *version) DevPodWorkspacePresets() DevPodWorkspacePresetInformer {
	return &devPodWorkspacePresetInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// DevPodWorkspaceTemplates returns a DevPodWorkspaceTemplateInformer.
func (v *version) DevPodWorkspaceTemplates() DevPodWorkspaceTemplateInformer {
	return &devPodWorkspaceTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// NetworkPeers returns a NetworkPeerInformer.
func (v *version) NetworkPeers() NetworkPeerInformer {
	return &networkPeerInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Projects returns a ProjectInformer.
func (v *version) Projects() ProjectInformer {
	return &projectInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// SharedSecrets returns a SharedSecretInformer.
func (v *version) SharedSecrets() SharedSecretInformer {
	return &sharedSecretInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// SpaceInstances returns a SpaceInstanceInformer.
func (v *version) SpaceInstances() SpaceInstanceInformer {
	return &spaceInstanceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// SpaceTemplates returns a SpaceTemplateInformer.
func (v *version) SpaceTemplates() SpaceTemplateInformer {
	return &spaceTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Tasks returns a TaskInformer.
func (v *version) Tasks() TaskInformer {
	return &taskInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Teams returns a TeamInformer.
func (v *version) Teams() TeamInformer {
	return &teamInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Users returns a UserInformer.
func (v *version) Users() UserInformer {
	return &userInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// VirtualClusterInstances returns a VirtualClusterInstanceInformer.
func (v *version) VirtualClusterInstances() VirtualClusterInstanceInformer {
	return &virtualClusterInstanceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// VirtualClusterTemplates returns a VirtualClusterTemplateInformer.
func (v *version) VirtualClusterTemplates() VirtualClusterTemplateInformer {
	return &virtualClusterTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
