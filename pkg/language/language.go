package language

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/go-enry/go-enry/v2"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/log"
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
	limit := int64(16 * 1024)
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

	out := map[ProgrammingLanguage]int{}
	walkedFiles := 0
	err = filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		walkedFiles++
		if walkedFiles > maxFiles {
			return filepath.SkipDir
		} else if err != nil {
			return filepath.SkipDir
		}

		if !f.Mode().IsDir() && !f.Mode().IsRegular() {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		if relativePath == "." {
			return nil
		}

		if f.IsDir() {
			relativePath = relativePath + "/"
		}

		if enry.IsVendor(relativePath) || enry.IsDotFile(relativePath) ||
			enry.IsDocumentation(relativePath) || enry.IsConfiguration(relativePath) ||
			enry.IsGenerated(relativePath, nil) {
			if f.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if f.IsDir() {
			return nil
		}

		// TODO(bzz): provide API that mimics lingust CLI output for
		// - running ByExtension & ByFilename
		// - reading the file, if that did not work
		// - GetLanguage([]Strategy)
		content, err := readFile(path, limit)
		if err != nil {
			return nil
		}

		if enry.IsGenerated(relativePath, content) {
			return nil
		}

		language := enry.GetLanguage(filepath.Base(path), content)
		if language == enry.OtherLanguage {
			return nil
		}

		if enry.GetLanguageType(language) != enry.Programming {
			return nil
		} else if !SupportedLanguages[ProgrammingLanguage(language)] {
			return nil
		}

		// try to map language
		if MapLanguages[ProgrammingLanguage(language)] != "" {
			language = string(MapLanguages[ProgrammingLanguage(language)])
		}

		// inc found map
		out[ProgrammingLanguage(language)]++
		return nil
	})
	if err != nil {
		return None, err
	}

	programmingLanguage := None
	count := 0
	for k, v := range out {
		if v > count {
			programmingLanguage = k
			count = v
		}
	}

	return programmingLanguage, nil
}

func readFile(path string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return os.ReadFile(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()
	if limit > 0 && size > limit {
		size = limit
	}
	buf := bytes.NewBuffer(nil)
	buf.Grow(int(size))
	_, err = io.Copy(buf, io.LimitReader(f, limit))
	return buf.Bytes(), err
}
