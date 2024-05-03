package image

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
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
