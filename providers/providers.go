package providers

import (
	_ "embed"
)

//go:embed docker/provider.yaml
var DockerProvider string

//go:embed kubernetes/provider.yaml
var KubernetesProvider string

// GetBuiltInProviders retrieves the built in providers
func GetBuiltInProviders() map[string]string {
	return map[string]string{
		"docker":     DockerProvider,
		"kubernetes": KubernetesProvider,
	}
}
