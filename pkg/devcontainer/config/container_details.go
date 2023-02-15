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
	Id              string
	Created         string
	Name            string
	State           ContainerDetailsState
	Config          ContainerDetailsConfig
	Mounts          []ContainerDetailsMount
	NetworkSettings ContainerDetailsNetworkSettings
	Ports           []ContainerDetailsPort
}

type ContainerDetailsPort struct {
	IP          string
	PrivatePort int
	PublicPort  int
	Type        string
}

type ContainerDetailsNetworkSettings struct {
	Ports map[string][]ContainerDetailsNetworkSettingsPort
}

type ContainerDetailsNetworkSettingsPort struct {
	HostIp   string
	HostPort string
}

type ContainerDetailsMount struct {
	Type        string
	Name        string
	Source      string
	Destination string
}

type ContainerDetailsConfig struct {
	Image  string
	User   string
	Env    []string
	Labels map[string]string
}

type ContainerDetailsState struct {
	Status     string
	StartedAt  string
	FinishedAt string
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
