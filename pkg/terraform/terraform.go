package terraform

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"os"
	"runtime"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pkg/errors"
	"path/filepath"
)

var (
	terraformVersion = "1.3.7"
)

func InstallTerraform(ctx context.Context) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	installDir := filepath.Join(configDir, "terraform")
	err = os.MkdirAll(installDir, 0755)
	if err != nil {
		return "", err
	}

	// get terraform binary path
	terraformPath := filepath.Join(installDir, "terraform")
	if runtime.GOOS == "windows" {
		terraformPath += ".exe"
	}

	// create a new terraform client
	tf, err := tfexec.NewTerraform(".", terraformPath)
	if err != nil {
		return "", err
	}

	// get version
	tfVersion, _, err := tf.Version(ctx, true)
	if tfVersion.String() == terraformVersion && err == nil {
		return terraformPath, nil
	}

	// install if not found
	installer := &releases.ExactVersion{
		Product:    product.Terraform,
		InstallDir: installDir,
		Version:    version.Must(version.NewVersion(terraformVersion)),
	}

	execPath, err := installer.Install(ctx)
	if err != nil {
		return "", errors.Wrap(err, "install terraform")
	}

	return execPath, nil
}
