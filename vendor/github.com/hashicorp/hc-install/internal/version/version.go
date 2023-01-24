package version

const version = "0.1.0"

// ModuleVersion returns the current version of the github.com/hashicorp/hc-install Go module.
// This is a function to allow for future possible enhancement using debug.BuildInfo.
func ModuleVersion() string {
	return version
}
