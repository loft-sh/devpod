package licenseapi

// This code was generated. Change features.yaml to add, remove, or edit features.

// Features
const (
	VirtualCluster FeatureName = "vclusters" // Virtual Cluster Management

	VirtualClusterSleepMode FeatureName = "vcluster-sleep-mode" // Sleep Mode for Virtual Clusters

	VirtualClusterHostPathMapper FeatureName = "vcluster-host-path-mapper" // Central HostPath Mapper

	VirtualClusterEnterprisePlugins FeatureName = "vcluster-enterprise-plugins" // Enterprise Plugins

	VirtualClusterProDistroImage FeatureName = "vcp-distro-image" // Security-Hardened vCluster Image

	VirtualClusterProDistroBuiltInCoreDNS FeatureName = "vcp-distro-built-in-coredns" // Built-In CoreDNS

	VirtualClusterProDistroAdmissionControl FeatureName = "vcp-distro-admission-control" // Virtual Admission Control

	VirtualClusterProDistroSyncPatches FeatureName = "vcp-distro-sync-patches" // Sync Patches

	VirtualClusterProDistroEmbeddedEtcd FeatureName = "vcp-distro-embedded-etcd" // Embedded etcd

	VirtualClusterProDistroIsolatedControlPlane FeatureName = "vcp-distro-isolated-cp" // Isolated Control Plane

	VirtualClusterProDistroCentralizedAdmissionControl FeatureName = "vcp-distro-centralized-admission-control" // Centralized Admission Control

	VirtualClusterProDistroGenericSync FeatureName = "vcp-distro-generic-sync" // Generic Sync

	VirtualClusterProDistroTranslatePatches FeatureName = "vcp-distro-translate-patches" // Translate Patches

	VirtualClusterProDistroIntegrationsKubeVirt FeatureName = "vcp-distro-integrations-kube-virt" // KubeVirt Integration

	VirtualClusterProDistroIntegrationsExternalSecrets FeatureName = "vcp-distro-integrations-external-secrets" // External Secrets Integration

	VirtualClusterProDistroIntegrationsCertManager FeatureName = "vcp-distro-integrations-cert-manager" // Cert Manager Integration

	VirtualClusterProDistroFips FeatureName = "vcp-distro-fips" // FIPS

	VirtualClusterProDistroExternalDatabase FeatureName = "vcp-distro-external-database" // External Database

	ConnectorExternalDatabase FeatureName = "connector-external-database" // Database Connector

	VirtualClusterProDistroSleepMode FeatureName = "vcp-distro-sleep-mode" // SleepMode

	Devpod FeatureName = "devpod" // Dev Environment Management

	Namespaces FeatureName = "namespaces" // Namespace Management

	NamespaceSleepMode FeatureName = "namespace-sleep-mode" // Sleep Mode for Namespaces

	ConnectedClusters FeatureName = "connected-clusters" // Connected Clusters

	ClusterAccess FeatureName = "cluster-access" // Cluster Access

	ClusterRoles FeatureName = "cluster-roles" // Cluster Role Management

	SSOAuth FeatureName = "sso-authentication" // Single Sign-On

	AuditLogging FeatureName = "audit-logging" // Audit Logging

	AutoIngressAuth FeatureName = "auto-ingress-authentication" // Automatic Auth For Ingresses

	OIDCProvider FeatureName = "oidc-provider" // Loft as OIDC Provider

	MultipleSSOProviders FeatureName = "multiple-sso-providers" // Multiple SSO Providers

	Apps FeatureName = "apps" // Apps

	TemplateVersioning FeatureName = "template-versioning" // Template Versioning

	ArgoIntegration FeatureName = "argo-integration" // Argo Integration

	RancherIntegration FeatureName = "rancher-integration" // Rancher Integration

	Secrets FeatureName = "secrets" // Secrets Sync

	SecretEncryption FeatureName = "secret-encryption" // Secrets Encryption

	VaultIntegration FeatureName = "vault-integration" // HashiCorp Vault Integration

	HighAvailabilityMode FeatureName = "ha-mode" // High-Availability Mode

	MultiRegionMode FeatureName = "multi-region-mode" // Multi-Region Mode

	AirGappedMode FeatureName = "air-gapped-mode" // Air-Gapped Mode

	CustomBranding FeatureName = "custom-branding" // Custom Branding

	AdvancedUICustomizations FeatureName = "advanced-ui-customizations" // Advanced UI Customizations

	VNodeRuntime FeatureName = "vnode-runtime" // vNode Runtime

	ProjectQuotas FeatureName = "project-quotas" // Project Quotas

	ResolveDns FeatureName = "resolve-dns" // Resolve DNS

)

func GetFeatures() []FeatureName {
	return []FeatureName{
		VirtualCluster,
		VirtualClusterSleepMode,
		VirtualClusterHostPathMapper,
		VirtualClusterEnterprisePlugins,
		VirtualClusterProDistroImage,
		VirtualClusterProDistroBuiltInCoreDNS,
		VirtualClusterProDistroAdmissionControl,
		VirtualClusterProDistroSyncPatches,
		VirtualClusterProDistroEmbeddedEtcd,
		VirtualClusterProDistroIsolatedControlPlane,
		VirtualClusterProDistroCentralizedAdmissionControl,
		VirtualClusterProDistroGenericSync,
		VirtualClusterProDistroTranslatePatches,
		VirtualClusterProDistroIntegrationsKubeVirt,
		VirtualClusterProDistroIntegrationsExternalSecrets,
		VirtualClusterProDistroIntegrationsCertManager,
		VirtualClusterProDistroFips,
		VirtualClusterProDistroExternalDatabase,
		ConnectorExternalDatabase,
		VirtualClusterProDistroSleepMode,
		Devpod,
		Namespaces,
		NamespaceSleepMode,
		ConnectedClusters,
		ClusterAccess,
		ClusterRoles,
		SSOAuth,
		AuditLogging,
		AutoIngressAuth,
		OIDCProvider,
		MultipleSSOProviders,
		Apps,
		TemplateVersioning,
		ArgoIntegration,
		RancherIntegration,
		Secrets,
		SecretEncryption,
		VaultIntegration,
		HighAvailabilityMode,
		MultiRegionMode,
		AirGappedMode,
		CustomBranding,
		AdvancedUICustomizations,
		VNodeRuntime,
		ProjectQuotas,
		ResolveDns,
	}
}
