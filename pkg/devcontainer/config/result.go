package config

const UserLabel = "devpod.user"

type Result struct {
	DevContainerConfigWithPath *DevContainerConfigWithPath `json:"DevContainerConfigWithPath"`
	MergedConfig               *MergedConfig               `json:"MergedConfig"`
	SubstitutionContext        *SubstitutionContext        `json:"SubstitutionContext"`
	ContainerDetails           *ContainerDetails           `json:"ContainerDetails"`
}

type DevContainerConfigWithPath struct {
	// Config is the devcontainer.json config
	Config *Config `json:"config,omitempty"`

	// Path is the relative path to the devcontainer.json from the workspace folder
	Path string `json:"path,omitempty"`
}

func GetMounts(result *Result) []*Mount {
	workspaceMount := ParseMount(result.SubstitutionContext.WorkspaceMount)
	mounts := []*Mount{&workspaceMount}
	for _, m := range result.MergedConfig.Mounts {
		if m.Type == "bind" {
			mounts = append(mounts, m)
		}
	}

	return mounts
}

func GetRemoteUser(result *Result) string {
	user := "root"
	if result != nil {
		if result.MergedConfig != nil && result.MergedConfig.RemoteUser != "" {
			user = result.MergedConfig.RemoteUser
		} else if result.ContainerDetails != nil && result.ContainerDetails.Config.Labels != nil && result.ContainerDetails.Config.Labels[UserLabel] != "" {
			user = result.ContainerDetails.Config.Labels[UserLabel]
		}
	}

	return user
}

func GetDevPodCustomizations(parsedConfig *Config) *DevPodCustomizations {
	if parsedConfig.Customizations == nil || parsedConfig.Customizations["devpod"] == nil {
		return &DevPodCustomizations{}
	}

	devPod := &DevPodCustomizations{}
	err := Convert(parsedConfig.Customizations["devpod"], devPod)
	if err != nil {
		return &DevPodCustomizations{}
	}

	return devPod
}

func GetVSCodeConfiguration(mergedConfig *MergedConfig) *VSCodeCustomizations {
	if mergedConfig.Customizations == nil || mergedConfig.Customizations["vscode"] == nil {
		return &VSCodeCustomizations{}
	}

	retVSCodeCustomizations := &VSCodeCustomizations{
		Settings:   map[string]interface{}{},
		Extensions: nil,
	}
	for _, customization := range mergedConfig.Customizations["vscode"] {
		vsCode := &VSCodeCustomizations{}
		err := Convert(customization, vsCode)
		if err != nil {
			continue
		}

		for _, extension := range vsCode.Extensions {
			if contains(retVSCodeCustomizations.Extensions, extension) {
				continue
			}

			retVSCodeCustomizations.Extensions = append(retVSCodeCustomizations.Extensions, extension)
		}

		for k, v := range vsCode.Settings {
			retVSCodeCustomizations.Settings[k] = v
		}
	}

	return retVSCodeCustomizations
}

func contains(stack []string, k string) bool {
	for _, s := range stack {
		if s == k {
			return true
		}
	}
	return false
}
