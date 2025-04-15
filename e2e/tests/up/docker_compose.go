package up

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/compose"
	docker "github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = DevPodDescribe("devpod up test suite", func() {
	ginkgo.Context("testing up command", ginkgo.Label("up-docker-compose"), ginkgo.Ordered, func() {
		var dockerHelper *docker.DockerHelper
		var composeHelper *compose.ComposeHelper
		var initialDir string

		ginkgo.BeforeEach(func() {
			var err error
			initialDir, err = os.Getwd()
			framework.ExpectNoError(err)

			dockerHelper = &docker.DockerHelper{DockerCommand: "docker", Log: log.Default}
			composeHelper, err = compose.NewComposeHelper("", dockerHelper)
			framework.ExpectNoError(err)
		})

		ginkgo.Context("with docker-compose", func() {
			ginkgo.It("should start a new workspace with root folder configuration", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Mounts).To(gomega.HaveLen(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.BeTrue())
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with sub-folder configuration", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-subfolder")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Mounts).To(gomega.HaveLen(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.BeTrue())
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with multiple services", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-multiple-services")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := composeHelper.GetProjectName(workspace.UID)

				appIDs, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(appIDs).To(gomega.HaveLen(1), "app container to be created")

				dbIDs, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "db"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(dbIDs).To(gomega.HaveLen(1), "db container to be created")
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with specific services", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-run-services")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := composeHelper.GetProjectName(workspace.UID)

				appIDs, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(appIDs).To(gomega.HaveLen(1), "app container to be created")

				dbIDs, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "db"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(dbIDs).To(gomega.BeEmpty(), "db container not to be created")
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with .devcontainer docker-compose overrides", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-overrides")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := composeHelper.GetProjectName(workspace.UID)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Mounts).To(gomega.HaveLen(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.BeTrue())
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with container environment variables set", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-container-env")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "BAR", []string{"ssh", "--command", "echo $FOO", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with container user", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-container-user")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "root", []string{"ssh", "--command", "ps u -p 1", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with privileged", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-privileged")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.Privileged).To(gomega.BeTrue(), "container run with privileged true")
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with capAdd", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-capadd")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.CapAdd).To(gomega.ContainElement("SYS_PTRACE"), "image capabilities are not duplicated")
				gomega.Expect(containerDetail.HostConfig.CapAdd).To(gomega.ContainElement("NET_ADMIN"), "devcontainer configuration can add capabilities")
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with securityOpt", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-securityOpt")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.SecurityOpt).To(gomega.ContainElement("seccomp=unconfined"), "securityOpts contain seccomp=unconfined")
				gomega.Expect(containerDetail.HostConfig.SecurityOpt).To(gomega.ContainElement("apparmor=unconfined"), "securityOpts contain apparmor=unconfined")
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with override command", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-override-command")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Config.Entrypoint).NotTo(gomega.ContainElement("bash"), "overrides container entry point")
				gomega.Expect(containerDetail.Config.Cmd).To(gomega.BeEmpty(), "overrides container command")
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with remote env", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-remote-env")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "/home/vscode/remote-env.out", []string{"ssh", "--command", "ls $HOME/remote-env.out", projectName})
				framework.ExpectNoError(err)

				err = f.ExecCommand(ctx, true, true, "BAR", []string{"ssh", "--command", "cat $HOME/remote-env.out", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with remote user", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-remote-user")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "root", []string{"ssh", "--command", "cat $HOME/remote-user.out", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace and substitute devcontainer.json variables", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-variables")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				devContainerID, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/dev-container-id.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(devContainerID).NotTo(gomega.BeEmpty())

				containerEnvPath, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-env-path.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerEnvPath).To(gomega.ContainSubstring("/usr/local/bin"))

				localEnvHome, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/local-env-home.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(localEnvHome).To(gomega.Equal(os.Getenv("HOME")))

				localWorkspaceFolder, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/local-workspace-folder.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(framework.CleanString(localWorkspaceFolder)).To(gomega.Equal(framework.CleanString(tempDir)))

				localWorkspaceFolderBasename, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/local-workspace-folder-basename.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(localWorkspaceFolderBasename).To(gomega.Equal(filepath.Base(tempDir)))

				containerWorkspaceFolder, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-workspace-folder.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerWorkspaceFolder).To(gomega.Equal("/workspaces"))

				containerWorkspaceFolderBasename, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-workspace-folder-basename.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerWorkspaceFolderBasename).To(gomega.Equal("workspaces"))
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with mounts", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-mounts")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir, "--debug")
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				_, _, err = f.ExecCommandCapture(ctx, []string{"ssh", "--command", "touch /home/vscode/mnt1/foo.txt", projectName, "--user", "root"})
				framework.ExpectNoError(err)

				_, _, err = f.ExecCommandCapture(ctx, []string{"ssh", "--command", "echo -n BAR > /home/vscode/mnt1/foo.txt", projectName, "--user", "root"})
				framework.ExpectNoError(err)

				foo, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/mnt1/foo.txt", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(foo).To(gomega.Equal("BAR"))

				bar, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/mnt2/bar.txt", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(bar).To(gomega.Equal("FOO"))
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with host:port forwardPorts", func(ctx context.Context) {
				if runtime.GOOS == "windows" {
					ginkgo.Skip("skipping on windows")
				}
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-forward-ports")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir, "--debug")
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				done := make(chan error)

				sshContext, sshCancel := context.WithCancel(context.Background())
				go func() {
					cmd := exec.CommandContext(sshContext, filepath.Join(f.DevpodBinDir, f.DevpodBinName), "ssh", projectName, "--command", "sleep 30")

					if err := cmd.Start(); err != nil {
						done <- err
						return
					}

					if err := cmd.Wait(); err != nil {
						done <- err
						return
					}

					done <- nil
				}()

				gomega.Eventually(func(g gomega.Gomega) {
					response, err := http.Get("http://localhost:8080")
					g.Expect(err).NotTo(gomega.HaveOccurred())

					body, err := io.ReadAll(response.Body)
					g.Expect(err).NotTo(gomega.HaveOccurred())
					g.Expect(body).To(gomega.ContainSubstring("Thank you for using nginx."))
				}).
					WithPolling(1 * time.Second).
					WithTimeout(20 * time.Second).
					Should(gomega.Succeed())

				sshCancel()
				err = <-done

				// skip windows for now
				if runtime.GOOS != "windows" {
					gomega.Expect(err).To(gomega.Or(
						gomega.MatchError("signal: killed"),
						gomega.MatchError(context.Canceled),
					))
				}
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with features", func(ctx context.Context) {
				if runtime.GOOS == "windows" {
					ginkgo.Skip("skipping on windows")
				}

				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-features")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir, "--debug")
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				vclusterVersionOutput, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "vcluster --version", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(vclusterVersionOutput).To(gomega.ContainSubstring("vcluster version 0.24.1"))
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with env-file", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-env-file")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				devPodUpOutput, _, err := f.ExecCommandCapture(ctx, []string{"up", "--debug", "--ide", "none", tempDir})
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")
				gomega.Expect(devPodUpOutput).NotTo(gomega.ContainSubstring("Defaulting to a blank string."))
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should restart a workspace", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-env-file")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				devPodUpOutput, _, err := f.ExecCommandCapture(ctx, []string{"up", "--debug", "--ide", "none", tempDir})
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")
				gomega.Expect(devPodUpOutput).NotTo(gomega.ContainSubstring("Defaulting to a blank string."))
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.Context("with lifecycle commands", func() {
				ginkgo.It("should start a new workspace and execute array based lifecycle commands", func(ctx context.Context) {
					tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-lifecycle-array")
					framework.ExpectNoError(err)
					ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

					f := framework.NewDefaultFramework(initialDir + "/bin")
					_ = f.DevPodProviderAdd(ctx, "docker")
					err = f.DevPodProviderUse(ctx, "docker")
					framework.ExpectNoError(err)

					ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

					err = f.DevPodUp(ctx, tempDir)
					framework.ExpectNoError(err)

					workspace, err := f.FindWorkspace(ctx, tempDir)
					framework.ExpectNoError(err)
					projectName := workspace.ID

					ids, err := dockerHelper.FindContainer(ctx, []string{
						fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
						fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
					})
					framework.ExpectNoError(err)
					gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

					initializeCommand, err := os.ReadFile(filepath.Join(tempDir, "initialize-command.out"))
					framework.ExpectNoError(err)
					gomega.Expect(initializeCommand).To(gomega.ContainSubstring("initializeCommand"))

					onCreateCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/on-create-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(onCreateCommand).To(gomega.ContainSubstring("onCreateCommand"))

					updateContentCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/update-content-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(updateContentCommand).To(gomega.Equal("updateContentCommand"))

					postCreateCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-create-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(postCreateCommand).To(gomega.Equal("postCreateCommand"))

					postStartCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-start-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(postStartCommand).To(gomega.Equal("postStartCommand"))

					postAttachCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-attach-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(postAttachCommand).To(gomega.Equal("postAttachCommand"))
				}, ginkgo.SpecTimeout(framework.GetTimeout()))

				//ginkgo.FIt("should start a new workspace and execute object based lifecycle commands", func(ctx context.Context) {
				//	tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-lifecycle-object")
				//	framework.ExpectNoError(err)
				//	ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)
				//
				//	f := framework.NewDefaultFramework(initialDir + "/bin")
				//	_ = f.DevPodProviderAdd(ctx, "docker"})
				//	err = f.DevPodProviderUse(context.Background(), "docker")
				//	framework.ExpectNoError(err)
				//
				//	err = f.DevPodUp(ctx, tempDir)
				//	framework.ExpectNoError(err)
				//
				//	// Check for docker-compose container running
				//	projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				//	ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), projectName)
				//
				//	ids, err := dockerHelper.FindContainer(ctx, []string{
				//		fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
				//		fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				//	})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")
				//
				//	initializeCommand, err := os.ReadFile(filepath.Join(tempDir, "initialize-command.out"))
				//	framework.ExpectNoError(err)
				//	gomega.Expect(initializeCommand).To(gomega.ContainSubstring("initializeCommand"))
				//
				//	onCreateCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/on-create-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(onCreateCommand).To(gomega.ContainSubstring("onCreateCommand"))
				//
				//	updateContentCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/update-content-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(updateContentCommand).To(gomega.Equal("updateContentCommand"))
				//
				//	postCreateCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-create-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(postCreateCommand).To(gomega.Equal("postCreateCommand"))
				//
				//	postStartCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-start-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(postStartCommand).To(gomega.Equal("postStartCommand"))
				//
				//	postAttachCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-attach-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(postAttachCommand).To(gomega.Equal("postAttachCommand"))
				//}, ginkgo.SpecTimeout(framework.GetTimeout()))
			})
		})
	})
})
