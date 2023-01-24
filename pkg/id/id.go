package id

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var convertRegEx1 = regexp.MustCompile(`[\@/\.\:\s]+`)
var convertRegEx2 = regexp.MustCompile(`[^a-z0-9\-]+`)

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

func Convert(ID string) string {
	ID = strings.ToLower(ID)
	ID = convertRegEx1.ReplaceAllString(ID, "-")
	ID = convertRegEx2.ReplaceAllString(ID, "")
	return SafeConcatName(ID)
}

func WorkspaceID(repository string) string {
	repository = strings.TrimPrefix(repository, "http://")
	repository = strings.TrimPrefix(repository, "https://")
	if strings.HasPrefix(repository, "git@") {
		repository = strings.TrimPrefix(repository, "git@")
		repository = strings.Replace(repository, ":", "/", 1)
	}
	repository = strings.TrimSuffix(repository, ".git")
	return Convert(repository)
}
