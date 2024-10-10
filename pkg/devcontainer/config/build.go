package config

import (
	"github.com/loft-sh/devpod/pkg/dockerfile"
)

const (
	DockerIDLabel           = "dev.containers.id"
	DockerfileDefaultTarget = "dev_container_auto_added_stage_label"

	DevPodContextFeatureFolder      = ".devpod-internal"
	DevPodDockerlessBuildInfoFolder = "/workspaces/.dockerless"
)

func GetDockerLabelForID(id string) []string {
	return []string{DockerIDLabel + "=" + id}
}

// BuildInfo is returned once an image has been built and contains details about the image
type BuildInfo struct {
	ImageDetails  *ImageDetails
	ImageMetadata *ImageMetadataConfig
	ImageName     string
	PrebuildHash  string
	RegistryCache string

	Dockerless *BuildInfoDockerless
}

// BuildInfoDockerless is the specific config used by kaniko to build a "dockerless image"
type BuildInfoDockerless struct {
	Context    string
	Dockerfile string

	BuildArgs map[string]string
	Target    string

	User string
}

// ImageBuildInfo contains the spec to use when building an image
type ImageBuildInfo struct {
	User     string
	Metadata *ImageMetadataConfig

	// Either on of these will be filled as will
	Dockerfile   *dockerfile.Dockerfile
	ImageDetails *ImageDetails
}
