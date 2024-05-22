package licenseapi

type ProductName string
type ModuleName string
type PlanStatus string
type PlanInterval string
type TierMode string
type ResourceName string
type ResourceStatus string
type TrialStatus string
type FeatureStatus string
type FeatureName string
type ButtonName string

// Products
const (
	Loft        ProductName = "loft"
	VClusterPro ProductName = "vcluster-pro"
	DevPodPro   ProductName = "devpod-pro"
)

// Modules
const (
	KubernetesNamespaceModule ModuleName = "k8s-namespaces"
	KubernetesClusterModule   ModuleName = "k8s-clusters"
	VirtualClusterModule      ModuleName = "vclusters"
	VClusterProDistroModule   ModuleName = "vcluster-pro-distro"
	DevPodModule              ModuleName = "devpod"
	AuthModule                ModuleName = "auth"
	TemplatingModule          ModuleName = "templating"
	SecretsModule             ModuleName = "secrets"
	DeploymentModesModule     ModuleName = "deployment-modes"
	UIModule                  ModuleName = "ui"
)

// Plan Status
const (
	PlanStatusActive    PlanStatus = "active"
	PlanStatusTrialing  PlanStatus = "trialing"
	PlanStatusLegacy    PlanStatus = "legacy"
	PlanStatusAvailable PlanStatus = ""
)

// Plan Interval
const (
	PlanIntervalMonth PlanInterval = "month"
	PlanIntervalYear  PlanInterval = "year"
)

// Tier Modes
const (
	TierModeGraduated TierMode = "graduated"
	TierModeVolume    TierMode = "volume"
)

// Resources (e.g. for limits)
const (
	ConnectedClusterLimit        ResourceName = "connected-cluster"
	VirtualClusterInstanceLimit  ResourceName = "virtual-cluster-instance"
	SpaceInstanceLimit           ResourceName = "space-instance"
	DevPodWorkspaceInstanceLimit ResourceName = "devpod-workspace-instance"
	UserLimit                    ResourceName = "user"
	InstanceLimit                ResourceName = "instance"
)

// Resource Status
const (
	ResourceStatusActive       ResourceStatus = "active"
	ResourceStatusTotalCreated ResourceStatus = "created"
	ResourceStatusTotal        ResourceStatus = ""
)

// Trial Status
const (
	TrialStatusExpired TrialStatus = "expired"
	TrialStatusActive  TrialStatus = ""
)

// Buttons
const (
	ButtonContactSales  ButtonName = "contact-sales"
	ButtonManageBilling ButtonName = "manage-billing"
)

// Feature Status
const (
	FeatureStatusActive     FeatureStatus = "active"
	FeatureStatusPreview    FeatureStatus = "preview"
	FeatureStatusIncluded   FeatureStatus = "included"
	FeatureStatusHidden     FeatureStatus = "hidden"
	FeatureStatusDisallowed FeatureStatus = ""
)
