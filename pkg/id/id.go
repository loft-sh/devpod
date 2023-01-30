package id

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var dockerImageNameRegEx = regexp.MustCompile(`[^a-z0-9\-_]+`)

func SafeConcatName(name ...string) string {
	return SafeConcatNameMax(name, 63)
}

func SafeConcatNameMax(name []string, max int) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > max {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.ReplaceAll(fullPath[0:max-8]+"-"+hex.EncodeToString(digest[0:])[0:7], ".-", "-")
	}
	return fullPath
}

func ToDockerImageName(name string) string {
	name = strings.ToLower(name)
	name = dockerImageNameRegEx.ReplaceAllString(name, "")
	return name
}
