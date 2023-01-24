package build

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"
)

// GetGoVersion obtains version of locally installed Go via "go version"
func GetGoVersion(ctx context.Context) (*version.Version, error) {
	cmd := exec.CommandContext(ctx, "go", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("unable to build: %w\n%s", err, out)
	}

	output := strings.TrimSpace(string(out))

	// e.g. "go version go1.15"
	re := regexp.MustCompile(`^go version go([0-9.]+)\s+`)
	matches := re.FindStringSubmatch(output)
	if len(matches) != 2 {
		return nil, fmt.Errorf("unexpected go version output: %q", output)
	}

	rawGoVersion := matches[1]
	v, err := version.NewVersion(rawGoVersion)
	if err != nil {
		return nil, fmt.Errorf("unexpected go version output: %w", err)
	}

	return v, nil
}
