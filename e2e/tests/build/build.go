package build

import (
	"context"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/dockerfile"
	dockerdriver "github.com/loft-sh/devpod/pkg/driver/docker"
	"github.com/loft-sh/log"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod build test suite", func() {
	var initialDir string
	var dockerHelper *docker.DockerHelper

	ginkgo.BeforeEach(func() {
		var err error
		initialDir, err = os.Getwd()
		framework.ExpectNoError(err)
		dockerHelper = &docker.DockerHelper{DockerCommand: "docker"}
	})

	ginkgo.It("build docker buildx", func() {
		ctx := context.Background()

		f := framework.NewDefaultFramework(initialDir + "/bin")
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		_ = f.DevPodProviderDelete(ctx, "docker")
		err = f.DevPodProviderAdd(ctx, "docker")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		cfg := getDevcontainerConfig(tempDir)

		dockerfilePath := tempDir + "/.devcontainer/Dockerfile"
		dockerfileContent, err := os.ReadFile(dockerfilePath)
		framework.ExpectNoError(err)
		_, modifiedDockerfileContents, err := dockerfile.EnsureDockerfileHasFinalStageName(string(dockerfileContent), config.DockerfileDefaultTarget)
		framework.ExpectNoError(err)

		prebuildRepo := "test-repo"

		// do the build
		err = f.DevPodBuild(ctx, tempDir, "--force-build", "--platform", "linux/amd64,linux/arm64", "--repository", prebuildRepo, "--skip-push")
		framework.ExpectNoError(err)

		// make sure images are there
		prebuildHash, err := config.CalculatePrebuildHash(cfg, "linux/amd64", "amd64", filepath.Dir(cfg.Origin), dockerfilePath, modifiedDockerfileContents, log.Default)
		framework.ExpectNoError(err)
		_, err = dockerHelper.InspectImage(ctx, prebuildRepo+":"+prebuildHash, false)
		framework.ExpectNoError(err)

		prebuildHash, err = config.CalculatePrebuildHash(cfg, "linux/arm64", "arm64", filepath.Dir(cfg.Origin), dockerfilePath, modifiedDockerfileContents, log.Default)
		framework.ExpectNoError(err)
		_, err = dockerHelper.InspectImage(ctx, prebuildRepo+":"+prebuildHash, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build image without repository specified if skip-push flag is set", func() {
		ctx := context.Background()

		f := framework.NewDefaultFramework(initialDir + "/bin")
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		_ = f.DevPodProviderDelete(ctx, "docker")
		err = f.DevPodProviderAdd(ctx, "docker")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		cfg := getDevcontainerConfig(tempDir)

		dockerfilePath := tempDir + "/.devcontainer/Dockerfile"
		dockerfileContent, err := os.ReadFile(dockerfilePath)
		framework.ExpectNoError(err)
		_, modifiedDockerfileContents, err := dockerfile.EnsureDockerfileHasFinalStageName(string(dockerfileContent), config.DockerfileDefaultTarget)
		framework.ExpectNoError(err)

		// do the build
		err = f.DevPodBuild(ctx, tempDir, "--skip-push")
		framework.ExpectNoError(err)

		// make sure images are there
		prebuildHash, err := config.CalculatePrebuildHash(cfg, "linux/amd64", "amd64", filepath.Dir(cfg.Origin), dockerfilePath, modifiedDockerfileContents, log.Default)
		framework.ExpectNoError(err)
		_, err = dockerHelper.InspectImage(ctx, dockerdriver.GetImageName(tempDir, prebuildHash), false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("build docker internal buildkit", func() {
		ctx := context.Background()

		f := framework.NewDefaultFramework(initialDir + "/bin")
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		_ = f.DevPodProviderDelete(ctx, "docker")
		err = f.DevPodProviderAdd(ctx, "docker")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		cfg := getDevcontainerConfig(tempDir)

		dockerfilePath := tempDir + "/.devcontainer/Dockerfile"
		dockerfileContent, err := os.ReadFile(dockerfilePath)
		framework.ExpectNoError(err)
		_, modifiedDockerfileContents, err := dockerfile.EnsureDockerfileHasFinalStageName(string(dockerfileContent), config.DockerfileDefaultTarget)
		framework.ExpectNoError(err)

		prebuildRepo := "test-repo"

		// do the build
		err = f.DevPodBuild(ctx, tempDir, "--force-build", "--force-internal-buildkit", "--repository", prebuildRepo, "--skip-push")
		framework.ExpectNoError(err)

		// make sure images are there
		prebuildHash, err := config.CalculatePrebuildHash(cfg, "linux/amd64", "amd64", filepath.Dir(cfg.Origin), dockerfilePath, modifiedDockerfileContents, log.Default)
		framework.ExpectNoError(err)

		_, err = dockerHelper.InspectImage(ctx, prebuildRepo+":"+prebuildHash, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("build kubernetes buildkit", func() {
		ctx := context.Background()

		f := framework.NewDefaultFramework(initialDir + "/bin")
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/kubernetes")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		_ = f.DevPodProviderDelete(ctx, "kubernetes")
		err = f.DevPodProviderAdd(ctx, "kubernetes")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(context.Background(), "kubernetes", "-o", "BUILD_REPOSITORY=test-repo", "-o", "KUBERNETES_NAMESPACE=devpod")
		framework.ExpectNoError(err)

		// do the build
		err = f.DevPodBuild(ctx, tempDir, "--force-build", "--repository", "test-repo", "--skip-push")
		framework.ExpectNoError(err)
	})
})

func getDevcontainerConfig(dir string) *config.DevContainerConfig {
	return &config.DevContainerConfig{
		DevContainerConfigBase: config.DevContainerConfigBase{
			Name: "Build Example",
		},
		DevContainerActions: config.DevContainerActions{},
		NonComposeBase:      config.NonComposeBase{},
		ImageContainer:      config.ImageContainer{},
		ComposeContainer:    config.ComposeContainer{},
		DockerfileContainer: config.DockerfileContainer{
			Dockerfile: "Dockerfile",
			Context:    "",
			Build:      config.ConfigBuildOptions{},
		},
		Origin: dir + "/.devcontainer/devcontainer.json",
	}
}
