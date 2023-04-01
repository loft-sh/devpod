package config

type ImageDetails struct {
	Id     string
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
	Id      string
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
		Id: containerDetails.Id,
		Config: ImageDetailsConfig{
			User:   containerDetails.Config.User,
			Env:    containerDetails.Config.Env,
			Labels: containerDetails.Config.Labels,
		},
	}
}
