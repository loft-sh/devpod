package releasesjson

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/internal/httpclient"
)

const defaultBaseURL = "https://releases.hashicorp.com"

// Product is a top-level product like "Consul" or "Nomad". A Product may have
// one or more versions.
type Product struct {
	Name     string             `json:"name"`
	Versions ProductVersionsMap `json:"versions"`
}

type ProductBuilds []*ProductBuild

func (pbs ProductBuilds) FilterBuild(os string, arch string, suffix string) (*ProductBuild, bool) {
	for _, pb := range pbs {
		if pb.OS == os && pb.Arch == arch && strings.HasSuffix(pb.Filename, suffix) {
			return pb, true
		}
	}
	return nil, false
}

// ProductBuild is an OS/arch-specific representation of a product. This is the
// actual file that a user would download, like "consul_0.5.1_linux_amd64".
type ProductBuild struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

type Releases struct {
	logger  *log.Logger
	BaseURL string
}

func NewReleases() *Releases {
	return &Releases{
		logger:  log.New(ioutil.Discard, "", 0),
		BaseURL: defaultBaseURL,
	}
}

func (r *Releases) SetLogger(logger *log.Logger) {
	r.logger = logger
}

func (r *Releases) ListProductVersions(ctx context.Context, productName string) (ProductVersionsMap, error) {
	client := httpclient.NewHTTPClient()

	productIndexURL := fmt.Sprintf("%s/%s/index.json",
		r.BaseURL,
		url.PathEscape(productName))
	r.logger.Printf("requesting versions from %s", productIndexURL)

	resp, err := client.Get(productIndexURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to obtain product versions from %q: %s ",
			productIndexURL, resp.Status)
	}

	contentType := resp.Header.Get("content-type")
	if contentType != "application/json" && contentType != "application/vnd+hashicorp.releases-api.v0+json" {
		return nil, fmt.Errorf("unexpected Content-Type: %q", contentType)
	}

	defer resp.Body.Close()

	r.logger.Printf("received %s", resp.Status)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	p := Product{}
	err = json.Unmarshal(body, &p)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal: %q",
			err, string(body))
	}

	for rawVersion := range p.Versions {
		v, err := version.NewVersion(rawVersion)
		if err != nil {
			// remove unparseable version
			delete(p.Versions, rawVersion)
			continue
		}

		if ok, _ := versionIsSupported(v); !ok {
			// Remove (currently unsupported) enterprise
			// version and any other "custom" build
			delete(p.Versions, rawVersion)
			continue
		}

		p.Versions[rawVersion].Version = v
	}

	return p.Versions, nil
}

func (r *Releases) GetProductVersion(ctx context.Context, product string, version *version.Version) (*ProductVersion, error) {
	if ok, err := versionIsSupported(version); !ok {
		return nil, fmt.Errorf("%s: %w", product, err)
	}

	client := httpclient.NewHTTPClient()

	indexURL := fmt.Sprintf("%s/%s/%s/index.json",
		r.BaseURL,
		url.PathEscape(product),
		url.PathEscape(version.String()))
	r.logger.Printf("requesting version from %s", indexURL)

	resp, err := client.Get(indexURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to obtain product version from %q: %s ",
			indexURL, resp.Status)
	}

	contentType := resp.Header.Get("content-type")
	if contentType != "application/json" && contentType != "application/vnd+hashicorp.releases-api.v0+json" {
		return nil, fmt.Errorf("unexpected Content-Type: %q", contentType)
	}

	defer resp.Body.Close()

	r.logger.Printf("received %s", resp.Status)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	pv := &ProductVersion{}
	err = json.Unmarshal(body, pv)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal response: %q",
			err, string(body))
	}

	return pv, nil
}

func versionIsSupported(v *version.Version) (bool, error) {
	isSupported := v.Metadata() == ""
	if !isSupported {
		return false, fmt.Errorf("cannot obtain %s (enterprise versions are not supported)",
			v.String())
	}
	return true, nil
}
