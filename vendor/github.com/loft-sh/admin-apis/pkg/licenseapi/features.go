package licenseapi

// Features
const (
	// DevPod
	DevPod FeatureName = "devpod"

	// Virtual Clusters
	// when adding a new vCluster feature, add it to GetVClusterFeatures() as well
	VirtualCluster                                     FeatureName = "vclusters"
	VirtualClusterSleepMode                            FeatureName = "vcluster-sleep-mode"
	VirtualClusterCentralHostPathMapper                FeatureName = "vcluster-host-path-mapper"
	VirtualClusterProDistroImage                       FeatureName = "vcp-distro-image"
	VirtualClusterProDistroAdmissionControl            FeatureName = "vcp-distro-admission-control"
	VirtualClusterProDistroBuiltInCoreDNS              FeatureName = "vcp-distro-built-in-coredns"
	VirtualClusterProDistroIsolatedControlPlane        FeatureName = "vcp-distro-isolated-cp"
	VirtualClusterProDistroSyncPatches                 FeatureName = "vcp-distro-sync-patches"
	VirtualClusterProDistroCentralizedAdmissionControl FeatureName = "vcp-distro-centralized-admission-control"
	VirtualClusterProEmbeddedEtcd                      FeatureName = "vcp-distro-embedded-etcd"

	// Spaces & Clusters
	ConnectedClusters  FeatureName = "connected-clusters"
	ClusterAccess      FeatureName = "cluster-access"
	ClusterRoles       FeatureName = "cluster-roles"
	Namespace          FeatureName = "namespaces"
	NamespaceSleepMode FeatureName = "namespace-sleep-mode"

	// Auth-Related Features
	AuditLogging         FeatureName = "audit-logging"
	AutomaticIngressAuth FeatureName = "auto-ingress-authentication"
	MultipleSSOProviders FeatureName = "multiple-sso-providers"
	OIDCProvider         FeatureName = "oidc-provider"
	SSOAuth              FeatureName = "sso-authentication"

	// Templating Features
	Apps               FeatureName = "apps"
	TemplateVersioning FeatureName = "template-versioning"

	// Secrets
	Secrets          FeatureName = "secrets"
	SecretEncryption FeatureName = "secret-encryption"

	// Integrations
	ArgoIntegration    FeatureName = "argo-integration"
	VaultIntegration   FeatureName = "vault-integration"
	RancherIntegration FeatureName = "rancher-integration"

	// HA & Other Advanced Deployment Features
	AirGappedMode        FeatureName = "air-gapped-mode"
	HighAvailabilityMode FeatureName = "ha-mode"
	MultiRegionMode      FeatureName = "multi-region-mode"

	// UI Customization Features
	AdvancedUICustomizations FeatureName = "advanced-ui-customizations"
	CustomBranding           FeatureName = "custom-branding"

	// Internal Features - not to be directly used by the license service
	Metrics                FeatureName = "metrics"
	Runners                FeatureName = "runners"
	ConnectLocalCluster    FeatureName = "connect-local-cluster"
	PasswordAuthentication FeatureName = "password-auth"
)

func GetVClusterFeatures() []FeatureName {
	return []FeatureName{
		VirtualCluster,
		VirtualClusterSleepMode,
		VirtualClusterCentralHostPathMapper,
		VirtualClusterProDistroImage,
		VirtualClusterProDistroAdmissionControl,
		VirtualClusterProDistroBuiltInCoreDNS,
		VirtualClusterProDistroIsolatedControlPlane,
		VirtualClusterProDistroSyncPatches,
		VirtualClusterProDistroCentralizedAdmissionControl,
		VirtualClusterProEmbeddedEtcd,
	}
}
