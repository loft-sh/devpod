package image

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
)

var (
	dockerTagRegexp  = regexp.MustCompile(`^[\w][\w.-]*$`)
	DockerTagMaxSize = 128
)

func GetImage(ctx context.Context, image string) (v1.Image, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	keychain, err := getKeychain(ctx)
	if err != nil {
		return nil, fmt.Errorf("create authentication keychain: %w", err)
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(keychain))
	if err != nil {
		return nil, errors.Wrapf(err, "retrieve image %s", image)
	}

	return img, err
}

func GetImageForArch(ctx context.Context, image, arch string) (v1.Image, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	keychain, err := getKeychain(ctx)
	if err != nil {
		return nil, fmt.Errorf("create authentication keychain: %w", err)
	}

	remoteOptions := []remote.Option{
		remote.WithAuthFromKeychain(keychain),
		remote.WithPlatform(v1.Platform{Architecture: arch, OS: "linux"}),
	}

	img, err := remote.Image(ref, remoteOptions...)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieve image %s", image)
	}

	return img, err
}

func CheckPushPermissions(image string) error {
	ref, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	err = remote.CheckPushPermission(ref, authn.DefaultKeychain, http.DefaultTransport)
	if err != nil {
		return err
	}

	return nil
}

func GetImageConfig(ctx context.Context, image string) (*v1.ConfigFile, v1.Image, error) {
	img, err := GetImage(ctx, image)
	if err != nil {
		return nil, nil, err
	}

	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, nil, errors.Wrap(err, "config file")
	}

	return configFile, img, nil
}

func ValidateTags(tags []string) error {
	for _, tag := range tags {
		if !IsValidDockerTag(tag) {
			return fmt.Errorf(`%q is not a valid docker tag`, tag)
		}
	}
	return nil
}

func IsValidDockerTag(tag string) bool {
	return shouldNotBeSlugged(tag, dockerTagRegexp, DockerTagMaxSize)
}

func shouldNotBeSlugged(data string, regexp *regexp.Regexp, maxSize int) bool {
	return len(data) == 0 || regexp.Match([]byte(data)) && len(data) <= maxSize
}
