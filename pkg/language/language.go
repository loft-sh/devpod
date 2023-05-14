package language

import (
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/log"

	detector "github.com/loft-sh/programming-language-detection/pkg/detector"
)

type ProgrammingLanguage string

const (
	JavaScript ProgrammingLanguage = "JavaScript"
	TypeScript ProgrammingLanguage = "TypeScript"
	Python     ProgrammingLanguage = "Python"
	Go         ProgrammingLanguage = "Go"
	Cpp        ProgrammingLanguage = "C++"
	C          ProgrammingLanguage = "C"
	DotNet     ProgrammingLanguage = "C#"
	PHP        ProgrammingLanguage = "Php"
	Java       ProgrammingLanguage = "Java"
	Rust       ProgrammingLanguage = "Rust"
	Ruby       ProgrammingLanguage = "Ruby"
	None       ProgrammingLanguage = "None"
)

var SupportedLanguages = map[ProgrammingLanguage]bool{
	JavaScript: true,
	TypeScript: true,
	Python:     true,
	C:          true,
	Cpp:        true,
	DotNet:     true,
	Go:         true,
	PHP:        true,
	Java:       true,
	Rust:       true,
	Ruby:       true,
	None:       true,
}

var MapLanguages = map[ProgrammingLanguage]ProgrammingLanguage{
	TypeScript: JavaScript,
	C:          Cpp,
}

var MapConfig = map[ProgrammingLanguage]*config.DevContainerConfig{
	None: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/base:ubuntu",
		},
	},
	JavaScript: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/javascript-node",
		},
	},
	Python: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/python:3",
		},
	},
	Java: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/java",
		},
	},
	Go: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/go",
		},
	},
	Rust: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/rust:latest",
		},
	},
	Ruby: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/ruby",
		},
	},
	PHP: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/php",
		},
	},
	Cpp: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/cpp",
		},
	},
	DotNet: {
		ImageContainer: config.ImageContainer{
			Image: "mcr.microsoft.com/devcontainers/dotnet",
		},
	},
}

func DefaultConfig(startPath string, log log.Logger) *config.DevContainerConfig {
	language, err := DetectLanguage(startPath)
	if err != nil {
		log.Errorf("Error detecting project language: %v", err)
		log.Infof("Couldn't detect project language, fallback to 'None'")
		return MapConfig[None]
	} else if MapConfig[language] == nil {
		log.Infof("Couldn't detect project language, fallback to 'None'")
		return MapConfig[None]
	}

	log.Infof("Detected project language '%s'", language)
	return MapConfig[language]
}

func DetectLanguage(startPath string) (ProgrammingLanguage, error) {
	maxFiles := 5000

	root, err := filepath.Abs(startPath)
	if err != nil {
		return None, err
	}

	fileInfo, err := os.Stat(root)
	if err != nil {
		return None, err
	}

	if fileInfo.Mode().IsRegular() {
		return None, err
	}

	language := detector.GetLanguage(root, maxFiles)
	if !SupportedLanguages[ProgrammingLanguage(language)] {
		return None, nil
	}

	// try to map language
	if MapLanguages[ProgrammingLanguage(language)] != "" {
		language = string(MapLanguages[ProgrammingLanguage(language)])
	}

	return ProgrammingLanguage(language), nil
}
