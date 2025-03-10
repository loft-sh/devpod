package licenseapi

// This code was generated. Change features.yaml to add, remove, or edit features.

import (
	"cmp"
	"slices"
)

func New() *License {
	limits := make([]*Limit, 0, len(Limits))
	for _, limit := range Limits {
		limits = append(limits, limit)
	}
	slices.SortFunc(limits, func(a, b *Limit) int {
		return cmp.Compare(a.Name, b.Name)
	})

	// Sorting features by module is not requires here. However, to maintain backwards compatibility, the structure of
	// features being contained within a module is still necessary. Therefore, all features are now returned in one module.
	return &License{
		Modules: []*Module{
			{
				DisplayName: "All Features",
				Name:        string(VirtualClusterModule),
				Limits:      limits,
				Features: []*Feature{
					{
						DisplayName: "Virtual Cluster Management",
						Name:        "vclusters",
					},
					{
						DisplayName: "Sleep Mode for Virtual Clusters",
						Name:        "vcluster-sleep-mode",
					},
					{
						DisplayName: "Central HostPath Mapper",
						Name:        "vcluster-host-path-mapper",
					},
					{
						DisplayName: "Enterprise Plugins",
						Name:        "vcluster-enterprise-plugins",
					},
					{
						DisplayName: "Security-Hardened vCluster Image",
						Name:        "vcp-distro-image",
					},
					{
						DisplayName: "Built-In CoreDNS",
						Name:        "vcp-distro-built-in-coredns",
					},
					{
						DisplayName: "Virtual Admission Control",
						Name:        "vcp-distro-admission-control",
					},
					{
						DisplayName: "Sync Patches",
						Name:        "vcp-distro-sync-patches",
					},
					{
						DisplayName: "Embedded etcd",
						Name:        "vcp-distro-embedded-etcd",
					},
					{
						DisplayName: "Isolated Control Plane",
						Name:        "vcp-distro-isolated-cp",
					},
					{
						DisplayName: "Centralized Admission Control",
						Name:        "vcp-distro-centralized-admission-control",
					},
					{
						DisplayName: "Generic Sync",
						Name:        "vcp-distro-generic-sync",
					},
					{
						DisplayName: "Translate Patches",
						Name:        "vcp-distro-translate-patches",
					},
					{
						DisplayName: "KubeVirt Integration",
						Name:        "vcp-distro-integrations-kube-virt",
					},
					{
						DisplayName: "External Secrets Integration",
						Name:        "vcp-distro-integrations-external-secrets",
					},
					{
						DisplayName: "Cert Manager Integration",
						Name:        "vcp-distro-integrations-cert-manager",
					},
					{
						DisplayName: "FIPS",
						Name:        "vcp-distro-fips",
					},
					{
						DisplayName: "External Database",
						Name:        "vcp-distro-external-database",
					},
					{
						DisplayName: "Database Connector",
						Name:        "connector-external-database",
					},
					{
						DisplayName: "SleepMode",
						Name:        "vcp-distro-sleep-mode",
					},
					{
						DisplayName: "Dev Environment Management",
						Name:        "devpod",
					},
					{
						DisplayName: "Namespace Management",
						Name:        "namespaces",
					},
					{
						DisplayName: "Sleep Mode for Namespaces",
						Name:        "namespace-sleep-mode",
					},
					{
						DisplayName: "Connected Clusters",
						Name:        "connected-clusters",
					},
					{
						DisplayName: "Cluster Access",
						Name:        "cluster-access",
					},
					{
						DisplayName: "Cluster Role Management",
						Name:        "cluster-roles",
					},
					{
						DisplayName: "Single Sign-On",
						Name:        "sso-authentication",
					},
					{
						DisplayName: "Audit Logging",
						Name:        "audit-logging",
					},
					{
						DisplayName: "Automatic Auth For Ingresses",
						Name:        "auto-ingress-authentication",
					},
					{
						DisplayName: "Loft as OIDC Provider",
						Name:        "oidc-provider",
					},
					{
						DisplayName: "Multiple SSO Providers",
						Name:        "multiple-sso-providers",
					},
					{
						DisplayName: "Apps",
						Name:        "apps",
					},
					{
						DisplayName: "Template Versioning",
						Name:        "template-versioning",
					},
					{
						DisplayName: "Argo Integration",
						Name:        "argo-integration",
					},
					{
						DisplayName: "Rancher Integration",
						Name:        "rancher-integration",
					},
					{
						DisplayName: "Secrets Sync",
						Name:        "secrets",
					},
					{
						DisplayName: "Secrets Encryption",
						Name:        "secret-encryption",
					},
					{
						DisplayName: "HashiCorp Vault Integration",
						Name:        "vault-integration",
					},
					{
						DisplayName: "High-Availability Mode",
						Name:        "ha-mode",
					},
					{
						DisplayName: "Multi-Region Mode",
						Name:        "multi-region-mode",
					},
					{
						DisplayName: "Air-Gapped Mode",
						Name:        "air-gapped-mode",
					},
					{
						DisplayName: "Custom Branding",
						Name:        "custom-branding",
					},
					{
						DisplayName: "Advanced UI Customizations",
						Name:        "advanced-ui-customizations",
					},
					{
						DisplayName: "vNode Runtime",
						Name:        "vnode-runtime",
					},
					{
						DisplayName: "Project Quotas",
						Name:        "project-quotas",
					},
					{
						DisplayName: "Resolve DNS",
						Name:        "resolve-dns",
					},
				},
			},
		},
	}
}
