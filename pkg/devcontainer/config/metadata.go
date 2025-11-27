package config

type ImageMetadataConfig struct {
	Raw    []*ImageMetadata
	Config []*ImageMetadata
}

type ImageMetadata struct {
	ID                     string `json:"id,omitempty"`
	Entrypoint             string `json:"entrypoint,omitempty"`
	DevContainerConfigBase `json:",inline"`
	DevContainerActions    `json:",inline"`
	NonComposeBase         `json:",inline"`
}

// AddConfigToImageMetadata add a configuration to the given image metadata.
// This will be used to generate the final image metadata.
func AddConfigToImageMetadata(config *DevContainerConfig, imageMetadataConfig *ImageMetadataConfig) {
	userMetadata := &ImageMetadata{}
	userMetadata.DevContainerConfigBase = config.DevContainerConfigBase
	userMetadata.DevContainerActions = config.DevContainerActions
	userMetadata.NonComposeBase = config.NonComposeBase
	imageMetadataConfig.Config = append(imageMetadataConfig.Config, userMetadata)
}
