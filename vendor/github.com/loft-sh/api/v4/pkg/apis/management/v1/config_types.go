package v1

import (
	auditv1 "github.com/loft-sh/api/v4/pkg/apis/audit/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	uiv1 "github.com/loft-sh/api/v4/pkg/apis/ui/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Config holds the loft configuration
// +k8s:openapi-gen=true
// +resource:path=configs,rest=ConfigREST
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

// ConfigSpec holds the specification
type ConfigSpec struct {
	// Raw holds the raw config
	// +optional
	Raw []byte `json:"raw,omitempty"`
}

// ConfigStatus holds the status, which is the parsed raw config
type ConfigStatus struct {
	// Authentication holds the information for authentication
	// +optional
	Authentication Authentication `json:"auth,omitempty"`

	// DEPRECATED: Configure the OIDC clients using either the OIDC Client UI or a secret. By default, vCluster Platform as an OIDC Provider is enabled but does not function without OIDC clients.
	// +optional
	OIDC *OIDC `json:"oidc,omitempty"`

	// Apps holds configuration around apps
	// +optional
	Apps *Apps `json:"apps,omitempty"`

	// Audit holds audit configuration
	// +optional
	Audit *Audit `json:"audit,omitempty"`

	// LoftHost holds the domain where the loft instance is hosted. This should not include https or http. E.g. loft.my-domain.com
	// +optional
	LoftHost string `json:"loftHost,omitempty"`

	// ProjectNamespacePrefix holds the prefix for loft project namespaces. Omitted defaults to "p-"
	// +optional
	ProjectNamespacePrefix *string `json:"projectNamespacePrefix,omitempty"`

	// DevPodSubDomain holds a subdomain in the following form *.workspace.my-domain.com
	// +optional
	DevPodSubDomain string `json:"devPodSubDomain,omitempty"`

	// UISettings holds the settings for modifying the Loft user interface
	// +optional
	UISettings *uiv1.UISettingsConfig `json:"uiSettings,omitempty"`

	// VaultIntegration holds the vault integration configuration
	// +optional
	VaultIntegration *storagev1.VaultIntegrationSpec `json:"vault,omitempty"`

	// DisableLoftConfigEndpoint will disable setting config via the UI and config.management.loft.sh endpoint
	DisableConfigEndpoint bool `json:"disableConfigEndpoint,omitempty"`

	// Cloud holds the settings to be used exclusively in vCluster Cloud based
	// environments and deployments.
	Cloud *Cloud `json:"cloud,omitempty"`

	// CostControl holds the settings related to the Cost Control ROI dashboard and its metrics gathering infrastructure
	CostControl *CostControl `json:"costControl,omitempty"`

	// ImageBuilder holds the settings related to the image builder
	ImageBuilder *ImageBuilder `json:"imageBuilder,omitempty"`
}

// Audit holds the audit configuration options for loft. Changing any options will require a loft restart
// to take effect.
type Audit struct {
	// If audit is enabled and incoming api requests will be logged based on the supplied policy.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// If true, the agent will not send back any audit logs to Loft itself.
	// +optional
	DisableAgentSyncBack bool `json:"disableAgentSyncBack,omitempty"`

	// Level is an optional log level for audit logs. Cannot be used together with policy
	// +optional
	Level int `json:"level,omitempty"`

	// The audit policy to use and log requests. By default loft will not log anything
	// +optional
	Policy AuditPolicy `json:"policy,omitempty"`

	// DataStoreEndpoint is an endpoint to store events in.
	// +optional
	DataStoreEndpoint string `json:"dataStoreEndpoint,omitempty"`

	// DataStoreMaxAge is the maximum number of hours to retain old log events in the datastore
	// +optional
	DataStoreMaxAge *int `json:"dataStoreTTL,omitempty"`

	// The path where to save the audit log files. This is required if audit is enabled. Backup log files will
	// be retained in the same directory.
	// +optional
	Path string `json:"path,omitempty"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	// +optional
	MaxAge int `json:"maxAge,omitempty"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	// +optional
	MaxBackups int `json:"maxBackups,omitempty"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	// +optional
	MaxSize int `json:"maxSize,omitempty"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	// +optional
	Compress bool `json:"compress,omitempty"`
}

// AuditPolicy describes the audit policy to use for loft
type AuditPolicy struct {
	// Rules specify the audit Level a request should be recorded at.
	// A request may match multiple rules, in which case the FIRST matching rule is used.
	// The default audit level is None, but can be overridden by a catch-all rule at the end of the list.
	// PolicyRules are strictly ordered.
	Rules []AuditPolicyRule `json:"rules,omitempty"`

	// OmitStages is a list of stages for which no events are created. Note that this can also
	// be specified per rule in which case the union of both are omitted.
	// +optional
	OmitStages []auditv1.Stage `json:"omitStages,omitempty"`
}

// AuditPolicyRule describes a policy for auditing
type AuditPolicyRule struct {
	// The Level that requests matching this rule are recorded at.
	Level auditv1.Level `json:"level"`

	// The users (by authenticated user name) this rule applies to.
	// An empty list implies every user.
	// +optional
	Users []string `json:"users,omitempty"`
	// The user groups this rule applies to. A user is considered matching
	// if it is a member of any of the UserGroups.
	// An empty list implies every user group.
	// +optional
	UserGroups []string `json:"userGroups,omitempty"`

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

	// NonResourceURLs is a set of URL paths that should be audited.
	// *s are allowed, but only as the full, final step in the path.
	// Examples:
	//  "/metrics" - Log requests for apiserver metrics
	//  "/healthz*" - Log all health checks
	// +optional
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`

	// OmitStages is a list of stages for which no events are created. Note that this can also
	// be specified policy wide in which case the union of both are omitted.
	// An empty list means no restrictions will apply.
	// +optional
	OmitStages []auditv1.Stage `json:"omitStages,omitempty" protobuf:"bytes,8,rep,name=omitStages"`

	// RequestTargets is a list of request targets for which events are created.
	// An empty list implies every request.
	// +optional
	RequestTargets []auditv1.RequestTarget `json:"requestTargets,omitempty"`

	// Clusters that this rule matches. Only applies to cluster requests.
	// If this is set, no events for non cluster requests will be created.
	// An empty list means no restrictions will apply.
	// +optional
	Clusters []string `json:"clusters,omitempty"`
}

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

// Apps holds configuration for apps that should be shown
type Apps struct {
	// If this option is true, loft will not try to parse the default apps
	// +optional
	NoDefault bool `json:"noDefault,omitempty"`

	// These are additional repositories that are parsed by loft
	// +optional
	Repositories []storagev1.HelmChartRepository `json:"repositories,omitempty"`

	// Predefined apps that can be selected in the Spaces > Space menu
	// +optional
	PredefinedApps []PredefinedApp `json:"predefinedApps,omitempty"`
}

// PredefinedApp holds information about a predefined app
type PredefinedApp struct {
	// Chart holds the repo/chart name of the predefined app
	// +optional
	Chart string `json:"chart"`

	// InitialVersion holds the initial version of this app.
	// This version will be selected automatically.
	// +optional
	InitialVersion string `json:"initialVersion,omitempty"`

	// InitialValues holds the initial values for this app.
	// The values will be prefilled automatically. There are certain
	// placeholders that can be used within the values that are replaced
	// by the loft UI automatically.
	// +optional
	InitialValues string `json:"initialValues,omitempty"`

	// Holds the cluster names where to display this app
	// +optional
	Clusters []string `json:"clusters,omitempty"`

	// Title is the name that should be displayed for the predefined app.
	// If empty the chart name is used.
	// +optional
	Title string `json:"title,omitempty"`

	// IconURL specifies an url to the icon that should be displayed for this app.
	// If none is specified the icon from the chart metadata is used.
	// +optional
	IconURL string `json:"iconUrl,omitempty"`

	// ReadmeURL specifies an url to the readme page of this predefined app. If empty
	// an url will be constructed to artifact hub.
	// +optional
	ReadmeURL string `json:"readmeUrl,omitempty"`
}

// OIDC holds oidc provider relevant information
type OIDC struct {
	// If true indicates that loft will act as an OIDC server
	Enabled bool `json:"enabled,omitempty"`

	// If true indicates that loft will allow wildcard '*' in client redirectURIs
	WildcardRedirect bool `json:"wildcardRedirect,omitempty"`

	// The clients that are allowed to request loft tokens
	Clients []OIDCClientSpec `json:"clients,omitempty"`
}

// Authentication holds authentication relevant information
type Authentication struct {
	Connector `json:",inline"`

	// Rancher holds the rancher authentication options
	// +optional
	Rancher *AuthenticationRancher `json:"rancher,omitempty"`

	// Password holds password authentication relevant information
	// +optional
	Password *AuthenticationPassword `json:"password,omitempty"`

	// Connectors are optional additional connectors for Loft.
	// +optional
	Connectors []ConnectorWithName `json:"connectors,omitempty"`

	// Prevents from team creation for the new groups associated with the user at the time of logging in through sso,
	// Default behaviour is false, this means that teams will be created for new groups.
	// +optional
	DisableTeamCreation bool `json:"disableTeamCreation,omitempty"`

	// DisableUserCreation prevents the SSO connectors from creating a new user on a users initial signin through sso.
	// Default behaviour is false, this means that a new user object will be created once a user without
	// a Kubernetes user object logs in.
	// +optional
	DisableUserCreation bool `json:"disableUserCreation,omitempty"`

	// AccessKeyMaxTTLSeconds is the global maximum lifespan of an accesskey in seconds.
	// Leaving it 0 or unspecified will disable it.
	// Specifying 2592000 will mean all keys have a Time-To-Live of 30 days.
	// +optional
	AccessKeyMaxTTLSeconds int64 `json:"accessKeyMaxTTLSeconds,omitempty"`

	// LoginAccessKeyTTLSeconds is the time in seconds an access key is kept
	// until it is deleted.
	// Leaving it unspecified will default to 20 days.
	// Setting it to zero will disable the ttl.
	// Specifying 2592000 will mean all keys have a  default Time-To-Live of 30 days.
	// +optional
	LoginAccessKeyTTLSeconds *int64 `json:"loginAccessKeyTTLSeconds,omitempty"`

	// CustomHttpHeaders are additional headers that should be set for the authentication endpoints
	// +optional
	CustomHttpHeaders map[string]string `json:"customHttpHeaders,omitempty"`

	// GroupsFilters is a regex expression to only save matching sso groups into the user resource
	GroupsFilters []string `json:"groupsFilters,omitempty"`
}

type AuthenticationRancher struct {
	// Host holds the rancher host, e.g. my-domain.com
	// +optional
	Host string `json:"host,omitempty"`

	// BearerToken holds the rancher API key in token username and password form. E.g. my-token:my-secret
	// +optional
	BearerToken string `json:"bearerToken,omitempty"`

	// Insecure tells Loft if the Rancher endpoint is insecure.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

type ConnectorWithName struct {
	// ID is the id that should show up in the url
	// +optional
	ID string `json:"id,omitempty"`

	// DisplayName is the name that should show up in the ui
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	Connector `json:",inline"`
}

type Connector struct {
	// OIDC holds oidc authentication configuration
	// +optional
	OIDC *AuthenticationOIDC `json:"oidc,omitempty"`

	// Github holds github authentication configuration
	// +optional
	Github *AuthenticationGithub `json:"github,omitempty"`

	// Gitlab holds gitlab authentication configuration
	// +optional
	Gitlab *AuthenticationGitlab `json:"gitlab,omitempty"`

	// Google holds google authentication configuration
	// +optional
	Google *AuthenticationGoogle `json:"google,omitempty"`

	// Microsoft holds microsoft authentication configuration
	// +optional
	Microsoft *AuthenticationMicrosoft `json:"microsoft,omitempty"`

	// SAML holds saml authentication configuration
	// +optional
	SAML *AuthenticationSAML `json:"saml,omitempty"`
}

type AuthenticationSAML struct {
	// If the response assertion status value contains a Destination element, it
	// must match this value exactly.
	// Usually looks like https://your-loft-domain/auth/saml/callback
	RedirectURI string `json:"redirectURI,omitempty"`
	// SSO URL used for POST value.
	SSOURL string `json:"ssoURL,omitempty"`
	// CAData is a base64 encoded string that holds the ca certificate for validating the signature of the SAML response.
	// Either CAData, CA or InsecureSkipSignatureValidation needs to be defined.
	// +optional
	CAData []byte `json:"caData,omitempty"`

	// Name of attribute in the returned assertions to map to username
	UsernameAttr string `json:"usernameAttr,omitempty"`
	// Name of attribute in the returned assertions to map to email
	EmailAttr string `json:"emailAttr,omitempty"`
	// Name of attribute in the returned assertions to map to groups
	// +optional
	GroupsAttr string `json:"groupsAttr,omitempty"`

	// CA to use when validating the signature of the SAML response.
	// +optional
	CA string `json:"ca,omitempty"`
	// Ignore the ca cert
	// +optional
	InsecureSkipSignatureValidation bool `json:"insecureSkipSignatureValidation,omitempty"`

	// When provided Loft will include this as the Issuer value during AuthnRequest.
	// It will also override the redirectURI as the required audience when evaluating
	// AudienceRestriction elements in the response.
	// +optional
	EntityIssuer string `json:"entityIssuer,omitempty"`
	// Issuer value expected in the SAML response. Optional.
	// +optional
	SSOIssuer string `json:"ssoIssuer,omitempty"`

	// If GroupsDelim is supplied the connector assumes groups are returned as a
	// single string instead of multiple attribute values. This delimiter will be
	// used split the groups string.
	// +optional
	GroupsDelim string `json:"groupsDelim,omitempty"`
	// List of groups to filter access based on membership
	// +optional
	AllowedGroups []string `json:"allowedGroups,omitempty"`
	// If used with allowed groups, only forwards the allowed groups and not all
	// groups specified.
	// +optional
	FilterGroups bool `json:"filterGroups,omitempty"`

	// Requested format of the NameID. The NameID value is is mapped to the ID Token
	// 'sub' claim.
	//
	// This can be an abbreviated form of the full URI with just the last component. For
	// example, if this value is set to "emailAddress" the format will resolve to:
	//
	//		urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
	//
	// If no value is specified, this value defaults to:
	//
	//		urn:oasis:names:tc:SAML:2.0:nameid-format:persistent
	//
	// +optional
	NameIDPolicyFormat string `json:"nameIDPolicyFormat,omitempty"`
}

type AuthenticationPassword struct {
	// If true login via password is disabled
	Disabled bool `json:"disabled,omitempty"`
}

type AuthenticationMicrosoft struct {
	// Microsoft client id
	ClientID string `json:"clientId"`

	// Microsoft client secret
	ClientSecret string `json:"clientSecret"`

	// loft redirect uri. Usually https://loft.my.domain/auth/microsoft/callback
	RedirectURI string `json:"redirectURI"`

	// tenant configuration parameter controls what kinds of accounts may be authenticated in loft.
	// By default, all types of Microsoft accounts (consumers and organizations) can authenticate in loft via Microsoft.
	// To change this, set the tenant parameter to one of the following:
	//
	// common - both personal and business/school accounts can authenticate in loft via Microsoft (default)
	// consumers - only personal accounts can authenticate in loft
	// organizations - only business/school accounts can authenticate in loft
	// tenant uuid or tenant name - only accounts belonging to specific tenant identified by either tenant uuid or tenant name can authenticate in loft
	// +optional
	Tenant string `json:"tenant,omitempty"`

	// It is possible to require a user to be a member of a particular group in order to be successfully authenticated in loft.
	// +optional
	Groups []string `json:"groups,omitempty"`

	// configuration option restricts the list to include only security groups. By default all groups (security, Office 365, mailing lists) are included.
	// +optional
	OnlySecurityGroups bool `json:"onlySecurityGroups,omitempty"`

	// Restrict the groups claims to include only the user’s groups that are in the configured groups
	// +optional
	UseGroupsAsWhitelist bool `json:"useGroupsAsWhitelist,omitempty"`
}

type AuthenticationGoogle struct {
	// Google client id
	ClientID string `json:"clientId"`

	// Google client secret
	ClientSecret string `json:"clientSecret"`

	// loft redirect uri. E.g. https://loft.my.domain/auth/google/callback
	RedirectURI string `json:"redirectURI"`

	// defaults to "profile" and "email"
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// Optional list of whitelisted domains
	// If this field is nonempty, only users from a listed domain will be allowed to log in
	// +optional
	HostedDomains []string `json:"hostedDomains,omitempty"`

	// Optional list of whitelisted groups
	// If this field is nonempty, only users from a listed group will be allowed to log in
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Optional path to service account json
	// If nonempty, and groups claim is made, will use authentication from file to
	// check groups with the admin directory api
	// +optional
	ServiceAccountFilePath string `json:"serviceAccountFilePath,omitempty"`

	// Required if ServiceAccountFilePath
	// The email of a GSuite super user which the service account will impersonate
	// when listing groups
	// +optional
	AdminEmail string `json:"adminEmail,omitempty"`
}

type AuthenticationGitlab struct {
	// Gitlab client id
	ClientID string `json:"clientId"`

	// Gitlab client secret
	ClientSecret string `json:"clientSecret"`

	// Redirect URI
	RedirectURI string `json:"redirectURI"`

	// BaseURL is optional, default = https://gitlab.com
	// +optional
	BaseURL string `json:"baseURL,omitempty"`

	// Optional groups whitelist, communicated through the "groups" scope.
	// If `groups` is omitted, all of the user's GitLab groups are returned.
	// If `groups` is provided, this acts as a whitelist - only the user's GitLab groups that are in the configured `groups` below will go into the groups claim. Conversely, if the user is not in any of the configured `groups`, the user will not be authenticated.
	// +optional
	Groups []string `json:"groups,omitempty"`
}

type AuthenticationGithub struct {
	// ClientID holds the github client id
	ClientID string `json:"clientId,omitempty"`

	// ClientID holds the github client secret
	ClientSecret string `json:"clientSecret"`

	// RedirectURI holds the redirect URI. Should be https://loft.domain.tld/auth/github/callback
	RedirectURI string `json:"redirectURI"`

	// Loft queries the following organizations for group information.
	// Group claims are formatted as "(org):(team)".
	// For example if a user is part of the "engineering" team of the "coreos"
	// org, the group claim would include "coreos:engineering".
	//
	// If orgs are specified in the config then user MUST be a member of at least one of the specified orgs to
	// authenticate with loft.
	// +optional
	Orgs []AuthenticationGithubOrg `json:"orgs,omitempty"`

	// Required ONLY for GitHub Enterprise.
	// This is the Hostname of the GitHub Enterprise account listed on the
	// management console. Ensure this domain is routable on your network.
	// +optional
	HostName string `json:"hostName,omitempty"`

	// ONLY for GitHub Enterprise. Optional field.
	// Used to support self-signed or untrusted CA root certificates.
	// +optional
	RootCA string `json:"rootCA,omitempty"`
}

// AuthenticationGithubOrg holds org-team filters, in which teams are optional.
type AuthenticationGithubOrg struct {
	// Organization name in github (not slug, full name). Only users in this github
	// organization can authenticate.
	// +optional
	Name string `json:"name"`

	// Names of teams in a github organization. A user will be able to
	// authenticate if they are members of at least one of these teams. Users
	// in the organization can authenticate if this field is omitted from the
	// config file.
	// +optional
	Teams []string `json:"teams,omitempty"`
}

type AuthenticationOIDC struct {
	// IssuerURL is the URL the provider signs ID Tokens as. This will be the "iss"
	// field of all tokens produced by the provider and is used for configuration
	// discovery.
	//
	// The URL is usually the provider's URL without a path, for example
	// "https://accounts.google.com" or "https://login.salesforce.com".
	//
	// The provider must implement configuration discovery.
	// See: https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig
	IssuerURL string `json:"issuerUrl,omitempty"`

	// ClientID the JWT must be issued for, the "sub" field. This plugin only trusts a single
	// client to ensure the plugin can be used with public providers.
	//
	// The plugin supports the "authorized party" OpenID Connect claim, which allows
	// specialized providers to issue tokens to a client for a different client.
	// See: https://openid.net/specs/openid-connect-core-1_0.html#IDToken
	ClientID string `json:"clientId,omitempty"`

	// ClientSecret to issue tokens from the OIDC provider
	ClientSecret string `json:"clientSecret,omitempty"`

	// loft redirect uri. E.g. https://loft.my.domain/auth/oidc/callback
	RedirectURI string `json:"redirectURI,omitempty"`

	// Loft URI to be redirected to after successful logout by OIDC Provider
	// +optional
	PostLogoutRedirectURI string `json:"postLogoutRedirectURI,omitempty"`

	// Path to a PEM encoded root certificate of the provider. Optional
	// +optional
	CAFile string `json:"caFile,omitempty"`

	// Specify whether to communicate without validating SSL certificates
	// +optional
	InsecureCA bool `json:"insecureCa,omitempty"`

	// Configurable key which contains the preferred username claims
	// +optional
	PreferredUsernameClaim string `json:"preferredUsername,omitempty"`

	// LoftUsernameClaim is the JWT field to use as the user's username.
	// +optional
	LoftUsernameClaim string `json:"loftUsernameClaim,omitempty"`

	// UsernameClaim is the JWT field to use as the user's id.
	// +optional
	UsernameClaim string `json:"usernameClaim,omitempty"`

	// EmailClaim is the JWT field to use as the user's email.
	// +optional
	EmailClaim string `json:"emailClaim,omitempty"`

	// UsernamePrefix, if specified, causes claims mapping to username to be prefix with
	// the provided value. A value "oidc:" would result in usernames like "oidc:john".
	// +optional
	UsernamePrefix string `json:"usernamePrefix,omitempty"`

	// GroupsClaim, if specified, causes the OIDCAuthenticator to try to populate the user's
	// groups with an ID Token field. If the GroupsClaim field is present in an ID Token the value
	// must be a string or list of strings.
	// +optional
	GroupsClaim string `json:"groupsClaim,omitempty"`

	// If required groups is non empty, access is denied if the user is not part of at least one
	// of the specified groups.
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Scopes that should be sent to the server. If empty, defaults to "email" and "profile".
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// GetUserInfo, if specified, tells the OIDCAuthenticator to try to populate the user's
	// information from the UserInfo.
	// +optional
	GetUserInfo bool `json:"getUserInfo,omitempty"`

	// GroupsPrefix, if specified, causes claims mapping to group names to be prefixed with the
	// value. A value "oidc:" would result in groups like "oidc:engineering" and "oidc:marketing".
	// +optional
	GroupsPrefix string `json:"groupsPrefix,omitempty"`

	// Type of the OIDC to show in the UI. Only for displaying purposes
	// +optional
	Type string `json:"type,omitempty"`
}

type Cloud struct {
	// ReleaseChannel specifies the release channel for the cloud configuration.
	// This can be used to determine which updates or versions are applied.
	ReleaseChannel string `json:"releaseChannel,omitempty"`

	// MaintenanceWindow specifies the maintenance window for the cloud configuration.
	// This is a structured representation of the time window during which maintenance can occur.
	MaintenanceWindow MaintenanceWindow `json:"maintenanceWindow,omitempty"`
}

type MaintenanceWindow struct {
	// DayOfWeek specifies the day of the week for the maintenance window.
	// It should be a string representing the day, e.g., "Monday", "Tuesday", etc.
	DayOfWeek string `json:"dayOfWeek,omitempty"`

	// TimeWindow specifies the time window for the maintenance.
	// It should be a string representing the time range in 24-hour format, in UTC, e.g., "02:00-03:00".
	TimeWindow string `json:"timeWindow,omitempty"`
}

type CostControl struct {
	// Enabled specifies whether the ROI dashboard should be available in the UI, and if the metrics infrastructure
	// that provides dashboard data is deployed
	Enabled *bool `json:"enabled,omitempty"`

	// Global are settings for globally managed components
	Global CostControlGlobalConfig `json:"global,omitempty"`

	// Cluster are settings for each cluster's managed components. These settings apply to all connected clusters
	// unless overridden by modifying the Cluster's spec
	Cluster CostControlClusterConfig `json:"cluster,omitempty"`

	// Settings specify price-related settings that are taken into account for the ROI dashboard calculations.
	Settings *CostControlSettings `json:"settings,omitempty"`
}

type CostControlGlobalConfig struct {
	// Metrics these settings apply to metric infrastructure used to aggregate metrics across all connected clusters
	Metrics *storagev1.Metrics `json:"metrics,omitempty"`
}

type CostControlClusterConfig struct {
	// Metrics are settings applied to metric infrastructure in each connected cluster. These can be overridden in
	// individual clusters by modifying the Cluster's spec
	Metrics *storagev1.Metrics `json:"metrics,omitempty"`

	// OpenCost are settings applied to OpenCost deployments in each connected cluster. These can be overridden in
	// individual clusters by modifying the Cluster's spec
	OpenCost *storagev1.OpenCost `json:"opencost,omitempty"`
}

type CostControlSettings struct {
	// PriceCurrency specifies the currency.
	PriceCurrency string `json:"priceCurrency,omitempty"`

	// AvgCPUPricePerNode specifies the average CPU price per node.
	AvgCPUPricePerNode *CostControlResourcePrice `json:"averageCPUPricePerNode,omitempty"`

	// AvgRAMPricePerNode specifies the average RAM price per node.
	AvgRAMPricePerNode *CostControlResourcePrice `json:"averageRAMPricePerNode,omitempty"`

	// GPUSettings specifies GPU related settings.
	GPUSettings *CostControlGPUSettings `json:"gpuSettings,omitempty"`

	// ControlPlanePricePerCluster specifies the price of one physical cluster.
	ControlPlanePricePerCluster *CostControlResourcePrice `json:"controlPlanePricePerCluster,omitempty"`
}

type CostControlGPUSettings struct {
	// Enabled specifies whether GPU settings should be available in the UI.
	Enabled bool `json:"enabled,omitempty"`

	// AvgGPUPrice specifies the average GPU price.
	AvgGPUPrice *CostControlResourcePrice `json:"averageGPUPrice,omitempty"`
}

type CostControlResourcePrice struct {
	// Price specifies the price.
	Price float64 `json:"price,omitempty"`

	// TimePeriod specifies the time period for the price.
	TimePeriod string `json:"timePeriod,omitempty"`
}

type ImageBuilder struct {
	// Enabled specifies whether the remote image builder should be available.
	// If it's not available building ad-hoc images from a devcontainer.json is not supported
	Enabled *bool `json:"enabled,omitempty"`

	// Replicas is the number of desired replicas.
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources are compute resource required by the buildkit containers
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}
