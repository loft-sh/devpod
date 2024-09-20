package config

type ImageMetadataConfig struct {
	Raw    []*ImageMetadata
	Config []*ImageMetadata
}

type ImageMetadata struct {
	ID             string `json:"id,omitempty"`
	Entrypoint     string `json:"entrypoint,omitempty"`
	ConfigBase     `json:",inline"`
	Actions        `json:",inline"`
	NonComposeBase `json:",inline"`
}
