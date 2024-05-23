package licenseapi

var Limits = map[ResourceName]*Limit{
	ConnectedClusterLimit: {
		DisplayName: "Connected Clusters",
		Name:        string(ConnectedClusterLimit),
	},
	VirtualClusterInstanceLimit: {
		DisplayName: "Virtual Clusters",
		Name:        string(VirtualClusterInstanceLimit),
	},
	DevPodWorkspaceInstanceLimit: {
		DisplayName: "Dev Environments",
		Name:        string(DevPodWorkspaceInstanceLimit),
	},
	UserLimit: {
		DisplayName: "Users",
		Name:        string(UserLimit),
	},
	InstanceLimit: {
		DisplayName: "Instances",
		Name:        string(InstanceLimit),
	},
}

func New(product ProductName) *License {
	allowedStatus := string(FeatureStatusActive)

	connectedClusterStatus := string(FeatureStatusActive)
	if product != VClusterPro && product != Loft {
		connectedClusterStatus = string(FeatureStatusDisallowed)
	}

	namespaceStatus := string(FeatureStatusActive)
	if product != Loft {
		namespaceStatus = string(FeatureStatusDisallowed)
	}

	virtualClusterStatus := string(FeatureStatusActive)
	if product != VClusterPro && product != Loft {
		virtualClusterStatus = string(FeatureStatusDisallowed)
	}

	devpodStatus := string(FeatureStatusActive)
	if product != DevPodPro {
		devpodStatus = string(FeatureStatusDisallowed)
	}

	return &License{
		Modules: []*Module{
			{
				DisplayName: "Virtual Clusters",
				Name:        string(VirtualClusterModule),
				Limits: []*Limit{
					Limits[VirtualClusterInstanceLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "Virtual Cluster Management",
						Name:        string(VirtualCluster),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Sleep Mode for Virtual Clusters",
						Name:        string(VirtualClusterSleepMode),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Central HostPath Mapper",
						Name:        string(VirtualClusterCentralHostPathMapper),
						Status:      virtualClusterStatus,
					},
				},
			},
			{
				DisplayName: "vCluster.Pro Distro",
				Name:        string(VClusterProDistroModule),
				Features: []*Feature{
					{
						DisplayName: "Security-Hardened vCluster Image",
						Name:        string(VirtualClusterProDistroImage),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Built-In CoreDNS",
						Name:        string(VirtualClusterProDistroBuiltInCoreDNS),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Virtual Admission Control",
						Name:        string(VirtualClusterProDistroAdmissionControl),
						Status:      string(FeatureStatusHidden),
					},
					{
						DisplayName: "Sync Patches",
						Name:        string(VirtualClusterProDistroSyncPatches),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Isolated Control Plane",
						Name:        string(VirtualClusterProDistroIsolatedControlPlane),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Centralized Admission Control",
						Name:        string(VirtualClusterProDistroCentralizedAdmissionControl),
						Status:      virtualClusterStatus,
					},
				},
			},
			{
				DisplayName: "Dev Environments",
				Name:        string(DevPodModule),
				Limits: []*Limit{
					Limits[DevPodWorkspaceInstanceLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "Dev Environment Management",
						Name:        string(DevPod),
						Status:      devpodStatus,
					},
				},
			},
			{
				DisplayName: "Kubernetes Namespaces",
				Name:        string(KubernetesNamespaceModule),
				Features: []*Feature{
					{
						DisplayName: "Namespace Management",
						Name:        string(Namespace),
						Status:      namespaceStatus,
					},
					{
						DisplayName: "Sleep Mode for Namespaces",
						Name:        string(NamespaceSleepMode),
						Status:      namespaceStatus,
					},
				},
			},
			{
				DisplayName: "Kubernetes Clusters",
				Name:        string(KubernetesClusterModule),
				Limits: []*Limit{
					Limits[ConnectedClusterLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "Connected Clusters",
						Name:        string(ConnectedClusters),
						Status:      connectedClusterStatus,
					},
					{
						DisplayName: "Cluster Access",
						Name:        string(ClusterAccess),
						Status:      connectedClusterStatus,
					},
					{
						DisplayName: "Cluster Role Management",
						Name:        string(ClusterRoles),
						Status:      connectedClusterStatus,
					},
				},
			},
			{
				DisplayName: "Authentication & Audit Logging",
				Name:        string(AuthModule),
				Limits: []*Limit{
					Limits[UserLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "Single Sign-On",
						Name:        string(SSOAuth),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Audit Logging",
						Name:        string(AuditLogging),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Automatic Auth For Ingresses",
						Name:        string(AutomaticIngressAuth),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Loft as OIDC Provider",
						Name:        string(OIDCProvider),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Multiple SSO Providers",
						Name:        string(MultipleSSOProviders),
						Status:      allowedStatus,
					},
				},
			},
			{
				DisplayName: "Templating & GitOps",
				Name:        string(TemplatingModule),
				Features: []*Feature{
					{
						DisplayName: "Apps",
						Name:        string(Apps),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Template Versioning",
						Name:        string(TemplateVersioning),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Argo Integration",
						Name:        string(ArgoIntegration),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Rancher Integration",
						Name:        string(RancherIntegration),
						Status:      allowedStatus,
					},
				},
			},
			{
				DisplayName: "Secrets Management",
				Name:        string(SecretsModule),
				Features: []*Feature{
					{
						DisplayName: "Secrets Sync",
						Name:        string(Secrets),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Secrets Encryption",
						Name:        string(SecretEncryption),
						Status:      allowedStatus,
					},
					{
						DisplayName: "HashiCorp Vault Integration",
						Name:        string(VaultIntegration),
						Status:      allowedStatus,
					},
				},
			},
			{
				DisplayName: "Deployment Modes",
				Name:        string(DeploymentModesModule),
				Limits: []*Limit{
					Limits[InstanceLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "High-Availability Mode",
						Name:        string(HighAvailabilityMode),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Multi-Region Mode",
						Name:        string(MultiRegionMode),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Air-Gapped Mode",
						Name:        string(AirGappedMode),
						Status:      allowedStatus,
					},
				},
			},
			{
				DisplayName: "UI Customization",
				Name:        string(UIModule),
				Features: []*Feature{
					{
						DisplayName: "Custom Branding",
						Name:        string(CustomBranding),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Advanced UI Customizations",
						Name:        string(AdvancedUICustomizations),
						Status:      allowedStatus,
					},
				},
			},
		},
	}
}
