package buildkit

import (
	"context"
	"github.com/loft-sh/devpod/pkg/devcontainer/build"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/log"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func Build(ctx context.Context, client *buildkit.Client, writer io.Writer, options *build.BuildOptions, log log.Logger) error {
	dockerConfig, err := docker.LoadDockerConfig()
	if err != nil {
		return err
	}

	// cache from
	cacheFrom, err := ParseCacheEntry(options.CacheFrom)
	if err != nil {
		return err
	}

	// is context stream?
	attachable := []session.Attachable{}
	attachable = append(attachable, authprovider.NewDockerAuthProvider(dockerConfig))

	// create solve options
	solveOptions := buildkit.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": filepath.Base(options.Dockerfile),
			"context":  options.Context,
		},
		Session:      attachable,
		CacheImports: cacheFrom,
	}

	// set options target
	if options.Target != "" {
		solveOptions.FrontendAttrs["target"] = options.Target
	}

	// add context and dockerfile to local dirs
	solveOptions.LocalDirs = map[string]string{}
	solveOptions.LocalDirs["context"] = options.Context
	solveOptions.LocalDirs["dockerfile"] = filepath.Dir(options.Dockerfile)

	// multi contexts
	for k, v := range options.Contexts {
		st, err := os.Stat(v)
		if err != nil {
			return errors.Wrapf(err, "failed to get build context %v", k)
		}
		if !st.IsDir() {
			return errors.Wrapf(syscall.ENOTDIR, "failed to get build context path %v", v)
		}
		localName := k
		if k == "context" || k == "dockerfile" {
			localName = "_" + k // underscore to avoid collisions
		}
		solveOptions.LocalDirs[localName] = v
		solveOptions.FrontendAttrs["context:"+k] = "local:" + localName
	}

	// load?
	if options.Load {
		solveOptions.Exports = append(solveOptions.Exports, buildkit.ExportEntry{
			Type: "moby",
			Attrs: map[string]string{
				"name": strings.Join(options.Images, ","),
			},
		})
	} else if options.Push {
		solveOptions.Exports = append(solveOptions.Exports, buildkit.ExportEntry{
			Type: "image",
			Attrs: map[string]string{
				"name":           strings.Join(options.Images, ","),
				"name-canonical": "",
				"push":           "true",
			},
		})
	}

	// add labels
	for k, v := range options.Labels {
		solveOptions.FrontendAttrs["label:"+k] = v
	}

	// add build args
	for key, value := range options.BuildArgs {
		solveOptions.FrontendAttrs["build-arg:"+key] = value
	}

	pw, err := NewPrinter(ctx, writer)
	if err != nil {
		return err
	}

	// build
	_, err = client.Solve(ctx, nil, solveOptions, pw.Status())
	if err != nil {
		return err
	}

	return nil
}
