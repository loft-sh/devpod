package auth

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const GroupVersion = "authentication.loft.sh/v1"

// OIDCTokenRequest is used by the /auth/oidc/token route
type OIDCTokenRequest struct {
	Token       string `json:"token,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
}

// OIDCRefreshRequest is used by the /auth/oidc/refresh route
type OIDCRefreshRequest struct {
	RefreshToken string `json:"refreshToken,omitempty"`
}

// TokenRequest is used by the /auth/token route
type TokenRequest struct {
	Key string `json:"key,omitempty"`
}

// PasswordLoginRequest is used by the /auth/password/login route
type PasswordLoginRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Token string `json:"token"`
}

type AccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	User      string `json:"user"`
	Username  string `json:"username"`
	AccessKey string `json:"accessKey"`
}

type OIDCRedirect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Redirect string `json:"redirect,omitempty"`

	// SAML specific values
	SamlID   string `json:"samlId,omitempty"`
	SamlData string `json:"samlData,omitempty"`
}

type OIDCToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	IDToken      string `json:"idToken"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type Info struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Methods InfoMethods `json:"methods,omitempty"`
}

type InfoMethods struct {
	SSO      []*MethodSSO    `json:"sso,omitempty"`
	Rancher  *MethodRancher  `json:"rancher,omitempty"`
	Password *MethodPassword `json:"password,omitempty"`
}

type MethodSSO struct {
	// ID of the SSO to show in the UI. Only for displaying purposes
	ID string `json:"id,omitempty"`

	// DisplayName of the SSO to show in the UI. Only for displaying purposes
	DisplayName string `json:"displayName,omitempty"`

	// LoginEndpoint is the path the UI will request a login url from
	LoginEndpoint string `json:"loginEndpoint,omitempty"`

	// LogoutEndpoint is the path the UI will request a logout url from
	LogoutEndpoint string `json:"logoutEndpoint,omitempty"`
}

type MethodPassword struct {
	// Indicates if the authentication method is enabled
	Enabled bool `json:"enabled,omitempty"`
}

type MethodRancher struct {
	// Indicates if the authentication method is enabled
	Enabled bool `json:"enabled,omitempty"`

	// Host is the rancher host to use for redirects
	Host string `json:"host,omitempty"`
}

type Version struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Version string `json:"version"`
	Major   string `json:"major,omitempty"`
	Minor   string `json:"minor,omitempty"`

	KubeVersion   string `json:"kubeVersion,omitempty"`
	DevPodVersion string `json:"devPodVersion,omitempty"`

	NewerVersion  string `json:"newerVersion,omitempty"`
	ShouldUpgrade bool   `json:"shouldUpgrade,omitempty"`
}
