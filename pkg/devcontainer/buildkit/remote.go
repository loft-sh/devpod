package buildkit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/loft-sh/devpod/pkg/devcontainer/build"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/sirupsen/logrus"
	"github.com/tonistiigi/fsutil"
)

func BuildRemote(
	ctx context.Context,
	prebuildHash string,
	parsedConfig *config.SubstitutedConfig,
	extendedBuildInfo *feature.ExtendedBuildInfo,
	dockerfilePath,
	dockerfileContent string,
	localWorkspaceFolder string,
	options provider.BuildOptions,
	targetArch string,
	log log.Logger,
) (*config.BuildInfo, error) {
	if options.NoBuild {
		return nil, fmt.Errorf("you cannot build in this mode. Please run 'devpod up' to rebuild the container")
	}
	if !options.CLIOptions.Platform.Enabled {
		return nil, errors.New("remote builds are only supported in DevPod Pro")
	}
	if options.CLIOptions.Platform.BuilderAddress == "" {
		return nil, errors.New("builder address is required to build image remotely")
	}
	if options.CLIOptions.Platform.BuildRegistry == "" && !options.SkipPush {
		return nil, errors.New("remote builds require a registry to be provided")
	}

	// Do we already have image?

	// initialize remote buildkit client
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	c, err := client.New(timeoutCtx, options.CLIOptions.Platform.BuilderAddress)
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	defer c.Close()

	info, err := c.Info(timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("get info: %w", err)
	}

	imageName := options.CLIOptions.Platform.BuildRegistry + "/" + build.GetImageName(localWorkspaceFolder, prebuildHash)

	// check push permissions before building
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, fmt.Errorf("parse image name %s: %w", imageName, err)
	}
	keychain, err := image.GetKeychain(ctx)
	if err != nil {
		return nil, fmt.Errorf("get docker auth keychain: %w", err)
	}
	imageDetails, err := getImageDetails(ctx, ref, targetArch, keychain)
	// we can return early if we find an existing image with the exact configuration in the repository
	if err == nil {
		log.Infof("Found existing image %s, skipping build", imageName)
		return &config.BuildInfo{
			ImageDetails:  imageDetails,
			ImageMetadata: extendedBuildInfo.MetadataConfig,
			ImageName:     imageName,
			PrebuildHash:  prebuildHash,
			RegistryCache: options.RegistryCache,
			Tags:          options.Tag,
		}, nil
	}

	// check push permissions early
	err = remote.CheckPushPermission(ref, keychain, http.DefaultTransport)
	if err != nil {
		return nil, fmt.Errorf("pushing %s is not allowed: %w", err)
	}

	// resolve credentials for registry
	auth, err := keychain.Resolve(ref.Context())
	if err != nil {
		return nil, fmt.Errorf("get authentication for %s: %w", ref.Context().String(), err)
	}
	authConfig, err := auth.Authorization()
	if err != nil {
		return nil, fmt.Errorf("get auth config for %s: %w", ref.Context().String(), err)
	}

	registry := ref.Context().Registry.RegistryStr()
	session := []session.Attachable{
		authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
			ConfigFile: &configfile.ConfigFile{
				AuthConfigs: map[string]types.AuthConfig{
					registry: types.AuthConfig{
						Username:      authConfig.Username,
						Auth:          authConfig.Auth,
						Password:      authConfig.Password,
						IdentityToken: authConfig.IdentityToken,
						RegistryToken: authConfig.RegistryToken,
					},
				},
			},
		}),
	}

	buildOptions, err := build.NewOptions(dockerfilePath, dockerfileContent, parsedConfig, extendedBuildInfo, imageName, options, prebuildHash)
	if err != nil {
		return nil, fmt.Errorf("create build buildOptions: %w", err)
	}

	// cache from
	cacheFrom, err := ParseCacheEntry(buildOptions.CacheFrom)
	if err != nil {
		return nil, err
	}
	cacheTo, err := ParseCacheEntry(buildOptions.CacheTo)
	if err != nil {
		return nil, err
	}

	dockerfileDir := filepath.Dir(buildOptions.Dockerfile)
	localMounts := map[string]fsutil.FS{}
	dockerfileMount, err := fsutil.NewFS(dockerfileDir)
	if err != nil {
		return nil, fmt.Errorf("create local dockerfile mount: %w", err)
	}
	localMounts["dockerfile"] = dockerfileMount
	contextMount, err := fsutil.NewFS(buildOptions.Context)
	if err != nil {
		return nil, fmt.Errorf("create local context mount: %w", err)
	}
	localMounts["context"] = contextMount

	// create solve options
	solveOptions := client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": filepath.Base(buildOptions.Dockerfile),
			"context":  buildOptions.Context,
		},
		LocalMounts:  localMounts,
		Session:      session,
		CacheImports: cacheFrom,
		CacheExports: cacheTo,
	}

	// set buildOptions target
	if buildOptions.Target != "" {
		solveOptions.FrontendAttrs["target"] = buildOptions.Target
	}

	// add platforms
	if options.Platform != "" {
		solveOptions.FrontendAttrs["platform"] = options.Platform
	} else if targetArch != "" {
		solveOptions.FrontendAttrs["platform"] = "linux/" + targetArch
	}

	// multi contexts
	for k, v := range buildOptions.Contexts {
		st, err := os.Stat(v)
		if err != nil {
			return nil, fmt.Errorf("get build context %v: %w", k, err)
		}
		if !st.IsDir() {
			return nil, fmt.Errorf("build context '%s' is not a directory", v)
		}
		localName := k
		if k == "context" || k == "dockerfile" {
			localName = "_" + k // underscore to avoid collisions
		}

		solveOptions.LocalMounts[localName], err = fsutil.NewFS(v)
		if err != nil {
			return nil, fmt.Errorf("create local mount for %s at %s: %w", localName, v, err)
		}

		solveOptions.FrontendAttrs["context:"+k] = "local:" + localName
	}

	push := "true"
	if options.SkipPush {
		push = "false"
	}
	solveOptions.Exports = append(solveOptions.Exports, client.ExportEntry{
		Type: client.ExporterImage,
		Attrs: map[string]string{
			string(exptypes.OptKeyName): strings.Join(buildOptions.Images, ","),
			string(exptypes.OptKeyPush): push,
		},
	})

	// add labels
	for k, v := range buildOptions.Labels {
		solveOptions.FrontendAttrs["label:"+k] = v
	}

	// add build args
	for key, value := range buildOptions.BuildArgs {
		solveOptions.FrontendAttrs["build-arg:"+key] = value
	}

	log.Infof("Start building %s using platform builder (%s)", strings.Join(buildOptions.Images, ","), info.BuildkitVersion.Version)

	// TODO: Writer should be async to prevent blocking while waiting for tunnel response
	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()
	pw, err := NewPrinter(ctx, writer)
	if err != nil {
		return nil, err
	}

	_, err = c.Solve(ctx, nil, solveOptions, pw.Status())
	if err != nil {
		return nil, err
	}

	imageDetails, err = getImageDetails(ctx, ref, targetArch, keychain)
	if err != nil {
		return nil, fmt.Errorf("get image details: %w", err)
	}
	return &config.BuildInfo{
		ImageDetails:  imageDetails,
		ImageMetadata: extendedBuildInfo.MetadataConfig,
		ImageName:     imageName,
		PrebuildHash:  prebuildHash,
		RegistryCache: options.RegistryCache,
		Tags:          options.Tag,
	}, nil
}

func getImageDetails(ctx context.Context, ref name.Reference, targetArch string, keychain authn.Keychain) (*config.ImageDetails, error) {
	remoteImage, err := remote.Image(ref,
		remote.WithAuthFromKeychain(keychain),
		remote.WithPlatform(v1.Platform{Architecture: targetArch, OS: "linux"}),
	)
	if err != nil {
		return nil, err
	}
	imageConfig, err := remoteImage.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("get image config file: %w", err)
	}

	imageDetails := &config.ImageDetails{
		ID: ref.Name(),
		Config: config.ImageDetailsConfig{
			User:       imageConfig.Config.User,
			Env:        imageConfig.Config.Env,
			Labels:     imageConfig.Config.Labels,
			Entrypoint: imageConfig.Config.Entrypoint,
			Cmd:        imageConfig.Config.Cmd,
		},
	}

	return imageDetails, nil
}
