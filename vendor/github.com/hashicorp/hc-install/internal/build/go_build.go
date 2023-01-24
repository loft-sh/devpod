package build

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-version"
)

var discardLogger = log.New(ioutil.Discard, "", 0)

// GoBuild represents a Go builder (to run "go build")
type GoBuild struct {
	Version         *version.Version
	DetectVendoring bool

	pathToRemove string
	logger       *log.Logger
}

func (gb *GoBuild) SetLogger(logger *log.Logger) {
	gb.logger = logger
}

func (gb *GoBuild) log() *log.Logger {
	if gb.logger == nil {
		return discardLogger
	}
	return gb.logger
}

// Build runs "go build" within a given repo to produce binaryName in targetDir
func (gb *GoBuild) Build(ctx context.Context, repoDir, targetDir, binaryName string) (string, error) {
	goCmd, cleanupFunc, err := gb.ensureRequiredGoVersion(ctx, repoDir)
	if err != nil {
		return "", err
	}
	defer cleanupFunc(ctx)

	goArgs := []string{"build", "-o", filepath.Join(targetDir, binaryName)}

	if gb.DetectVendoring {
		vendorDir := filepath.Join(repoDir, "vendor")
		if fi, err := os.Stat(vendorDir); err == nil && fi.IsDir() {
			goArgs = append(goArgs, "-mod", "vendor")
		}
	}

	gb.log().Printf("executing %s %q in %q", goCmd, goArgs, repoDir)
	cmd := exec.CommandContext(ctx, goCmd, goArgs...)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unable to build: %w\n%s", err, out)
	}

	binPath := filepath.Join(targetDir, binaryName)

	gb.pathToRemove = binPath

	return binPath, nil
}

func (gb *GoBuild) Remove(ctx context.Context) error {
	return os.RemoveAll(gb.pathToRemove)
}

func (gb *GoBuild) ensureRequiredGoVersion(ctx context.Context, repoDir string) (string, CleanupFunc, error) {
	cmdName := "go"
	noopCleanupFunc := func(context.Context) {}

	if gb.Version != nil {
		goVersion, err := GetGoVersion(ctx)
		if err != nil {
			return cmdName, noopCleanupFunc, err
		}

		if !goVersion.GreaterThanOrEqual(gb.Version) {
			// found incompatible version, try downloading the desired one
			return gb.installGoVersion(ctx, gb.Version)
		}
	}

	if requiredVersion, ok := guessRequiredGoVersion(repoDir); ok {
		goVersion, err := GetGoVersion(ctx)
		if err != nil {
			return cmdName, noopCleanupFunc, err
		}

		if !goVersion.GreaterThanOrEqual(requiredVersion) {
			// found incompatible version, try downloading the desired one
			return gb.installGoVersion(ctx, requiredVersion)
		}
	}

	return cmdName, noopCleanupFunc, nil
}

// CleanupFunc represents a function to be called once Go is no longer needed
// e.g. to remove any version installed temporarily per requirements
type CleanupFunc func(context.Context)

func guessRequiredGoVersion(repoDir string) (*version.Version, bool) {
	goEnvFile := filepath.Join(repoDir, ".go-version")
	if fi, err := os.Stat(goEnvFile); err == nil && !fi.IsDir() {
		b, err := ioutil.ReadFile(goEnvFile)
		if err != nil {
			return nil, false
		}
		requiredVersion, err := version.NewVersion(string(bytes.TrimSpace(b)))
		if err != nil {
			return nil, false
		}
		return requiredVersion, true
	}
	return nil, false
}
