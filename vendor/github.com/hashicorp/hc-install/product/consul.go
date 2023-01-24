package product

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/internal/build"
)

var consulVersionOutputRe = regexp.MustCompile(`Consul ` + simpleVersionRe)

var (
	v1_18 = version.Must(version.NewVersion("1.18"))
)

var Consul = Product{
	Name: "consul",
	BinaryName: func() string {
		if runtime.GOOS == "windows" {
			return "consul.exe"
		}
		return "consul"
	},
	GetVersion: func(ctx context.Context, path string) (*version.Version, error) {
		cmd := exec.CommandContext(ctx, path, "version")

		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		stdout := strings.TrimSpace(string(out))

		submatches := consulVersionOutputRe.FindStringSubmatch(stdout)
		if len(submatches) != 2 {
			return nil, fmt.Errorf("unexpected number of version matches %d for %s", len(submatches), stdout)
		}
		v, err := version.NewVersion(submatches[1])
		if err != nil {
			return nil, fmt.Errorf("unable to parse version %q: %w", submatches[1], err)
		}

		return v, err
	},
	BuildInstructions: &BuildInstructions{
		GitRepoURL:    "https://github.com/hashicorp/consul.git",
		PreCloneCheck: &build.GoIsInstalled{},
		Build:         &build.GoBuild{Version: v1_18},
	},
}
