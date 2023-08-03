package metadata

import (
	"encoding/json"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/log"
)

const ImageMetadataLabel = "devcontainer.metadata"

func GetDevContainerMetadata(substitutionContext *config.SubstitutionContext, baseImageMetadata *config.ImageMetadataConfig, devContainerConfig *config.SubstitutedConfig, featuresConfig []*config.FeatureSet) (*config.ImageMetadataConfig, error) {
	// features
	featuresRaw := []*config.ImageMetadata{}
	for _, featureSet := range featuresConfig {
		featuresRaw = append(featuresRaw, FeatureConfigToImageMetadata(featureSet.Config))
	}

	retImageMetadataConfig := &config.ImageMetadataConfig{}
	retImageMetadataConfig.Raw = append(retImageMetadataConfig.Raw, baseImageMetadata.Raw...)
	retImageMetadataConfig.Raw = append(retImageMetadataConfig.Raw, featuresRaw...)
	retImageMetadataConfig.Raw = append(retImageMetadataConfig.Raw, DevContainerConfigToImageMetadata(devContainerConfig.Raw))

	retImageMetadataConfig.Config = append(retImageMetadataConfig.Config, baseImageMetadata.Config...)
	for _, featureRaw := range featuresRaw {
		featureConfig := &config.ImageMetadata{}
		err := config.Substitute(substitutionContext, featureRaw, featureConfig)
		if err != nil {
			return nil, err
		}

		retImageMetadataConfig.Config = append(retImageMetadataConfig.Config, featureConfig)
	}
	retImageMetadataConfig.Config = append(retImageMetadataConfig.Config, DevContainerConfigToImageMetadata(devContainerConfig.Config))
	return retImageMetadataConfig, nil
}

func FeatureConfigToImageMetadata(feature *config.FeatureConfig) *config.ImageMetadata {
	return &config.ImageMetadata{
		Entrypoint: feature.Entrypoint,
		DevContainerActions: config.DevContainerActions{
			Customizations: feature.Customizations,
		},
		NonComposeBase: config.NonComposeBase{
			Mounts:      feature.Mounts,
			Init:        feature.Init,
			Privileged:  feature.Privileged,
			CapAdd:      feature.CapAdd,
			SecurityOpt: feature.SecurityOpt,
		},
	}
}

func DevContainerConfigToImageMetadata(devConfig *config.DevContainerConfig) *config.ImageMetadata {
	return &config.ImageMetadata{
		DevContainerConfigBase: config.DevContainerConfigBase{
			ForwardPorts:         devConfig.ForwardPorts,
			PortsAttributes:      devConfig.PortsAttributes,
			OtherPortsAttributes: devConfig.OtherPortsAttributes,
			UpdateRemoteUserUID:  devConfig.UpdateRemoteUserUID,
			RemoteEnv:            devConfig.RemoteEnv,
			RemoteUser:           devConfig.RemoteUser,
			ShutdownAction:       devConfig.ShutdownAction,
			WaitFor:              devConfig.WaitFor,
			UserEnvProbe:         devConfig.UserEnvProbe,
			HostRequirements:     devConfig.HostRequirements,
			OverrideCommand:      devConfig.OverrideCommand,
		},
		DevContainerActions: config.DevContainerActions{
			OnCreateCommand:      devConfig.OnCreateCommand,
			UpdateContentCommand: devConfig.UpdateContentCommand,
			PostCreateCommand:    devConfig.PostCreateCommand,
			PostStartCommand:     devConfig.PostStartCommand,
			PostAttachCommand:    devConfig.PostAttachCommand,
			Customizations:       devConfig.Customizations,
		},
		NonComposeBase: config.NonComposeBase{
			ContainerEnv:  devConfig.ContainerEnv,
			ContainerUser: devConfig.ContainerUser,
			Mounts:        devConfig.Mounts,
			Init:          devConfig.Init,
			Privileged:    devConfig.Privileged,
			CapAdd:        devConfig.CapAdd,
			SecurityOpt:   devConfig.SecurityOpt,
		},
	}
}

func GetImageMetadataFromContainer(containerDetails *config.ContainerDetails, substituteContext *config.SubstitutionContext, log log.Logger) (*config.ImageMetadataConfig, error) {
	if containerDetails == nil || containerDetails.Config.Labels == nil || containerDetails.Config.Labels[ImageMetadataLabel] == "" {
		return &config.ImageMetadataConfig{}, nil
	}

	imageMetadataConfig := &config.ImageMetadataConfig{}
	multiple := []*config.ImageMetadata{}
	err := json.Unmarshal([]byte(containerDetails.Config.Labels[ImageMetadataLabel]), &multiple)
	if err != nil {
		single := &config.ImageMetadata{}
		err = json.Unmarshal([]byte(containerDetails.Config.Labels[ImageMetadataLabel]), single)
		if err != nil {
			log.Errorf("Error parsing image metadata: %v", err)
			return &config.ImageMetadataConfig{}, nil
		}

		imageMetadataConfig.Raw = []*config.ImageMetadata{single}
	} else {
		imageMetadataConfig.Raw = multiple
	}

	err = substituteConfig(imageMetadataConfig, substituteContext)
	if err != nil {
		return nil, err
	}

	return imageMetadataConfig, nil
}

func GetImageMetadata(imageDetails *config.ImageDetails, substituteContext *config.SubstitutionContext, log log.Logger) (*config.ImageMetadataConfig, error) {
	if imageDetails.Config.Labels == nil || imageDetails.Config.Labels[ImageMetadataLabel] == "" {
		return &config.ImageMetadataConfig{}, nil
	}

	imageMetadataConfig := &config.ImageMetadataConfig{}
	multiple := []*config.ImageMetadata{}
	err := json.Unmarshal([]byte(imageDetails.Config.Labels[ImageMetadataLabel]), &multiple)
	if err != nil {
		single := &config.ImageMetadata{}
		err = json.Unmarshal([]byte(imageDetails.Config.Labels[ImageMetadataLabel]), single)
		if err != nil {
			log.Errorf("Error parsing image metadata: %v", err)
			return &config.ImageMetadataConfig{}, nil
		}

		imageMetadataConfig.Raw = []*config.ImageMetadata{single}
	} else {
		imageMetadataConfig.Raw = multiple
	}

	err = substituteConfig(imageMetadataConfig, substituteContext)
	if err != nil {
		return nil, err
	}

	return imageMetadataConfig, nil
}

func substituteConfig(imageConfig *config.ImageMetadataConfig, substituteContext *config.SubstitutionContext) error {
	imageConfig.Config = []*config.ImageMetadata{}
	for _, raw := range imageConfig.Raw {
		imageMetadata := &config.ImageMetadata{}
		err := config.Substitute(substituteContext, raw, imageMetadata)
		if err != nil {
			return err
		}

		imageConfig.Config = append(imageConfig.Config, imageMetadata)
	}

	return nil
}
