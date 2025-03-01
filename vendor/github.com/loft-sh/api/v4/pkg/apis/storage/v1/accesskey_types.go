package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessKey holds the session information
// +k8s:openapi-gen=true
type AccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessKeySpec   `json:"spec,omitempty"`
	Status AccessKeyStatus `json:"status,omitempty"`
}

type AccessKeySpec struct {
	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes an app
	// +optional
	Description string `json:"description,omitempty"`

	// The user this access key refers to
	// +optional
	User string `json:"user,omitempty"`

	// The team this access key refers to
	// +optional
	Team string `json:"team,omitempty"`

	// Subject is a generic subject that can be used
	// instead of user or team
	// +optional
	Subject string `json:"subject,omitempty"`

	// Groups specifies extra groups to apply when using
	// this access key
	// +optional
	Groups []string `json:"groups,omitempty"`

	// The actual access key that will be used as a bearer token
	// +optional
	Key string `json:"key,omitempty"`

	// If this field is true, the access key is still allowed to exist,
	// however will not work to access the api
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// The time to life for this access key
	// +optional
	TTL int64 `json:"ttl,omitempty"`

	// If this is specified, the time to life for this access key will
	// start after the lastActivity instead of creation timestamp
	// +optional
	TTLAfterLastActivity bool `json:"ttlAfterLastActivity,omitempty"`

	// Scope defines the scope of the access key.
	// +optional
	Scope *AccessKeyScope `json:"scope,omitempty"`

	// The type of an access key, which basically describes if the access
	// key is user managed or managed by loft itself.
	// +optional
	Type AccessKeyType `json:"type,omitempty"`

	// If available, contains information about the sso login data for this
	// access key
	// +optional
	Identity *AccessKeyIdentity `json:"identity,omitempty"`

	// The last time the identity was refreshed
	// +optional
	IdentityRefresh *metav1.Time `json:"identityRefresh,omitempty"`

	// If the token is a refresh token, contains information about it
	// +optional
	OIDCProvider *AccessKeyOIDCProvider `json:"oidcProvider,omitempty"`

	// DEPRECATED: do not use anymore
	// Parent is used to share OIDC and external token information
	// with multiple access keys. Since copying an OIDC refresh token
	// would result in the other access keys becoming invalid after a refresh
	// parent allows access keys to share that information.
	//
	// The use case for this is primarily user generated access keys,
	// which will have the users current access key as parent if it contains
	// an OIDC token.
	// +optional
	Parent string `json:"parent,omitempty"`

	// DEPRECATED: Use identity instead
	// If available, contains information about the oidc login data for this
	// access key
	// +optional
	OIDCLogin *AccessKeyOIDC `json:"oidcLogin,omitempty"`
}

type AccessKeyScope struct {
	// Roles is a set of managed permissions to apply to the access key.
	// +optional
	Roles []AccessKeyScopeRole `json:"roles,omitempty"`

	// Projects specifies the projects the access key should have access to.
	// +optional
	Projects []AccessKeyScopeProject `json:"projects,omitempty"`

	// Spaces specifies the spaces the access key is allowed to access.
	// +optional
	Spaces []AccessKeyScopeSpace `json:"spaces,omitempty"`

	// VirtualClusters specifies the virtual clusters the access key is allowed to access.
	// +optional
	VirtualClusters []AccessKeyScopeVirtualCluster `json:"virtualClusters,omitempty"`

	// Clusters specifies the project cluster the access key is allowed to access.
	// +optional
	Clusters []AccessKeyScopeCluster `json:"clusters,omitempty"`

	// DEPRECATED: Use Projects, Spaces and VirtualClusters instead
	// Rules specifies the rules that should apply to the access key.
	// +optional
	Rules []AccessKeyScopeRule `json:"rules,omitempty"`

	// AllowLoftCLI allows certain read-only management requests to
	// make sure loft cli works correctly with this specific access key.
	//
	// Deprecated: Use the `roles` field instead
	//  ```yaml
	//  # Example:
	//  roles:
	//    - role: loftCLI
	//  ```
	// +optional
	AllowLoftCLI bool `json:"allowLoftCli,omitempty"`
}

func (a AccessKeyScope) ContainsRole(val AccessKeyScopeRoleName) bool {
	if a.AllowLoftCLI && val == AccessKeyScopeRoleLoftCLI {
		return true
	}

	for _, entry := range a.Roles {
		if entry.Role == val {
			return true
		}

		// (ThomasK33): Add implicit network peer permissions
		if val == AccessKeyScopeRoleNetworkPeer {
			switch entry.Role {
			case AccessKeyScopeRoleVCluster, AccessKeyScopeRoleAgent, AccessKeyScopeRoleRunner:
				return true
			// (ThomasK33): Adding this so that the exhaustive linter is happy
			case AccessKeyScopeRoleNetworkPeer:
				return true
			case AccessKeyScopeRoleLoftCLI:
				return false
			}
		}
	}

	return false
}

func (a AccessKeyScope) GetRole(name AccessKeyScopeRoleName) AccessKeyScopeRole {
	for _, entry := range a.Roles {
		if entry.Role == name {
			return entry
		}
	}

	if a.ContainsRole(name) {
		return AccessKeyScopeRole{
			Role: name,
		}
	}

	return AccessKeyScopeRole{}
}

type AccessKeyScopeRole struct {
	// Role is the name of the role to apply to the access key scope.
	// +optional
	Role AccessKeyScopeRoleName `json:"role,omitempty"`

	// Projects specifies the projects the access key should have access to.
	// +optional
	Projects []string `json:"projects,omitempty"`

	// VirtualClusters specifies the virtual clusters the access key is allowed to access.
	// +optional
	VirtualClusters []string `json:"virtualClusters,omitempty"`
}

// AccessKeyScopeRoleName is the role name for a given scope
// +enum
type AccessKeyScopeRoleName string

const (
	AccessKeyScopeRoleAgent       AccessKeyScopeRoleName = "agent"
	AccessKeyScopeRoleVCluster    AccessKeyScopeRoleName = "vcluster"
	AccessKeyScopeRoleNetworkPeer AccessKeyScopeRoleName = "network-peer"
	AccessKeyScopeRoleLoftCLI     AccessKeyScopeRoleName = "loft-cli"
	AccessKeyScopeRoleRunner      AccessKeyScopeRoleName = "runner"
	AccessKeyScopeRoleWorkspace   AccessKeyScopeRoleName = "workspace"
)

type AccessKeyScopeCluster struct {
	// Cluster is the name of the cluster to access. You can specify * to select all clusters.
	// +optional
	Cluster string `json:"cluster,omitempty"`
}

type AccessKeyScopeVirtualCluster struct {
	// Project is the name of the project.
	// +optional
	Project string `json:"project,omitempty"`

	// VirtualCluster is the name of the virtual cluster to access. You can specify * to select all virtual clusters.
	// +optional
	VirtualCluster string `json:"virtualCluster,omitempty"`
}

type AccessKeyScopeSpace struct {
	// Project is the name of the project.
	// +optional
	Project string `json:"project,omitempty"`

	// Space is the name of the space. You can specify * to select all spaces.
	// +optional
	Space string `json:"space,omitempty"`
}

type AccessKeyScopeProject struct {
	// Project is the name of the project. You can specify * to select all projects.
	// +optional
	Project string `json:"project,omitempty"`
}

// AccessKeyScopeRule describes a rule for the access key
type AccessKeyScopeRule struct {
	// The verbs that match this rule.
	// An empty list implies every verb.
	// +optional
	Verbs []string `json:"verbs,omitempty"`

	// Rules can apply to API resources (such as "pods" or "secrets"),
	// non-resource URL paths (such as "/api"), or neither, but not both.
	// If neither is specified, the rule is treated as a default for all URLs.

	// Resources that this rule matches. An empty list implies all kinds in all API groups.
	// +optional
	Resources []GroupResources `json:"resources,omitempty"`

	// Namespaces that this rule matches.
	// The empty string "" matches non-namespaced resources.
	// An empty list implies every namespace.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// NonResourceURLs is a set of URL paths that should be checked.
	// *s are allowed, but only as the full, final step in the path.
	// Examples:
	//  "/metrics" - Log requests for apiserver metrics
	//  "/healthz*" - Log all health checks
	// +optional
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`

	// RequestTargets is a list of request targets that are allowed.
	// An empty list implies every request.
	// +optional
	RequestTargets []RequestTarget `json:"requestTargets,omitempty"`

	// Cluster that this rule matches. Only applies to cluster requests.
	// If this is set, no requests for non cluster requests are allowed.
	// An empty cluster means no restrictions will apply.
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// VirtualClusters that this rule matches. Only applies to virtual cluster requests.
	// An empty list means no restrictions will apply.
	// +optional
	VirtualClusters []AccessKeyVirtualCluster `json:"virtualClusters,omitempty"`
}

type AccessKeyVirtualCluster struct {
	// Name of the virtual cluster. Empty means all virtual clusters.
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace of the virtual cluster. Empty means all namespaces.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// RequestTarget defines the target of an incoming request
type RequestTarget string

// Valid request targets
const (
	// RequestTargetManagement specifies a loft management api request
	RequestTargetManagement RequestTarget = "Management"
	// RequestTargetCluster specifies a connected kubernetes cluster request
	RequestTargetCluster RequestTarget = "Cluster"
	// RequestTargetProjectSpace specifies a project space cluster request
	RequestTargetProjectSpace RequestTarget = "ProjectSpace"
	// RequestTargetProjectVirtualCluster specifies a project virtual kubernetes cluster request
	RequestTargetProjectVirtualCluster RequestTarget = "ProjectVirtualCluster"
)

// GroupResources represents resource kinds in an API group.
type GroupResources struct {
	// Group is the name of the API group that contains the resources.
	// The empty string represents the core API group.
	// +optional
	Group string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	// Resources is a list of resources this rule applies to.
	//
	// For example:
	// 'pods' matches pods.
	// 'pods/log' matches the log subresource of pods.
	// '*' matches all resources and their subresources.
	// 'pods/*' matches all subresources of pods.
	// '*/scale' matches all scale subresources.
	//
	// If wildcard is present, the validation rule will ensure resources do not
	// overlap with each other.
	//
	// An empty list implies all resources and subresources in this API groups apply.
	// +optional
	Resources []string `json:"resources,omitempty" protobuf:"bytes,2,rep,name=resources"`
	// ResourceNames is a list of resource instance names that the policy matches.
	// Using this field requires Resources to be specified.
	// An empty list implies that every instance of the resource is matched.
	// +optional
	ResourceNames []string `json:"resourceNames,omitempty" protobuf:"bytes,3,rep,name=resourceNames"`
}

type AccessKeyIdentity struct {
	// The subject of the user
	// +optional
	UserID string `json:"userId,omitempty"`

	// The username
	// +optional
	Username string `json:"username,omitempty"`

	// The preferred username / display name
	// +optional
	PreferredUsername string `json:"preferredUsername,omitempty"`

	// The user email
	// +optional
	Email string `json:"email,omitempty"`

	// If the user email was verified
	// +optional
	EmailVerified bool `json:"emailVerified,omitempty"`

	// The groups from the identity provider
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Connector is the name of the connector this access key was created from
	// +optional
	Connector string `json:"connector,omitempty"`

	// ConnectorData holds data used by the connector for subsequent requests after initial
	// authentication, such as access tokens for upstream providers.
	//
	// This data is never shared with end users, OAuth clients, or through the API.
	// +optional
	ConnectorData []byte `json:"connectorData,omitempty"`
}

type AccessKeyOIDCProvider struct {
	// ClientId the token was generated for
	// +optional
	ClientId string `json:"clientId,omitempty"`

	// Nonce to use
	// +optional
	Nonce string `json:"nonce,omitempty"`

	// RedirectUri to use
	// +optional
	RedirectUri string `json:"redirectUri,omitempty"`

	// Scopes to use
	// +optional
	Scopes string `json:"scopes,omitempty"`
}

type AccessKeyOIDC struct {
	// The current id token that was created during login
	// +optional
	IDToken []byte `json:"idToken,omitempty"`

	// The current access token that was created during login
	// +optional
	AccessToken []byte `json:"accessToken,omitempty"`

	// The current refresh token that was created during login
	// +optional
	RefreshToken []byte `json:"refreshToken,omitempty"`

	// The last time the id token was refreshed
	// +optional
	LastRefresh *metav1.Time `json:"lastRefresh,omitempty"`
}

// AccessKeyType describes the type of an access key
type AccessKeyType string

// These are the valid access key types
const (
	AccessKeyTypeNone             AccessKeyType = ""
	AccessKeyTypeLogin            AccessKeyType = "Login"
	AccessKeyTypeUser             AccessKeyType = "User"
	AccessKeyTypeOther            AccessKeyType = "Other"
	AccessKeyTypeReset            AccessKeyType = "Reset"
	AccessKeyTypeOIDCRefreshToken AccessKeyType = "OIDCRefreshToken"
	AccessKeyTypeNetworkPeer      AccessKeyType = "NetworkPeer"
	AccessKeyTypeWorkspace        AccessKeyType = "Workspace"
)

// AccessKeyStatus holds the status of an access key
type AccessKeyStatus struct {
	// The last time this access key was used to access the api
	// +optional
	LastActivity *metav1.Time `json:"lastActivity,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessKeyList contains a list of AccessKey
type AccessKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AccessKey{}, &AccessKeyList{})
}
