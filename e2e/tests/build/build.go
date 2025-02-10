package build

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/dockerfile"
	dockerdriver "github.com/loft-sh/devpod/pkg/driver/docker"
	"github.com/loft-sh/log"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod build test suite", func() {
	ginkgo.Context("testing build", ginkgo.Label("build"), ginkgo.Ordered, func() {
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

			// parse the dockerfile
			file, err := dockerfile.Parse(modifiedDockerfileContents)
			framework.ExpectNoError(err)
			info := &config.ImageBuildInfo{Dockerfile: file}

			// make sure images are there
			prebuildHash, err := config.CalculatePrebuildHash(cfg, "linux/amd64", "amd64", filepath.Dir(cfg.Origin), dockerfilePath, modifiedDockerfileContents, info, log.Default)
			framework.ExpectNoError(err)
			_, err = dockerHelper.InspectImage(ctx, prebuildRepo+":"+prebuildHash, false)
			framework.ExpectNoError(err)

			prebuildHash, err = config.CalculatePrebuildHash(cfg, "linux/arm64", "arm64", filepath.Dir(cfg.Origin), dockerfilePath, modifiedDockerfileContents, info, log.Default)
			framework.ExpectNoError(err)
			_, err = dockerHelper.InspectImage(ctx, prebuildRepo+":"+prebuildHash, false)
			framework.ExpectNoError(err)
			details, err := dockerHelper.InspectImage(ctx, prebuildRepo+":"+prebuildHash, false)
			framework.ExpectNoError(err)
			framework.ExpectEqual(details.Config.Labels["test"], "VALUE", "should contain test label")
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
			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			cfg := getDevcontainerConfig(tempDir)

			dockerfilePath := tempDir + "/.devcontainer/Dockerfile"
			dockerfileContent, err := os.ReadFile(dockerfilePath)
			framework.ExpectNoError(err)
			_, modifiedDockerfileContents, err := dockerfile.EnsureDockerfileHasFinalStageName(string(dockerfileContent), config.DockerfileDefaultTarget)
			framework.ExpectNoError(err)

			// do the build
			err = f.DevPodBuild(ctx, tempDir, "--skip-push")
			framework.ExpectNoError(err)

			// parse the dockerfile
			file, err := dockerfile.Parse(modifiedDockerfileContents)
			framework.ExpectNoError(err)
			info := &config.ImageBuildInfo{Dockerfile: file}

			// make sure images are there
			prebuildHash, err := config.CalculatePrebuildHash(cfg, "linux/amd64", "amd64", filepath.Dir(cfg.Origin), dockerfilePath, modifiedDockerfileContents, info, log.Default)
			framework.ExpectNoError(err)
			_, err = dockerHelper.InspectImage(ctx, dockerdriver.GetImageName(tempDir, prebuildHash), false)
			framework.ExpectNoError(err)
		})

		ginkgo.It("should build the image of the referenced service from the docker compose file", func() {
			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")
			tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker-compose")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

			_ = f.DevPodProviderDelete(ctx, "docker")
			err = f.DevPodProviderAdd(ctx, "docker")
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(context.Background(), "docker")
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			prebuildRepo := "test-repo"

			// do the build
			err = f.DevPodBuild(ctx, tempDir, "--repository", prebuildRepo, "--skip-push")
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

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

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

			// parse the dockerfile
			file, err := dockerfile.Parse(modifiedDockerfileContents)
			framework.ExpectNoError(err)
			info := &config.ImageBuildInfo{Dockerfile: file}

			// make sure images are there
			prebuildHash, err := config.CalculatePrebuildHash(cfg, "linux/amd64", "amd64", filepath.Dir(cfg.Origin), dockerfilePath, modifiedDockerfileContents, info, log.Default)
			framework.ExpectNoError(err)

			_, err = dockerHelper.InspectImage(ctx, prebuildRepo+":"+prebuildHash, false)
			framework.ExpectNoError(err)
		})

		ginkgo.It("build kubernetes dockerless", func() {
			// skip windows for now
			if runtime.GOOS == "windows" {
				return
			}

			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")
			tempDir, err := framework.CopyToTempDir("tests/build/testdata/kubernetes")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

			_ = f.DevPodProviderDelete(ctx, "kubernetes")
			err = f.DevPodProviderAdd(ctx, "kubernetes")
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(context.Background(), "kubernetes", "-o", "KUBERNETES_NAMESPACE=devpod")
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// do the up
			err = f.DevPodUp(ctx, tempDir)
			framework.ExpectNoError(err)

			// check if ssh works
			out, err := f.DevPodSSH(ctx, tempDir, "echo -n $MY_TEST")
			framework.ExpectNoError(err)
			framework.ExpectEqual(out, "test456", "should contain my-test")
		})

		ginkgo.It("rebuild kubernetes dockerless", func() {
			// skip windows for now
			if runtime.GOOS == "windows" {
				return
			}

			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")
			tempDir, err := framework.CopyToTempDir("tests/build/testdata/kubernetes")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

			_ = f.DevPodProviderDelete(ctx, "kubernetes")
			err = f.DevPodProviderAdd(ctx, "kubernetes")
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(context.Background(), "kubernetes", "-o", "KUBERNETES_NAMESPACE=devpod")
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// do the up
			err = f.DevPodUp(ctx, tempDir)
			framework.ExpectNoError(err)

			// create files in root and in workspace, after create we expect data to still be there
			_, err = f.DevPodSSH(ctx, tempDir, "touch /workspaces/"+filepath.Base(tempDir)+"/DATA")
			framework.ExpectNoError(err)
			_, err = f.DevPodSSH(ctx, tempDir, "touch /ROOTFS")
			framework.ExpectNoError(err)

			// recreate
			err = f.DevPodUpRecreate(ctx, tempDir)
			framework.ExpectNoError(err)

			// this should still be there
			_, err = f.DevPodSSH(ctx, tempDir, "ls /workspaces/"+filepath.Base(tempDir)+"/DATA")
			framework.ExpectNoError(err)
			// this should fail! because --recreare should trigger a new build, so a new rootfs
			_, err = f.DevPodSSH(ctx, tempDir, "ls /ROOTFS")
			framework.ExpectError(err)
		})

		ginkgo.It("reset kubernetes dockerless", func() {
			// skip windows for now
			if runtime.GOOS == "windows" {
				return
			}

			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")
			tempDir, err := framework.CopyToTempDir("tests/build/testdata/kubernetes")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

			_ = f.DevPodProviderDelete(ctx, "kubernetes")
			err = f.DevPodProviderAdd(ctx, "kubernetes")
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(context.Background(), "kubernetes", "-o", "KUBERNETES_NAMESPACE=devpod")
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// do the up
			err = f.DevPodUp(ctx, tempDir)
			framework.ExpectNoError(err)

			// create files in root and in workspace, after create we expect data to still be there
			_, err = f.DevPodSSH(ctx, tempDir, "touch /workspaces/"+filepath.Base(tempDir)+"/DATA")
			framework.ExpectNoError(err)
			_, err = f.DevPodSSH(ctx, tempDir, "touch /ROOTFS")
			framework.ExpectNoError(err)

			// recreate
			err = f.DevPodUpReset(ctx, tempDir)
			framework.ExpectNoError(err)

			// this should fail! because --reset should trigger a new git clone
			_, err = f.DevPodSSH(ctx, tempDir, "ls /workspaces/"+filepath.Base(tempDir)+"/DATA")
			framework.ExpectNoError(err)
			// this should fail! because --recreare should trigger a new build, so a new rootfs
			_, err = f.DevPodSSH(ctx, tempDir, "ls /ROOTFS")
			framework.ExpectError(err)
		})
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
			Build: &config.ConfigBuildOptions{
				Dockerfile: "Dockerfile",
				Context:    ".",
				Options:    []string{"--label=test=VALUE"},
			},
		},
		Origin: dir + "/.devcontainer/devcontainer.json",
	}
}
