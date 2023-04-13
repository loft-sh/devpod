package up

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devpod/pkg/compose"
	docker "github.com/loft-sh/devpod/pkg/docker"
	"github.com/onsi/gomega"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod up test suite", func() {
	var dockerHelper *docker.DockerHelper
	var composeHelper *compose.ComposeHelper
	var initialDir string

	ginkgo.BeforeEach(func() {
		var err error
		initialDir, err = os.Getwd()
		framework.ExpectNoError(err)

		dockerHelper = &docker.DockerHelper{DockerCommand: "docker"}
		composeHelper, err = compose.NewComposeHelper("", dockerHelper)
		framework.ExpectNoError(err)
	})

	ginkgo.Context("using local provider", func() {
		ginkgo.Context("with docker", func() {
			ginkgo.It("should start a new workspace with existing image", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/local-test")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(30*time.Second))
		})

		ginkgo.Context("with docker-compose", func() {
			ginkgo.It("should start a new workspace with root folder configuration", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(len(containerDetail.Mounts)).To(gomega.Equal(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.Equal(true))
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with sub-folder configuration", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-subfolder")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(len(containerDetail.Mounts)).To(gomega.Equal(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.Equal(true))
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with multiple services", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-multiple-services")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				appIDs, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(appIDs)).To(gomega.Equal(1), "app container to be created")

				dbIDs, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(dbIDs)).To(gomega.Equal(1), "db container to be created")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with .devcontainer docker-compose overrides", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-overrides")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(len(containerDetail.Mounts)).To(gomega.Equal(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.Equal(true))
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with container environment variables set", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-container-env")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "BAR", []string{"ssh", "--command", "echo $FOO", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with container user", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-container-user")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "vscode", []string{"ssh", "--command", "ps u -p 1", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with privileged", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-privileged")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.Privileged).To(gomega.BeTrue(), "container run with privileged true")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with capAdd", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-capadd")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.CapAdd).To(gomega.ContainElement("SYS_PTRACE"), "image capabilities are not duplicated")
				gomega.Expect(containerDetail.HostConfig.CapAdd).To(gomega.ContainElement("NET_ADMIN"), "devcontainer configuration can add capabilities")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with securityOpt", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-securityOpt")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.SecurityOpt).To(gomega.ContainElement("seccomp=unconfined"), "securityOpts contain seccomp=unconfined")
				gomega.Expect(containerDetail.HostConfig.SecurityOpt).To(gomega.ContainElement("apparmor=unconfined"), "securityOpts contain apparmor=unconfined")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with override command", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-overrideCommand")
				framework.ExpectNoError(err)
				defer framework.CleanupTempDir(initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd([]string{"docker"})
				err = f.DevPodProviderUse(context.Background(), "docker")
				framework.ExpectNoError(err)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				defer f.DevPodWorkspaceDelete(ctx, projectName)

				ids, err := dockerHelper.FindContainer([]string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(len(ids)).To(gomega.Equal(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Config.Entrypoint).NotTo(gomega.ContainElement("bash"), "overrides container entry point")
				gomega.Expect(containerDetail.Config.Cmd).To(gomega.BeEmpty(), "overrides container command")
			}, ginkgo.SpecTimeout(60*time.Second))
		})
	})

})
