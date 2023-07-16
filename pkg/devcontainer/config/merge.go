package config

import (
	"strconv"

	"github.com/loft-sh/devpod/pkg/types"
)

func MergeConfiguration(config *DevContainerConfig, imageMetadataEntries []*ImageMetadata) (*MergedDevContainerConfig, error) {
	customizations := map[string][]interface{}{}
	for _, imageMetadata := range imageMetadataEntries {
		for k, v := range imageMetadata.Customizations {
			customizations[k] = append(customizations[k], v)
		}
	}

	copiedConfig := CloneDevContainerConfig(config)

	// reverse the order
	reversed := ReverseSlice(imageMetadataEntries)

	// merge config
	mergedConfig := &MergedDevContainerConfig{
		UpdatedConfigProperties: UpdatedConfigProperties{
			Customizations: customizations,
		},
		DevContainerConfigBase: copiedConfig.DevContainerConfigBase,
		NonComposeBase:         copiedConfig.NonComposeBase,
		ImageContainer:         copiedConfig.ImageContainer,
		ComposeContainer:       copiedConfig.ComposeContainer,
		DockerfileContainer:    copiedConfig.DockerfileContainer,
	}

	// adjust config
	mergedConfig.Init = some(reversed, func(entry *ImageMetadata) *bool { return entry.Init })
	mergedConfig.Privileged = some(reversed, func(entry *ImageMetadata) *bool { return entry.Privileged })
	mergedConfig.CapAdd = unique(unionOrNil(reversed, func(entry *ImageMetadata) []string { return entry.CapAdd }))
	mergedConfig.SecurityOpt = unique(unionOrNil(reversed, func(entry *ImageMetadata) []string { return entry.SecurityOpt }))
	mergedConfig.Entrypoints = collectOrNil(reversed, func(entry *ImageMetadata) string { return entry.Entrypoint })
	mergedConfig.Mounts = mergeMounts(reversed)
	mergedConfig.OnCreateCommands = mergeLifestyleHooks(reversed, func(entry *ImageMetadata) types.LifecycleHook { return entry.OnCreateCommand })
	mergedConfig.UpdateContentCommands = mergeLifestyleHooks(reversed, func(entry *ImageMetadata) types.LifecycleHook { return entry.UpdateContentCommand })
	mergedConfig.PostCreateCommands = mergeLifestyleHooks(reversed, func(entry *ImageMetadata) types.LifecycleHook { return entry.PostCreateCommand })
	mergedConfig.PostStartCommands = mergeLifestyleHooks(reversed, func(entry *ImageMetadata) types.LifecycleHook { return entry.PostStartCommand })
	mergedConfig.PostAttachCommands = mergeLifestyleHooks(reversed, func(entry *ImageMetadata) types.LifecycleHook { return entry.PostAttachCommand })
	mergedConfig.WaitFor = firstString(reversed, func(entry *ImageMetadata) string { return entry.WaitFor })
	mergedConfig.RemoteUser = firstString(reversed, func(entry *ImageMetadata) string { return entry.RemoteUser })
	mergedConfig.ContainerUser = firstString(reversed, func(entry *ImageMetadata) string { return entry.ContainerUser })
	mergedConfig.UserEnvProbe = firstString(reversed, func(entry *ImageMetadata) string { return entry.UserEnvProbe })
	mergedConfig.RemoteEnv = mergeMaps(reversed, func(entry *ImageMetadata) map[string]string { return entry.RemoteEnv })
	mergedConfig.ContainerEnv = mergeMaps(reversed, func(entry *ImageMetadata) map[string]string { return entry.ContainerEnv })
	mergedConfig.PortsAttributes = mergeMaps(reversed, func(entry *ImageMetadata) map[string]PortAttribute { return entry.PortsAttributes })
	mergedConfig.OverrideCommand = some(reversed, func(entry *ImageMetadata) *bool { return entry.OverrideCommand })
	mergedConfig.OtherPortsAttributes = mergeOtherPortsAttributes(reversed)
	mergedConfig.ShutdownAction = firstString(reversed, func(entry *ImageMetadata) string { return entry.ShutdownAction })
	mergedConfig.ForwardPorts = mergeForwardPorts(reversed)
	mergedConfig.UpdateRemoteUserUID = some(reversed, func(entry *ImageMetadata) *bool { return entry.UpdateRemoteUserUID })
	mergedConfig.HostRequirements = mergeHostRequirements(reversed)

	return mergedConfig, nil
}

func mergeOtherPortsAttributes(entries []*ImageMetadata) map[string]PortAttribute {
	for _, entry := range entries {
		if len(entry.OtherPortsAttributes) > 0 {
			return entry.OtherPortsAttributes
		}
	}
	return nil
}

func mergeMaps[K any](entries []*ImageMetadata, m func(entry *ImageMetadata) map[string]K) map[string]K {
	retMap := map[string]K{}
	for _, entry := range entries {
		entryMap := m(entry)
		for k, v := range entryMap {
			retMap[k] = v
		}
	}

	return retMap
}

func firstString(entries []*ImageMetadata, m func(entry *ImageMetadata) string) string {
	for _, entry := range entries {
		str := m(entry)
		if str != "" {
			return str
		}
	}
	return ""
}

func mergeHostRequirements(entries []*ImageMetadata) *HostRequirements {
	// TODO: union requirements here
	for _, entry := range entries {
		if entry.HostRequirements != nil {
			return entry.HostRequirements
		}
	}

	return nil
}

func mergeForwardPorts(entries []*ImageMetadata) types.StrIntArray {
	portMap := map[string]bool{}
	var retPorts types.StrIntArray
	for _, entry := range entries {
		for _, port := range entry.ForwardPorts {
			portString := port
			_, err := strconv.Atoi(portString)
			if err == nil {
				portString = "localhost:" + portString
			}
			if portMap[portString] {
				continue
			}

			portMap[portString] = true
			retPorts = append(retPorts, port)
		}
	}

	return retPorts
}

func mergeMounts(entries []*ImageMetadata) []*Mount {
	targetMap := map[string]bool{}
	ret := []*Mount{}

	reversedEntries := ReverseSlice(entries)
	for _, entry := range reversedEntries {
		for _, mount := range entry.Mounts {
			if targetMap[mount.Target] {
				continue
			}

			ret = append(ret, mount)
			targetMap[mount.Target] = true
		}
	}
	return ReverseSlice(ret)
}

func mergeLifestyleHooks(entries []*ImageMetadata, m func(entry *ImageMetadata) types.LifecycleHook) []types.LifecycleHook {
	var out []types.LifecycleHook
	for _, entry := range entries {
		val := m(entry)
		if len(val) > 0 {
			out = append(out, m(entry))
		}
	}
	return out
}

func collectOrNil[T comparable, K any](entries []K, m func(entry K) T) []T {
	var out []T
	for _, entry := range entries {
		var defaultValue T
		val := m(entry)
		if val != defaultValue {
			out = append(out, m(entry))
		}
	}

	return out
}

func unionOrNil[T any, K any](entries []K, m func(entry K) []T) []T {
	var out []T
	for _, entry := range entries {
		vals := m(entry)
		if len(vals) > 0 {
			out = append(out, vals...)
		}
	}

	return out
}

func unique[T comparable](s []T) []T {
	inResult := make(map[T]bool)
	var result []T
	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
			result = append(result, str)
		}
	}
	return result
}

func some[T any](entries []T, m func(entry T) *bool) *bool {
	for _, entry := range entries {
		boolPtr := m(entry)
		if boolPtr != nil {
			return boolPtr
		}
	}
	return nil
}

func ReverseSlice[T comparable](s []T) []T {
	var r []T
	for i := len(s) - 1; i >= 0; i-- {
		r = append(r, s[i])
	}
	return r
}
