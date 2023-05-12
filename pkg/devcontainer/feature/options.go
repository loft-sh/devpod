package feature

import (
	"fmt"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
)

func getFeatureEnvVariables(feature *config.FeatureConfig, featureOptions interface{}) []string {
	options := getFeatureValueObject(feature, featureOptions)
	variables := []string{}
	for k, v := range options {
		variables = append(variables, fmt.Sprintf(`%s="%s"`, getFeatureSafeID(k), v))
	}

	return variables
}

func getFeatureValueObject(feature *config.FeatureConfig, featureOptions interface{}) map[string]interface{} {
	defaults := getFeatureDefaults(feature)
	switch t := featureOptions.(type) {
	case map[string]interface{}:
		for k, v := range t {
			defaults[k] = v
		}

		return defaults
	case string:
		if feature.Options == nil {
			return defaults
		}

		_, ok := feature.Options["version"]
		if ok {
			defaults["version"] = t
		}

		return defaults
	}

	return defaults
}

func getFeatureDefaults(feature *config.FeatureConfig) map[string]interface{} {
	ret := map[string]interface{}{}
	for k, v := range feature.Options {
		ret[k] = string(v.Default)
	}

	return ret
}
