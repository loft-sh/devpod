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
