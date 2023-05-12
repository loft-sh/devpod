package config

import "github.com/loft-sh/devpod/pkg/dockerfile"

const (
	DockerIDLabel           = "dev.containers.id"
	DockerfileDefaultTarget = "dev_container_auto_added_stage_label"
)

type BuildOptions struct {
	NoBuild bool

	PrebuildRepositories []string
	Platform             string
	PushRepository       string
}

type BuildInfo struct {
	ImageDetails  *ImageDetails
	ImageMetadata *ImageMetadataConfig
	ImageName     string
	PrebuildHash  string
}

type ImageBuildInfo struct {
	User     string
	Metadata *ImageMetadataConfig

	// Either on of these will be filled as will
	Dockerfile   *dockerfile.Dockerfile
	ImageDetails *ImageDetails
}
