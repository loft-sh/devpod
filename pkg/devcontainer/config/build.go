package config

import (
	"github.com/loft-sh/devpod/pkg/dockerfile"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
)

const (
	DockerIDLabel           = "dev.containers.id"
	DockerfileDefaultTarget = "dev_container_auto_added_stage_label"
)

func GetDockerLabelForID(id string) []string {
	return []string{DockerIDLabel + "=" + id}
}

type BuildOptions struct {
	provider2.CLIOptions

	Platform string
	NoBuild  bool
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
