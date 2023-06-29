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
	Image  string            `json:"Image,omitempty"`
	User   string            `json:"User,omitempty"`
	Env    []string          `json:"Env,omitempty"`
	Labels map[string]string `json:"Labels,omitempty"`
}

type ContainerDetailsState struct {
	Status    string `json:"Status,omitempty"`
	StartedAt string `json:"StartedAt,omitempty"`
}

func ContainerToImageDetails(containerDetails *ContainerDetails) *ImageDetails {
	return &ImageDetails{
		ID: containerDetails.ID,
		Config: ImageDetailsConfig{
			User:   containerDetails.Config.User,
			Env:    containerDetails.Config.Env,
			Labels: containerDetails.Config.Labels,
		},
	}
}
