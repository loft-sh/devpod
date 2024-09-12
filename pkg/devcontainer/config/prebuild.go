package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	util "github.com/loft-sh/devpod/pkg/util/hash"
	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"github.com/moby/patternmatcher"
	"github.com/pkg/errors"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/hash"
)

func CalculatePrebuildHash(
	originalConfig *DevContainerConfig,
	platform, architecture, contextPath, dockerfilePath, dockerfileContent string,
	buildInfo *ImageBuildInfo,
	log log.Logger) (string, error) {
	parsedConfig := CloneDevContainerConfig(originalConfig)

	if platform != "" {
		splitted := strings.Split(platform, "/")
		if len(splitted) == 2 && splitted[0] == "linux" {
			architecture = splitted[1]
		}
	}

	// delete all options that are not relevant for the build
	parsedConfig.Origin = ""
	parsedConfig.DevContainerActions = DevContainerActions{}
	parsedConfig.NonComposeBase = NonComposeBase{}
	parsedConfig.DevContainerConfigBase = DevContainerConfigBase{
		Name:                        parsedConfig.Name,
		Features:                    parsedConfig.Features,
		OverrideFeatureInstallOrder: parsedConfig.OverrideFeatureInstallOrder,
	}
	parsedConfig.ImageContainer = ImageContainer{
		Image: parsedConfig.Image,
	}
	parsedConfig.ComposeContainer = ComposeContainer{}
	parsedConfig.DockerfileContainer = DockerfileContainer{
		Dockerfile: parsedConfig.Dockerfile,
		Context:    parsedConfig.Context,
		Build:      parsedConfig.Build,
	}

	// marshal the config
	configStr, err := json.Marshal(parsedConfig)
	if err != nil {
		return "", err
	}

	// find out excludes from dockerignore
	excludes, err := readDockerignore(contextPath, dockerfilePath)
	if err != nil {
		return "", errors.Errorf("Error reading .dockerignore: %v", err)
	}
	excludes = append(excludes, DevPodContextFeatureFolder+"/")

	// find exact files to hash
	// todo pass down target or search all
	// todo update DirectoryHash function
	var includes []string
	if buildInfo.Dockerfile != nil {
		includes = buildInfo.Dockerfile.BuildContextFiles()
	}
	log.Debug("Build context files to use for hash are ", includes)

	// get hash of the context directory
	contextHash, err := util.DirectoryHash(contextPath, excludes, includes)
	if err != nil {
		return "", err
	}

	log.Debugf("Prebuild hash from:")
	log.Debugf("    Arch: %s", architecture)
	log.Debugf("    Config: %s", string(configStr))
	log.Debugf("    DockerfileContent: %s", dockerfileContent)
	log.Debugf("    ContextHash: %s", contextHash)
	return "devpod-" + hash.String(architecture + string(configStr) + dockerfileContent + contextHash)[:32], nil
}

// readDockerignore reads the .dockerignore file in the context directory and
// returns the list of paths to exclude.
func readDockerignore(contextDir string, dockerfile string) ([]string, error) {
	var (
		f        *os.File
		err      error
		excludes = []string{}
	)

	dockerignorefilepath := dockerfile + ".dockerignore"
	if filepath.IsAbs(dockerignorefilepath) {
		f, err = os.Open(dockerignorefilepath)
	} else {
		f, err = os.Open(filepath.Join(contextDir, dockerignorefilepath))
	}
	if os.IsNotExist(err) {
		dockerignorefilepath = ".dockerignore"
		f, err = os.Open(filepath.Join(contextDir, dockerignorefilepath))
		if os.IsNotExist(err) {
			return ensureDockerIgnoreAndDockerFile(excludes, dockerfile, dockerignorefilepath), nil
		} else if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	defer f.Close()

	excludes, err = dockerignore.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return ensureDockerIgnoreAndDockerFile(excludes, dockerfile, dockerignorefilepath), nil
}

func ensureDockerIgnoreAndDockerFile(excludes []string, dockerfile, dockerignorefilepath string) []string {
	_, dockerignorefile := filepath.Split(dockerignorefilepath)
	if keep, _ := patternmatcher.MatchesOrParentMatches(dockerignorefile, excludes); keep {
		excludes = append(excludes, "!"+dockerignorefile)
	}

	if keep, _ := patternmatcher.MatchesOrParentMatches(dockerfile, excludes); keep {
		excludes = append(excludes, "!"+dockerfile)
	}

	return excludes
}
