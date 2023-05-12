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
	ID      string
	Created string
	State   ContainerDetailsState
	Config  ContainerDetailsConfig
}

type ContainerDetailsConfig struct {
	Image  string
	User   string
	Env    []string
	Labels map[string]string
}

type ContainerDetailsState struct {
	Status    string
	StartedAt string
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
