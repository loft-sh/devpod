package licenseapi

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type InstanceTokenAuth struct {
	// Token is the jwt token identifying the loft instance.
	Token string `json:"token" query:"token" validate:"required"`
	// Certificate is the signing certificate for the token.
	Certificate string `json:"certificate" form:"certificate" query:"certificate" validate:"required"`
}

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type InstanceTokenClaims struct {
	// URLHash is the hash of the url to be signed with this token
	URLHash string `json:"url"`

	// PayloadHash is the hash of the payload to be signed with this token
	PayloadHash string `json:"payload"`
}
