package releases

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/go-version"
	rjson "github.com/hashicorp/hc-install/internal/releasesjson"
	"github.com/hashicorp/hc-install/internal/validators"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/src"
)

// Versions allows listing all versions of a product
// which match Constraints
type Versions struct {
	Product     product.Product
	Constraints version.Constraints

	ListTimeout time.Duration

	// Install represents configuration for installation of any listed version
	Install InstallationOptions
}

type InstallationOptions struct {
	Timeout time.Duration
	Dir     string

	SkipChecksumVerification bool

	// ArmoredPublicKey is a public PGP key in ASCII/armor format to use
	// instead of built-in pubkey to verify signature of downloaded checksums
	// during installation
	ArmoredPublicKey string
}

func (v *Versions) List(ctx context.Context) ([]src.Source, error) {
	if !validators.IsProductNameValid(v.Product.Name) {
		return nil, fmt.Errorf("invalid product name: %q", v.Product.Name)
	}

	timeout := defaultListTimeout
	if v.ListTimeout > 0 {
		timeout = v.ListTimeout
	}
	ctx, cancelFunc := context.WithTimeout(ctx, timeout)
	defer cancelFunc()

	r := rjson.NewReleases()
	pvs, err := r.ListProductVersions(ctx, v.Product.Name)
	if err != nil {
		return nil, err
	}

	versions := pvs.AsSlice()
	sort.Stable(versions)

	installables := make([]src.Source, 0)
	for _, pv := range versions {
		if !v.Constraints.Check(pv.Version) {
			// skip version which doesn't match constraint
			continue
		}

		ev := &ExactVersion{
			Product:    v.Product,
			Version:    pv.Version,
			InstallDir: v.Install.Dir,
			Timeout:    v.Install.Timeout,

			ArmoredPublicKey:         v.Install.ArmoredPublicKey,
			SkipChecksumVerification: v.Install.SkipChecksumVerification,
		}

		installables = append(installables, ev)
	}

	return installables, nil
}
