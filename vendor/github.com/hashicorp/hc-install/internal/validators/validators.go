package validators

import "regexp"

var (
	productNameRe = regexp.MustCompile(`^[a-z0-9-]+$`)
	binaryNameRe  = regexp.MustCompile(`^[a-zA-Z0-9-_.]+$`)
)

// IsProductNameValid provides early user-facing validation of a product name
func IsProductNameValid(productName string) bool {
	return productNameRe.MatchString(productName)
}

// IsBinaryNameValid provides early user-facing validation of binary name
func IsBinaryNameValid(binaryName string) bool {
	return binaryNameRe.MatchString(binaryName)
}
