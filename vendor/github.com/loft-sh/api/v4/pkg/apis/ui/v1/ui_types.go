package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ProductName string

const (
	ProductNameLoft        ProductName = "Loft"
	ProductNameVClusterPro ProductName = "vCluster Platform"
	ProductNameDevPodPro   ProductName = "DevPod.Pro"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UISettings holds the loft ui configuration settings
// +k8s:openapi-gen=true
type UISettings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UISettingsSpec   `json:"spec,omitempty"`
	Status UISettingsStatus `json:"status,omitempty"`
}

// UISettingsSpec holds the specification
type UISettingsSpec struct {
	UISettingsConfig `json:",inline"`

	// Name is the name of the product
	// +optional
	ProductName string `json:"productName,omitempty"`

	// Offline is true if loft is running in an airgapped environment
	// +optional
	Offline bool `json:"offline,omitempty"`

	// HasHelmRelease indicates whether loft has been installed via Helm
	HasHelmRelease bool `json:"hasHelmRelease,omitempty"`

	// DefaultVClusterVersion is the default version of vClusters
	DefaultVClusterVersion string `json:"defaultVClusterVersion,omitempty"`
}

type UISettingsConfig struct {
	// LoftVersion holds the current loft version
	// +optional
	LoftVersion string `json:"loftVersion,omitempty"`
	// LogoURL is url pointing to the logo to use in the Loft UI. This path must be accessible for clients accessing
	// the Loft UI!
	// +optional
	LogoURL string `json:"logoURL,omitempty"`
	// LogoBackgroundColor is the color value (ex: "#12345") to use as the background color for the logo
	// +optional
	LogoBackgroundColor string `json:"logoBackgroundColor,omitempty"`
	// LegalTemplate is a text (html) string containing the legal template to prompt to users when authenticating to Loft
	// +optional
	LegalTemplate string `json:"legalTemplate,omitempty"`
	// PrimaryColor is the color value (ex: "#12345") to use as the primary color
	// +optional
	PrimaryColor string `json:"primaryColor,omitempty"`
	// SidebarColor is the color value (ex: "#12345") to use for the sidebar
	// +optional
	SidebarColor string `json:"sidebarColor,omitempty"`
	// AccentColor is the color value (ex: "#12345") to use for the accent
	// +optional
	AccentColor string `json:"accentColor,omitempty"`
	// CustomCSS holds URLs with custom css files that should be included when loading the UI
	// +optional
	CustomCSS []string `json:"customCss,omitempty"`
	// CustomJavaScript holds URLs with custom js files that should be included when loading the UI
	// +optional
	CustomJavaScript []string `json:"customJavaScript,omitempty"`
	// NavBarButtons holds extra nav bar buttons
	// +optional
	NavBarButtons []NavBarButton `json:"navBarButtons,omitempty"`
}

type NavBarButton struct {
	// Position holds the position of the button, can be one of:
	// TopStart, TopEnd, BottomStart, BottomEnd. Defaults to BottomEnd
	// +optional
	Position string `json:"position,omitempty"`
	// Text holds text for the button
	// +optional
	Text string `json:"text,omitempty"`
	// Link holds the link of the navbar button
	// +optional
	Link string `json:"link,omitempty"`
	// Icon holds the url of the icon to display
	// +optional
	Icon string `json:"icon,omitempty"`
}

// UISettingsStatus holds the status
type UISettingsStatus struct{}
