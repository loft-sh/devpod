package version

import "strings"

var DevVersion = "v0.0.0"

var version = "v0.0.0"

func GetVersion() string {
	return version
}

func GetMajorVersion() string {
	// use golang.org/x/mod/semver instead?
	s := strings.Split(strings.TrimLeft(GetVersion(), "v"), ".")
	return s[0]
}

func GetMinorVersion() string {
	s := strings.Split(strings.TrimLeft(GetVersion(), "v"), ".")
	if len(s) >= 2 {
		return s[1]
	}
	return ""
}

func GetPatchVersion() string {
	s := strings.Split(strings.TrimLeft(GetVersion(), "v"), ".")
	if len(s) >= 3 {
		return strings.SplitN(s[2], "-", 2)[0]
	}
	return ""
}

func GetPrerelease() string {
	s := strings.SplitN(GetVersion(), "-", 2)
	if len(s) >= 2 {
		// remove build
		return strings.Split(s[1], "+")[0]
	}
	return ""
}

func GetBuild() string {
	s := strings.SplitN(GetVersion(), "+", 2)
	if len(s) >= 2 {
		return s[1]
	}
	return ""
}
