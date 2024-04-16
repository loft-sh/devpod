package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

type Node struct {
	Name       string      `json:"Name"`
	Endpoint   string      `json:"Endpoint"`
	Platforms  []Platform  `json:"Platforms"`
	Flags      interface{} `json:"Flags"`
	DriverOpts interface{} `json:"DriverOpts"`
	Files      interface{} `json:"Files"`
}

type BuildxInstance struct {
	Name    string `json:"Name"`
	Driver  string `json:"Driver"`
	Nodes   []Node `json:"Nodes"`
	Dynamic bool   `json:"Dynamic"`
}

type DefaultBuilder struct {
	Key    string `json:"Key"`
	Name   string `json:"Name"`
	Global bool   `json:"Global"`
}

func GetBuildxBuilder() ([]byte, error) {
	configDir := GetDockerConfigPath()

	buildxCurrentBuilder := filepath.Join(configDir, "buildx", "current")
	buildxInstancesDir := filepath.Join(configDir, "buildx", "instances")

	// check if we have custom instances
	if _, err := os.Stat(buildxInstancesDir); err != nil {
		// this is not an error, we just don't have custom instances
		return nil, nil
	}

	// check if a default builder is set
	if _, err := os.Stat(buildxCurrentBuilder); err != nil {
		// this is not an error, we just don't have custom builders
		return nil, nil
	}

	var builder DefaultBuilder

	defaultBuilder, err := os.ReadFile(buildxCurrentBuilder)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(defaultBuilder, &builder)
	if err != nil {
		return nil, err
	}

	// check if the buildx builder is a cloud one
	if !strings.Contains(builder.Name, "cloud-") {
		// its a local builder, we don't do anything
		// this is not an error, we just don't have cloud builders
		return nil, nil
	}

	// read the default builder config
	cloudBuilder, err := os.ReadFile(filepath.Join(buildxInstancesDir, builder.Name))
	if err != nil {
		return nil, err
	}

	return cloudBuilder, nil
}
