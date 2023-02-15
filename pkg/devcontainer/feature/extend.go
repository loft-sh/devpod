package feature

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/graph"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var featureSafeIDRegex1 = regexp.MustCompile(`[^\w_]`)
var featureSafeIDRegex2 = regexp.MustCompile(`^[\d_]+`)

const FEATURE_BASE_DOCKERFILE = `
FROM $_DEV_CONTAINERS_BASE_IMAGE AS dev_containers_target_stage

USER root

COPY --from=dev_containers_feature_content_source . /tmp/build-features/
RUN chmod -R 0700 /tmp/build-features

#{featureLayer}

ARG _DEV_CONTAINERS_IMAGE_USER=root
USER $_DEV_CONTAINERS_IMAGE_USER
`

type ExtendedBuildInfo struct {
	Features          []*config.FeatureSet
	FeaturesBuildInfo *BuildInfo

	MetadataConfig *config.ImageMetadataConfig
	MetadataLabel  string
}

type BuildInfo struct {
	FeaturesFolder          string
	DockerfileContent       string
	OverrideTarget          string
	DockerfilePrefixContent string
	BuildArgs               map[string]string
	BuildKitContexts        map[string]string
}

func GetExtendedBuildInfo(substitutionContext *config.SubstitutionContext, baseImageMetadata *config.ImageMetadataConfig, user, target string, devContainerConfig *config.SubstitutedConfig, log log.Logger) (*ExtendedBuildInfo, error) {
	features, err := fetchFeatures(devContainerConfig.Config, log)
	if err != nil {
		return nil, errors.Wrap(err, "fetch features")
	}

	mergedImageMetadataConfig, err := metadata.GetDevContainerMetadata(substitutionContext, baseImageMetadata, devContainerConfig, features)
	if err != nil {
		return nil, errors.Wrap(err, "get dev container metadata")
	}

	marshalled, err := json.Marshal(mergedImageMetadataConfig.Raw)
	if err != nil {
		return nil, err
	}

	// no features?
	if len(features) == 0 {
		return &ExtendedBuildInfo{
			MetadataLabel:  string(marshalled),
			MetadataConfig: mergedImageMetadataConfig,
		}, nil
	}

	buildInfo, err := getFeatureBuildOptions(baseImageMetadata, user, target, features)
	if err != nil {
		return nil, err
	}

	return &ExtendedBuildInfo{
		Features:          features,
		FeaturesBuildInfo: buildInfo,
		MetadataConfig:    mergedImageMetadataConfig,
		MetadataLabel:     string(marshalled),
	}, nil
}

func getFeatureBuildOptions(baseImageMetadata *config.ImageMetadataConfig, user, target string, features []*config.FeatureSet) (*BuildInfo, error) {
	containerUser, remoteUser := findContainerUsers(baseImageMetadata, "", user)

	// copy features
	featureFolder, err := copyFeaturesToDestination(features)
	if err != nil {
		return nil, err
	}

	// write devcontainer-features.builtin.env, its important to have a terminating \n here as we append to that file later
	err = os.WriteFile(filepath.Join(featureFolder, "devcontainer-features.builtin.env"), []byte(`_CONTAINER_USER=`+containerUser+`
_REMOTE_USER=`+remoteUser+"\n"), 0666)
	if err != nil {
		return nil, err
	}

	// prepare dockerfile
	dockerfileContent := strings.ReplaceAll(FEATURE_BASE_DOCKERFILE, "#{featureLayer}", getFeatureLayers(containerUser, remoteUser, features))
	dockerfilePrefix := `
# syntax=docker.io/docker/dockerfile:1.4
ARG _DEV_CONTAINERS_BASE_IMAGE=placeholder`

	return &BuildInfo{
		FeaturesFolder:          featureFolder,
		DockerfileContent:       dockerfileContent,
		OverrideTarget:          "",
		DockerfilePrefixContent: dockerfilePrefix,
		BuildArgs: map[string]string{
			"_DEV_CONTAINERS_BASE_IMAGE": target,
			"_DEV_CONTAINERS_IMAGE_USER": user,
		},
		BuildKitContexts: map[string]string{
			"dev_containers_feature_content_source": featureFolder,
		},
	}, nil
}

func copyFeaturesToDestination(features []*config.FeatureSet) (string, error) {
	tempDir, err := os.MkdirTemp("", "devpod")
	if err != nil {
		return "", errors.Wrap(err, "make temp dir")
	}

	for i, feature := range features {
		featureDir := filepath.Join(tempDir, strconv.Itoa(i))
		err = os.MkdirAll(featureDir, 0755)
		if err != nil {
			return "", err
		}

		err = copy.Directory(feature.Folder, featureDir)
		if err != nil {
			return "", errors.Wrapf(err, "copy feature %s", feature.ConfigID)
		}

		// copy feature folder
		envPath := filepath.Join(featureDir, "devcontainer-features.env")
		variables := getFeatureEnvVariables(feature.Config, feature.Options)
		err = os.WriteFile(envPath, []byte(strings.Join(variables, "\n")), 0666)
		if err != nil {
			return "", errors.Wrapf(err, "write variables of feature %s", feature.ConfigID)
		}

		installWrapperPath := filepath.Join(featureDir, "devcontainer-features-install.sh")
		installWrapperContent := getFeatureInstallWrapperScript(feature.ConfigID, feature.Config, variables)
		err = os.WriteFile(installWrapperPath, []byte(installWrapperContent), 0666)
		if err != nil {
			return "", errors.Wrapf(err, "write install wrapper script for feature %s", feature.ConfigID)
		}
	}

	return tempDir, nil
}

func getFeatureSafeID(featureID string) string {
	return strings.ToUpper(featureSafeIDRegex2.ReplaceAllString(featureSafeIDRegex1.ReplaceAllString(featureID, "_"), "_"))
}

func getFeatureLayers(containerUser, remoteUser string, features []*config.FeatureSet) string {
	result := `RUN \
echo "_CONTAINER_USER_HOME=$(getent passwd ` + containerUser + ` | cut -d: -f6)" >> /tmp/build-features/devcontainer-features.builtin.env && \
echo "_REMOTE_USER_HOME=$(getent passwd ` + remoteUser + ` | cut -d: -f6)" >> /tmp/build-features/devcontainer-features.builtin.env

`
	for i, feature := range features {
		result += generateContainerEnvs(feature)
		result += `
RUN cd /tmp/build-features/` + strconv.Itoa(i) + ` \
&& chmod +x ./devcontainer-features-install.sh \
&& ./devcontainer-features-install.sh

`
	}

	return result
}

func generateContainerEnvs(feature *config.FeatureSet) string {
	result := []string{}
	if len(feature.Config.ContainerEnv) == 0 {
		return ""
	}

	for k, v := range feature.Config.ContainerEnv {
		result = append(result, fmt.Sprintf("ENV %s=%s", k, v))
	}
	return strings.Join(result, "\n")
}

func findContainerUsers(baseImageMetadata *config.ImageMetadataConfig, composeServiceUser, imageUser string) (string, string) {
	reversed := config.ReverseSlice(baseImageMetadata.Config)
	containerUser := ""
	remoteUser := ""
	for _, imageMetadata := range reversed {
		if containerUser == "" && imageMetadata.ContainerUser != "" {
			containerUser = imageMetadata.ContainerUser
		}
		if remoteUser == "" && imageMetadata.RemoteUser != "" {
			remoteUser = imageMetadata.RemoteUser
		}
	}

	if containerUser == "" {
		if composeServiceUser != "" {
			containerUser = composeServiceUser
		} else if imageUser != "" {
			containerUser = imageUser
		}
	}
	if remoteUser == "" {
		if composeServiceUser != "" {
			remoteUser = composeServiceUser
		} else if imageUser != "" {
			remoteUser = imageUser
		}
	}
	return containerUser, remoteUser
}

func fetchFeatures(devContainerConfig *config.DevContainerConfig, log log.Logger) ([]*config.FeatureSet, error) {
	featureSets := []*config.FeatureSet{}
	for featureId, featureOptions := range devContainerConfig.Features {
		featureFolder, err := ProcessFeatureID(featureId, filepath.Dir(devContainerConfig.Origin), log)
		if err != nil {
			return nil, errors.Wrap(err, "process feature "+featureId)
		}

		// parse feature
		featureConfig, err := config.ParseDevContainerFeature(featureFolder)
		if err != nil {
			return nil, errors.Wrap(err, "parse feature "+featureId)
		}

		// add to return array
		featureSets = append(featureSets, &config.FeatureSet{
			ConfigID: NormalizeFeatureID(featureId),
			Folder:   featureFolder,
			Config:   featureConfig,
			Options:  featureOptions,
		})
	}

	// compute order here
	featureSets, err := computeFeatureOrder(devContainerConfig, featureSets)
	if err != nil {
		return nil, errors.Wrap(err, "compute feature order")
	}

	return featureSets, nil
}

func NormalizeFeatureID(featureID string) string {
	ref, err := name.ParseReference(featureID)
	if err != nil {
		return featureID
	}

	tag, ok := ref.(name.Tag)
	if ok {
		return tag.Repository.Name()
	}

	return ref.String()
}

func computeFeatureOrder(devContainer *config.DevContainerConfig, features []*config.FeatureSet) ([]*config.FeatureSet, error) {
	if len(devContainer.OverrideFeatureInstallOrder) == 0 {
		return computeAutomaticFeatureOrder(features)
	}

	automaticOrder, err := computeAutomaticFeatureOrder(features)
	if err != nil {
		return nil, err
	}

	orderedFeatures := []*config.FeatureSet{}
	for _, feature := range devContainer.OverrideFeatureInstallOrder {
		featureID := NormalizeFeatureID(feature)

		// remove from automaticOrder and move to orderedFeatures
		newAutomaticOrder := []*config.FeatureSet{}
		for _, featureConfig := range automaticOrder {
			if featureConfig.ConfigID == featureID {
				orderedFeatures = append(orderedFeatures, featureConfig)
				continue
			}

			newAutomaticOrder = append(newAutomaticOrder, featureConfig)
		}
		automaticOrder = newAutomaticOrder
	}

	orderedFeatures = append(orderedFeatures, automaticOrder...)
	return orderedFeatures, nil
}

func computeAutomaticFeatureOrder(features []*config.FeatureSet) ([]*config.FeatureSet, error) {
	g := graph.NewGraph(graph.NewNode("root", nil))

	// build lookup map
	lookup := map[string]*config.FeatureSet{}
	for _, feature := range features {
		lookup[feature.ConfigID] = feature
	}

	// build graph
	for _, feature := range features {
		_, err := g.InsertNodeAt("root", feature.ConfigID, feature)
		if err != nil {
			return nil, err
		}

		// add edges
		for _, installAfter := range feature.Config.InstallsAfter {
			installAfterFeature, ok := lookup[installAfter]
			if !ok {
				continue
			}

			// add an edge from feature to installAfterFeature
			_, err = g.InsertNodeAt(feature.ConfigID, installAfter, installAfterFeature)
			if err != nil {
				return nil, err
			}
		}
	}

	// now remove node after node
	ordered := []*config.FeatureSet{}
	for {
		leaf := g.GetNextLeaf(g.Root)
		if leaf == g.Root {
			break
		}

		err := g.RemoveNode(leaf.ID)
		if err != nil {
			return nil, err
		}

		ordered = append(ordered, leaf.Data.(*config.FeatureSet))
	}

	return ordered, nil
}
