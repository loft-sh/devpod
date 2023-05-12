package image

import (
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
)

func GetTag(image string) string {
	ref, err := name.ParseReference(image)
	if err != nil {
		return ""
	}

	tag, ok := ref.(name.Tag)
	if ok {
		return tag.TagStr()
	}

	return ""
}

func GetImage(image string) (v1.Image, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, err
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

func GetImageConfig(image string) (*v1.ConfigFile, v1.Image, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parse reference %s", image)
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "retrieve image %s", image)
	}

	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, nil, errors.Wrap(err, "config file")
	}

	return configFile, img, nil
}
