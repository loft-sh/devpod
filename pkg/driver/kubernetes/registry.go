package kubernetes

import (
	"strings"

	"github.com/distribution/reference"
)

const OfficialDockerRegistry = "https://index.docker.io/v1/"

// GetRegistryFromImageName retrieves the registry name from an imageName
func GetRegistryFromImageName(imageName string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return "", err
	}

	repoInfo, err := newIndexInfo(reference.Domain(ref))
	if err != nil {
		return "", err
	}

	if !strings.ContainsRune(reference.FamiliarName(ref), '/') || repoInfo == "hub.docker.com" || repoInfo == "docker.io" {
		return OfficialDockerRegistry, nil
	}

	return repoInfo, nil
}

// validateIndexName validates an index name. It is used by the daemon to
// validate the daemon configuration.
func validateIndexName(val string) (string, error) {
	// TODO: upstream this to check to reference package
	if val == "index.docker.io" {
		val = "docker.io"
	}
	return val, nil
}

// newIndexInfo returns IndexInfo configuration from indexName
func newIndexInfo(indexName string) (string, error) {
	var err error
	indexName, err = validateIndexName(indexName)
	if err != nil {
		return "", err
	}

	// Construct a non-configured index info.
	return indexName, nil
}
