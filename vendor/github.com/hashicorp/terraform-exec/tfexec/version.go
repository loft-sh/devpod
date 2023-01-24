package tfexec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
)

var (
	tf0_4_1  = version.Must(version.NewVersion("0.4.1"))
	tf0_5_0  = version.Must(version.NewVersion("0.5.0"))
	tf0_6_13 = version.Must(version.NewVersion("0.6.13"))
	tf0_7_7  = version.Must(version.NewVersion("0.7.7"))
	tf0_8_0  = version.Must(version.NewVersion("0.8.0"))
	tf0_10_0 = version.Must(version.NewVersion("0.10.0"))
	tf0_12_0 = version.Must(version.NewVersion("0.12.0"))
	tf0_13_0 = version.Must(version.NewVersion("0.13.0"))
	tf0_14_0 = version.Must(version.NewVersion("0.14.0"))
	tf0_15_0 = version.Must(version.NewVersion("0.15.0"))
	tf0_15_2 = version.Must(version.NewVersion("0.15.2"))
	tf1_1_0  = version.Must(version.NewVersion("1.1.0"))
)

// Version returns structured output from the terraform version command including both the Terraform CLI version
// and any initialized provider versions. This will read cached values when present unless the skipCache parameter
// is set to true.
func (tf *Terraform) Version(ctx context.Context, skipCache bool) (tfVersion *version.Version, providerVersions map[string]*version.Version, err error) {
	tf.versionLock.Lock()
	defer tf.versionLock.Unlock()

	if tf.execVersion == nil || skipCache {
		tf.execVersion, tf.provVersions, err = tf.version(ctx)
		if err != nil {
			return nil, nil, err
		}
	}

	return tf.execVersion, tf.provVersions, nil
}

// version does not use the locking on the Terraform instance and should probably not be used directly, prefer Version.
func (tf *Terraform) version(ctx context.Context) (*version.Version, map[string]*version.Version, error) {
	versionCmd := tf.buildTerraformCmd(ctx, nil, "version", "-json")

	var outBuf bytes.Buffer
	versionCmd.Stdout = &outBuf

	err := tf.runTerraformCmd(ctx, versionCmd)
	if err != nil {
		return nil, nil, err
	}

	tfVersion, providerVersions, err := parseJsonVersionOutput(outBuf.Bytes())
	if err != nil {
		if _, ok := err.(*json.SyntaxError); ok {
			return tf.versionFromPlaintext(ctx)
		}
	}

	return tfVersion, providerVersions, err
}

func parseJsonVersionOutput(stdout []byte) (*version.Version, map[string]*version.Version, error) {
	var out tfjson.VersionOutput
	err := json.Unmarshal(stdout, &out)
	if err != nil {
		return nil, nil, err
	}

	tfVersion, err := version.NewVersion(out.Version)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse version %q: %w", out.Version, err)
	}

	providerVersions := make(map[string]*version.Version, 0)
	for provider, versionStr := range out.ProviderSelections {
		v, err := version.NewVersion(versionStr)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse %q version %q: %w",
				provider, versionStr, err)
		}
		providerVersions[provider] = v
	}

	return tfVersion, providerVersions, nil
}

func (tf *Terraform) versionFromPlaintext(ctx context.Context) (*version.Version, map[string]*version.Version, error) {
	versionCmd := tf.buildTerraformCmd(ctx, nil, "version")

	var outBuf strings.Builder
	versionCmd.Stdout = &outBuf

	err := tf.runTerraformCmd(ctx, versionCmd)
	if err != nil {
		return nil, nil, err
	}

	tfVersion, providerVersions, err := parsePlaintextVersionOutput(outBuf.String())
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse version: %w", err)
	}

	return tfVersion, providerVersions, nil
}

var (
	simpleVersionRe = `v?(?P<version>[0-9]+(?:\.[0-9]+)*(?:-[A-Za-z0-9\.]+)?)`

	versionOutputRe         = regexp.MustCompile(`Terraform ` + simpleVersionRe)
	providerVersionOutputRe = regexp.MustCompile(`(\n\+ provider[\. ](?P<name>\S+) ` + simpleVersionRe + `)`)
)

func parsePlaintextVersionOutput(stdout string) (*version.Version, map[string]*version.Version, error) {
	stdout = strings.TrimSpace(stdout)

	submatches := versionOutputRe.FindStringSubmatch(stdout)
	if len(submatches) != 2 {
		return nil, nil, fmt.Errorf("unexpected number of version matches %d for %s", len(submatches), stdout)
	}
	v, err := version.NewVersion(submatches[1])
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse version %q: %w", submatches[1], err)
	}

	allSubmatches := providerVersionOutputRe.FindAllStringSubmatch(stdout, -1)
	provV := map[string]*version.Version{}

	for _, submatches := range allSubmatches {
		if len(submatches) != 4 {
			return nil, nil, fmt.Errorf("unexpected number of provider version matches %d for %s", len(submatches), stdout)
		}

		v, err := version.NewVersion(submatches[3])
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse provider version %q: %w", submatches[3], err)
		}

		provV[submatches[2]] = v
	}

	return v, provV, err
}

func errorVersionString(v *version.Version) string {
	if v == nil {
		return "-"
	}
	return v.String()
}

// compatible asserts compatibility of the cached terraform version with the executable, and returns a well known error if not.
func (tf *Terraform) compatible(ctx context.Context, minInclusive *version.Version, maxExclusive *version.Version) error {
	tfv, _, err := tf.Version(ctx, false)
	if err != nil {
		return err
	}
	if ok := versionInRange(tfv, minInclusive, maxExclusive); !ok {
		return &ErrVersionMismatch{
			MinInclusive: errorVersionString(minInclusive),
			MaxExclusive: errorVersionString(maxExclusive),
			Actual:       errorVersionString(tfv),
		}
	}

	return nil
}

func stripPrereleaseAndMeta(v *version.Version) *version.Version {
	if v == nil {
		return nil
	}
	segs := []string{}
	for _, s := range v.Segments() {
		segs = append(segs, strconv.Itoa(s))
	}
	vs := strings.Join(segs, ".")
	clean, _ := version.NewVersion(vs)
	return clean
}

// versionInRange checks compatibility of the Terraform version. The minimum is inclusive and the max
// is exclusive, equivalent to min <= expected version < max.
//
// Pre-release information is ignored for comparison.
func versionInRange(tfv *version.Version, minInclusive *version.Version, maxExclusive *version.Version) bool {
	if minInclusive == nil && maxExclusive == nil {
		return true
	}
	tfv = stripPrereleaseAndMeta(tfv)
	minInclusive = stripPrereleaseAndMeta(minInclusive)
	maxExclusive = stripPrereleaseAndMeta(maxExclusive)
	if minInclusive != nil && !tfv.GreaterThanOrEqual(minInclusive) {
		return false
	}
	if maxExclusive != nil && !tfv.LessThan(maxExclusive) {
		return false
	}

	return true
}
