package config

type ImageDetails struct {
	ID     string
	Config ImageDetailsConfig
}

type ImageDetailsConfig struct {
	User       string
	Env        []string
	Labels     map[string]string
	Entrypoint []string
	Cmd        []string
}

type ContainerDetails struct {
	ID      string                 `json:"ID,omitempty"`
	Created string                 `json:"Created,omitempty"`
	State   ContainerDetailsState  `json:"State,omitempty"`
	Config  ContainerDetailsConfig `json:"Config,omitempty"`
}

type ContainerDetailsConfig struct {
	Labels map[string]string `json:"Labels,omitempty"`

	// LegacyUser shouldn't get used anymore and is only there for backwards compatibility, please
	// use the label config.UserLabel instead
	LegacyUser string `json:"User,omitempty"`
	// LegacyImage shouldn't get used anymore and is only there for testing
	LegacyImage string `json:"Image,omitempty"`
}

type ContainerDetailsState struct {
	Status    string `json:"Status,omitempty"`
	StartedAt string `json:"StartedAt,omitempty"`
}
