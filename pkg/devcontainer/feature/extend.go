package feature

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/graph"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

var featureSafeIDRegex1 = regexp.MustCompile(`[^\w_]`)
var featureSafeIDRegex2 = regexp.MustCompile(`^[\d_]+`)

const FEATURE_BASE_DOCKERFILE = `
FROM $_DEV_CONTAINERS_BASE_IMAGE AS dev_containers_target_stage

USER root

COPY ./` + config.DevPodContextFeatureFolder + `/ /tmp/build-features/
RUN chmod -R 0755 /tmp/build-features && ls /tmp/build-features

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
}

func GetExtendedBuildInfo(ctx *config.SubstitutionContext, imageBuildInfo *config.ImageBuildInfo, target string, devContainerConfig *config.SubstitutedConfig, log log.Logger, forceBuild bool) (*ExtendedBuildInfo, error) {
	features, err := fetchFeatures(devContainerConfig.Config, log, forceBuild)
	if err != nil {
		return nil, errors.Wrap(err, "fetch features")
	}

	mergedImageMetadataConfig, err := metadata.GetDevContainerMetadata(ctx, imageBuildInfo.Metadata, devContainerConfig, features)
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

	contextPath := config.GetContextPath(devContainerConfig.Config)
	buildInfo, err := getFeatureBuildOptions(contextPath, imageBuildInfo, target, features)
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

func getFeatureBuildOptions(contextPath string, imageBuildInfo *config.ImageBuildInfo, target string, features []*config.FeatureSet) (*BuildInfo, error) {
	containerUser, remoteUser := findContainerUsers(imageBuildInfo.Metadata, "", imageBuildInfo.User)

	// copy features
	featureFolder := filepath.Join(contextPath, config.DevPodContextFeatureFolder)
	err := copyFeaturesToDestination(features, featureFolder)
	if err != nil {
		return nil, err
	}

	// write devcontainer-features.builtin.env, its important to have a terminating \n here as we append to that file later
	err = os.WriteFile(filepath.Join(featureFolder, "devcontainer-features.builtin.env"), []byte(`_CONTAINER_USER=`+containerUser+`
_REMOTE_USER=`+remoteUser+"\n"), 0600)
	if err != nil {
		return nil, err
	}

	// prepare dockerfile
	dockerfileContent := strings.ReplaceAll(FEATURE_BASE_DOCKERFILE, "#{featureLayer}", getFeatureLayers(containerUser, remoteUser, features))
	// get build syntax from Dockerfile or use default
	syntax := "docker.io/docker/dockerfile:1.4"
	if imageBuildInfo.Dockerfile != nil && imageBuildInfo.Dockerfile.Syntax != "" {
		syntax = imageBuildInfo.Dockerfile.Syntax
	}
	dockerfilePrefix := fmt.Sprintf(`
# syntax=%s
ARG _DEV_CONTAINERS_BASE_IMAGE=placeholder`, syntax)

	return &BuildInfo{
		FeaturesFolder:          featureFolder,
		DockerfileContent:       dockerfileContent,
		DockerfilePrefixContent: dockerfilePrefix,
		OverrideTarget:          "dev_containers_target_stage",
		BuildArgs: map[string]string{
			"_DEV_CONTAINERS_BASE_IMAGE": target,
			"_DEV_CONTAINERS_IMAGE_USER": imageBuildInfo.User,
		},
	}, nil
}

func copyFeaturesToDestination(features []*config.FeatureSet, targetDir string) error {
	// make sure the folder doesn't exist initially
	_ = os.RemoveAll(targetDir)
	for i, feature := range features {
		featureDir := filepath.Join(targetDir, strconv.Itoa(i))
		err := os.MkdirAll(featureDir, 0755)
		if err != nil {
			return err
		}

		err = copy.Directory(feature.Folder, featureDir)
		if err != nil {
			return errors.Wrapf(err, "copy feature %s", feature.ConfigID)
		}

		// copy feature folder
		envPath := filepath.Join(featureDir, "devcontainer-features.env")
		variables := getFeatureEnvVariables(feature.Config, feature.Options)
		err = os.WriteFile(envPath, []byte(strings.Join(variables, "\n")), 0600)
		if err != nil {
			return errors.Wrapf(err, "write variables of feature %s", feature.ConfigID)
		}

		installWrapperPath := filepath.Join(featureDir, "devcontainer-features-install.sh")
		installWrapperContent := getFeatureInstallWrapperScript(feature.ConfigID, feature.Config, variables)
		err = os.WriteFile(installWrapperPath, []byte(installWrapperContent), 0600)
		if err != nil {
			return errors.Wrapf(err, "write install wrapper script for feature %s", feature.ConfigID)
		}
	}

	return nil
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

func fetchFeatures(devContainerConfig *config.DevContainerConfig, log log.Logger, forceBuild bool) ([]*config.FeatureSet, error) {
	featureSets := []*config.FeatureSet{}
	for featureID, featureOptions := range devContainerConfig.Features {
		featureFolder, err := ProcessFeatureID(featureID, devContainerConfig, log, forceBuild)
		if err != nil {
			return nil, errors.Wrap(err, "process feature "+featureID)
		}

		// parse feature
		log.Debugf("Parse dev container feature in %s", featureFolder)
		featureConfig, err := config.ParseDevContainerFeature(featureFolder)
		if err != nil {
			return nil, errors.Wrap(err, "parse feature "+featureID)
		}

		// add to return array
		featureSets = append(featureSets, &config.FeatureSet{
			ConfigID: NormalizeFeatureID(featureID),
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
	g := graph.NewGraph[*config.FeatureSet](graph.NewNode[*config.FeatureSet]("root", nil))

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

		ordered = append(ordered, leaf.Data)
	}

	return ordered, nil
}
