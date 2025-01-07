package pro

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	netUrl "net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/mgutz/ansi"
	"github.com/skratchdot/open-golang/open"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/auth"
	loftclientset "github.com/loft-sh/api/v4/pkg/clientset/versioned"
	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/hash"
	"github.com/loft-sh/log/scanner"
	"github.com/loft-sh/log/survey"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubectl/pkg/util/term"
)

const LoftRouterDomainSecret = "loft-router-domain"
const passwordChangedHint = "(has been changed)"
const defaultUser = "admin"

const defaultReleaseName = "devpod-pro"

var defaultDeploymentName = "loft" // Need to update helm chart if we change this!

// StartCmd holds the login cmd flags
type StartCmd struct {
	proflags.GlobalFlags

	KubeClient       kubernetes.Interface
	Log              log.Logger
	RestConfig       *rest.Config
	Context          string
	Values           string
	LocalPort        string
	Version          string
	DockerImage      string
	Namespace        string
	Password         string
	Host             string
	Email            string
	ChartRepo        string
	Product          string
	ChartName        string
	ChartPath        string
	DockerArgs       []string
	Reset            bool
	NoPortForwarding bool
	NoTunnel         bool
	NoLogin          bool
	NoWait           bool
	Upgrade          bool
	ReuseValues      bool
	Docker           bool
}

// NewStartCmd creates a new command
func NewStartCmd(flags *proflags.GlobalFlags) *cobra.Command {
	cmd := &StartCmd{GlobalFlags: *flags,
		Product:   "devpod-pro",
		ChartName: "devpod-pro",
		Log:       log.Default,
	}
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a Devpod Pro instance",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background())
		},
	}

	startCmd.Flags().BoolVar(&cmd.Docker, "docker", false, "If enabled will try to deploy DevPod Pro to the local docker installation.")
	startCmd.Flags().StringVar(&cmd.DockerImage, "docker-image", "", "The docker image to install.")
	startCmd.Flags().StringArrayVar(&cmd.DockerArgs, "docker-arg", []string{}, "Extra docker args")
	startCmd.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")
	startCmd.Flags().StringVar(&cmd.Namespace, "namespace", "devpod-pro", "The namespace to install into")
	startCmd.Flags().StringVar(&cmd.Host, "host", "", "Provide a hostname to enable ingress and configure its hostname")
	startCmd.Flags().StringVar(&cmd.Password, "password", "", "The password to use for the admin account. (If empty this will be the namespace UID)")
	startCmd.Flags().StringVar(&cmd.Version, "version", "", "The version to install")
	startCmd.Flags().StringVar(&cmd.Values, "values", "", "Path to a file for extra helm chart values")
	startCmd.Flags().BoolVar(&cmd.ReuseValues, "reuse-values", true, "Reuse previous helm values on upgrade")
	startCmd.Flags().BoolVar(&cmd.Upgrade, "upgrade", false, "If true, will try to upgrade the release")
	startCmd.Flags().StringVar(&cmd.Email, "email", "", "The email to use for the installation")
	startCmd.Flags().BoolVar(&cmd.Reset, "reset", false, "If true, an existing instance will be deleted before installing DevPod Pro")
	startCmd.Flags().BoolVar(&cmd.NoWait, "no-wait", false, "If true, will not wait after installing it")
	startCmd.Flags().BoolVar(&cmd.NoTunnel, "no-tunnel", false, "If true, will not create a loft.host tunnel for this installation")
	startCmd.Flags().BoolVar(&cmd.NoLogin, "no-login", false, "If true, will not login to a DevPod Pro instance on start")
	startCmd.Flags().StringVar(&cmd.ChartPath, "chart-path", "", "The local chart path to deploy DevPod Pro")
	startCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", "https://charts.loft.sh/", "The chart repo to deploy DevPod Pro")

	return startCmd
}

// Run runs the command logic
func (cmd *StartCmd) Run(ctx context.Context) error {
	if cmd.Docker {
		return cmd.startDocker(ctx)
	}

	// only set local port by default in kubernetes installation
	if cmd.LocalPort == "" {
		cmd.LocalPort = "9898"
	}

	err := cmd.prepare()
	if err != nil {
		return err
	}
	cmd.Log.WriteString(logrus.InfoLevel, "\n")

	// Uninstall already existing instance
	if cmd.Reset {
		err = uninstall(ctx, cmd.KubeClient, cmd.RestConfig, cmd.Context, cmd.Namespace, cmd.Log)
		if err != nil {
			return err
		}
	}

	// Is already installed?
	isInstalled, err := isAlreadyInstalled(ctx, cmd.KubeClient, cmd.Namespace)
	if err != nil {
		return err
	}

	// Use default password if none is set
	if cmd.Password == "" {
		defaultPassword, err := getDefaultPassword(ctx, cmd.KubeClient, cmd.Namespace)
		if err != nil {
			return err
		}

		cmd.Password = defaultPassword
	}

	// Upgrade Loft if already installed
	if isInstalled {
		return cmd.handleAlreadyExistingInstallation(ctx)
	}

	// Install Loft
	cmd.Log.Info("Welcome to DevPod Pro!")
	cmd.Log.Info("This installer will help you to get started.")

	// make sure we are ready for installing
	err = cmd.prepareInstall(ctx)
	if err != nil {
		return err
	}

	err = cmd.upgrade()
	if err != nil {
		return err
	}

	return cmd.success(ctx)
}

func (cmd *StartCmd) upgrade() error {
	extraArgs := []string{}
	if cmd.NoTunnel {
		extraArgs = append(extraArgs, "--set-string", "env.DISABLE_LOFT_ROUTER=true")
	}
	if cmd.Password != "" {
		extraArgs = append(extraArgs, "--set", "admin.password="+cmd.Password)
	}
	if cmd.Host != "" {
		extraArgs = append(extraArgs, "--set", "ingress.enabled=true", "--set", "ingress.host="+cmd.Host)
	}
	if cmd.Version != "" {
		extraArgs = append(extraArgs, "--version", cmd.Version)
	}
	if cmd.Product != "" {
		extraArgs = append(extraArgs, "--set", "product="+cmd.Product)
	}

	// Do not use --reuse-values if --reset flag is provided because this should be a new install and it will cause issues with `helm template`
	if !cmd.Reset && cmd.ReuseValues {
		extraArgs = append(extraArgs, "--reuse-values")
	}

	if cmd.Values != "" {
		absValuesPath, err := filepath.Abs(cmd.Values)
		if err != nil {
			return err
		}
		extraArgs = append(extraArgs, "--values", absValuesPath)
	}

	chartName := cmd.ChartPath
	chartRepo := ""
	if chartName == "" {
		chartName = cmd.ChartName
		chartRepo = cmd.ChartRepo
	}

	err := upgradeRelease(chartName, chartRepo, cmd.Context, cmd.Namespace, extraArgs, cmd.Log)
	if err != nil {
		if !cmd.Reset {
			return errors.New(err.Error() + fmt.Sprintf("\n\nIf want to purge and reinstall DevPod Pro, run: %s\n", ansi.Color("devpod pro start --reset", "green+b")))
		}

		// Try to purge Loft and retry install
		cmd.Log.Info("Trying to delete objects blocking current installation")

		manifests, err := getReleaseManifests(chartName, chartRepo, cmd.Context, cmd.Namespace, extraArgs, cmd.Log)
		if err != nil {
			return err
		}

		kubectlDelete := exec.Command("kubectl", "delete", "-f", "-", "--ignore-not-found=true", "--grace-period=0", "--force")

		buffer := bytes.Buffer{}
		buffer.Write([]byte(manifests))

		kubectlDelete.Stdin = &buffer
		kubectlDelete.Stdout = os.Stdout
		kubectlDelete.Stderr = os.Stderr

		// Ignoring potential errors here
		_ = kubectlDelete.Run()

		// Retry Loft installation
		err = upgradeRelease(chartName, chartRepo, cmd.Context, cmd.Namespace, extraArgs, cmd.Log)
		if err != nil {
			return errors.New(err.Error() + fmt.Sprintf("\n\nExisting installation failed. Reach out to get help:\n- via Slack: %s (fastest option)\n- via Online Chat: %s\n- via Email: %s\n", ansi.Color("https://slack.loft.sh/", "green+b"), ansi.Color("https://loft.sh/", "green+b"), ansi.Color("support@loft.sh", "green+b")))
		}
	}

	return nil
}

func (cmd *StartCmd) success(ctx context.Context) error {
	if cmd.NoWait {
		return nil
	}

	// wait until deployment is ready
	loftPod, err := cmd.waitForDeployment(ctx)
	if err != nil {
		return err
	}

	// check if installed locally
	isLocal := isInstalledLocally(ctx, cmd.KubeClient, cmd.Namespace)
	if isLocal {
		// check if loft domain secret is there
		if !cmd.NoTunnel {
			loftRouterDomain, err := cmd.pingLoftRouter(ctx, loftPod)
			if err != nil {
				cmd.Log.Errorf("Error retrieving loft router domain: %v", err)
				cmd.Log.Info("Fallback to use port-forwarding")
			} else if loftRouterDomain != "" {
				return cmd.successLoftRouter(loftRouterDomain)
			}
		}

		return cmd.successLocal()
	}

	// get login link
	cmd.Log.Info("Checking Loft status...")
	host, err := getIngressHost(ctx, cmd.KubeClient, cmd.Namespace)
	if err != nil {
		return err
	}

	// check if loft is reachable
	reachable, err := isHostReachable(ctx, host)
	if !reachable || err != nil {
		const (
			YesOption = "Yes"
			NoOption  = "No, please re-run the DNS check"
		)

		answer, err := cmd.Log.Question(&survey.QuestionOptions{
			Question:     "Unable to reach Loft at https://" + host + ". Do you want to start port-forwarding instead?",
			DefaultValue: YesOption,
			Options: []string{
				YesOption,
				NoOption,
			},
		})
		if err != nil {
			return err
		}

		if answer == YesOption {
			return cmd.successLocal()
		}
	}

	return cmd.successRemote(ctx, host)
}

func (cmd *StartCmd) successRemote(ctx context.Context, host string) error {
	printSuccess := func() {
		url := "https://" + host

		password := cmd.Password
		if password == "" {
			password = passwordChangedHint
		}

		cmd.Log.WriteString(logrus.InfoLevel, fmt.Sprintf(`


##########################   LOGIN   ############################

Username: `+ansi.Color("admin", "green+b")+`
Password: `+ansi.Color(password, "green+b")+`  # Change via UI or via: `+ansi.Color("devpod pro reset password", "green+b")+`

Login via UI:  %s
Login via CLI: %s

!!! You must accept the untrusted certificate in your browser !!!

Follow this guide to add a valid certificate: %s

#################################################################

DevPod Pro was successfully installed and can now be reached at: %s

Thanks for using DevPod Pro!
`,
			ansi.Color(url, "green+b"),
			ansi.Color("devpod pro login"+" --insecure "+url, "green+b"),
			"https://loft.sh/docs/administration/ssl",
			url))
	}
	ready, err := isHostReachable(ctx, host)
	if err != nil {
		return err
	} else if ready {
		printSuccess()
		return nil
	}

	// Print DNS Configuration
	cmd.Log.WriteString(logrus.InfoLevel, `

###################################     DNS CONFIGURATION REQUIRED     ##################################

Create a DNS A-record for `+host+` with the EXTERNAL-IP of your nginx-ingress controller.
To find this EXTERNAL-IP, run the following command and look at the output:

> kubectl get services -n ingress-nginx
                                                     |---------------|
NAME                       TYPE           CLUSTER-IP | EXTERNAL-IP   |  PORT(S)                      AGE
ingress-nginx-controller   LoadBalancer   10.0.0.244 | XX.XXX.XXX.XX |  80:30984/TCP,443:31758/TCP   19m
                                                     |^^^^^^^^^^^^^^^|

EXTERNAL-IP may be 'pending' for a while until your cloud provider has created a new load balancer.

#########################################################################################################

The command will wait until DevPod Pro is reachable under the host.

`)

	cmd.Log.Info("Waiting for you to configure DNS, so DevPod Pro can be reached on https://" + host)
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, platform.Timeout(), true, func(ctx context.Context) (done bool, err error) {
		return isHostReachable(ctx, host)
	})
	if err != nil {
		return err
	}

	cmd.Log.Done("DevPod Pro is reachable at https://" + host)

	printSuccess()
	return nil
}
func (cmd *StartCmd) successLocal() error {
	url := "https://localhost:" + cmd.LocalPort

	if !cmd.NoLogin {
		err := cmd.login(url)
		if err != nil {
			return err
		}
	}

	password := cmd.Password
	if password == "" {
		password = passwordChangedHint
	}

	cmd.Log.WriteString(logrus.InfoLevel, fmt.Sprintf(`

##########################   LOGIN   ############################

Username: `+ansi.Color("admin", "green+b")+`
Password: `+ansi.Color(password, "green+b")+`  # Change via UI or via: `+ansi.Color("devpod pro reset password", "green+b")+`

Login via UI:  %s
Login via CLI: %s

!!! You must accept the untrusted certificate in your browser !!!

#################################################################

DevPod Pro was successfully installed.

Thanks for using DevPod Pro!
`, ansi.Color(url, "green+b"), ansi.Color("devpod pro login"+" --insecure "+url, "green+b")))
	blockChan := make(chan bool)
	<-blockChan
	return nil
}

func (cmd *StartCmd) startDocker(ctx context.Context) error {
	cmd.Log.Infof("Starting DevPod Pro in Docker...")
	name := "devpod-pro"

	// prepare installation
	err := cmd.prepareDocker()
	if err != nil {
		return err
	}

	// try to find loft container
	containerID, err := cmd.findLoftContainer(ctx, name, true)
	if err != nil {
		return err
	}

	// check if container is there
	if containerID != "" && (cmd.Reset || cmd.Upgrade) {
		cmd.Log.Info("Existing instance found.")
		err = cmd.uninstallDocker(ctx, containerID)
		if err != nil {
			return err
		}

		containerID = ""
	}

	// Use default password if none is set
	if cmd.Password == "" {
		cmd.Password = getMachineUID(cmd.Log)
	}

	// check if is installed
	if containerID != "" {
		cmd.Log.Info("Existing instance found. Run with --upgrade to apply new configuration")
		return cmd.successDocker(ctx, containerID)
	}

	// Install Loft
	cmd.Log.Info("Welcome to DevPod Pro!")
	cmd.Log.Info("This installer will help you get started.")

	// make sure we are ready for installing
	containerID, err = cmd.runInDocker(ctx, name)
	if err != nil {
		return err
	} else if containerID == "" {
		return fmt.Errorf("%w: %s", ErrMissingContainer, "couldn't find Loft container after starting it")
	}

	return cmd.successDocker(ctx, containerID)
}

func (cmd *StartCmd) successDocker(ctx context.Context, containerID string) error {
	if cmd.NoWait {
		return nil
	}

	// wait until Loft is ready
	host, err := cmd.waitForLoftDocker(ctx, containerID)
	if err != nil {
		return err
	}

	// wait for domain to become reachable
	cmd.Log.Infof("Wait for DevPod Pro to become available at %s...", host)
	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*10, true, func(ctx context.Context) (bool, error) {
		containerDetails, err := cmd.inspectContainer(ctx, containerID)
		if err != nil {
			return false, fmt.Errorf("inspect loft container: %w", err)
		} else if strings.ToLower(containerDetails.State.Status) == "exited" || strings.ToLower(containerDetails.State.Status) == "dead" {
			logs, _ := cmd.logsContainer(ctx, containerID)
			return false, fmt.Errorf("container failed (status: %s):\n %s", containerDetails.State.Status, logs)
		}

		return isHostReachable(ctx, host)
	})
	if err != nil {
		return fmt.Errorf("error waiting for DevPod Pro: %w", err)
	}

	// print success message
	PrintSuccessMessageDockerInstall(host, cmd.Password, cmd.Log)
	return nil
}

func PrintSuccessMessageDockerInstall(host, password string, log log.Logger) {
	url := "https://" + host
	log.WriteString(logrus.InfoLevel, fmt.Sprintf(`


##########################   LOGIN   ############################

Username: `+ansi.Color("admin", "green+b")+`
Password: `+ansi.Color(password, "green+b")+`

Login via UI:  %s
Login via CLI: %s

#################################################################

DevPod Pro was successfully installed and can now be reached at: %s

Thanks for using DevPod Pro!
`,
		ansi.Color(url, "green+b"),
		ansi.Color("devpod pro login"+" "+url, "green+b"),
		url,
	))
}

func (cmd *StartCmd) waitForLoftDocker(ctx context.Context, containerID string) (string, error) {
	cmd.Log.Info("Wait for DevPod Pro to become available...")

	// check for local port
	containerDetails, err := cmd.inspectContainer(ctx, containerID)
	if err != nil {
		return "", err
	} else if len(containerDetails.NetworkSettings.Ports) > 0 && len(containerDetails.NetworkSettings.Ports["10443/tcp"]) > 0 {
		return "localhost:" + containerDetails.NetworkSettings.Ports["10443/tcp"][0].HostPort, nil
	}

	// check if no tunnel
	if cmd.NoTunnel {
		return "", fmt.Errorf("%w: %s", ErrLoftNotReachable, "cannot connect to DevPod Pro as it has no exposed port and --no-tunnel is enabled")
	}

	// wait for router
	url := ""
	waitErr := wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*10, true, func(ctx context.Context) (bool, error) {
		url, err = cmd.findLoftRouter(ctx, containerID)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if waitErr != nil {
		return "", fmt.Errorf("error waiting for loft router domain: %w", err)
	}

	return url, nil
}

func (cmd *StartCmd) findLoftRouter(ctx context.Context, id string) (string, error) {
	out, err := cmd.buildDockerCmd(ctx, "exec", id, "cat", "/var/lib/loft/loft-domain.txt").Output()
	if err != nil {
		return "", WrapCommandError(out, err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (cmd *StartCmd) prepareDocker() error {
	// test for helm and kubectl
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("seems like docker is not installed. Docker is required for the installation of loft. Please visit https://docs.docker.com/engine/install/ for install instructions")
	}

	output, err := exec.Command("docker", "ps").CombinedOutput()
	if err != nil {
		return fmt.Errorf("seems like there are issues with your docker cli: \n\n%s", output)
	}

	return nil
}

func (cmd *StartCmd) uninstallDocker(ctx context.Context, id string) error {
	cmd.Log.Infof("Uninstalling...")

	// stop container
	out, err := cmd.buildDockerCmd(ctx, "stop", id).Output()
	if err != nil {
		return fmt.Errorf("stop container: %w", WrapCommandError(out, err))
	}

	// remove container
	out, err = cmd.buildDockerCmd(ctx, "rm", id).Output()
	if err != nil {
		return fmt.Errorf("remove container: %w", WrapCommandError(out, err))
	}

	return nil
}

func (cmd *StartCmd) runInDocker(ctx context.Context, name string) (string, error) {
	args := []string{"run", "-d", "--name", name}
	if cmd.NoTunnel {
		args = append(args, "--env", "DISABLE_LOFT_ROUTER=true")
	}
	if cmd.Password != "" {
		args = append(args, "--env", "ADMIN_PASSWORD_HASH="+hash.String(cmd.Password))
	}

	// run as root otherwise we get permission errors
	args = append(args, "-u", "root")

	// mount the loft lib
	args = append(args, "-v", "loft-data:/var/lib/loft")

	// set port
	if cmd.LocalPort != "" {
		args = append(args, "-p", cmd.LocalPort+":10443")
	}

	// set extra args
	args = append(args, cmd.DockerArgs...)

	// set image
	if cmd.DockerImage != "" {
		args = append(args, cmd.DockerImage)
	} else if cmd.Version != "" {
		args = append(args, "ghcr.io/loft-sh/devpod-pro:"+strings.TrimPrefix(cmd.Version, "v"))
	} else {
		args = append(args, "ghcr.io/loft-sh/devpod-pro:latest")
	}

	cmd.Log.Infof("Start DevPod Pro via 'docker %s'", strings.Join(args, " "))
	runCmd := cmd.buildDockerCmd(ctx, args...)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	err := runCmd.Run()
	if err != nil {
		return "", err
	}

	return cmd.findLoftContainer(ctx, name, false)
}

func (cmd *StartCmd) logsContainer(ctx context.Context, id string) (string, error) {
	args := []string{"logs", id}
	out, err := cmd.buildDockerCmd(ctx, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("logs container: %w", WrapCommandError(out, err))
	}

	return string(out), nil
}

func (cmd *StartCmd) inspectContainer(ctx context.Context, id string) (*ContainerDetails, error) {
	args := []string{"inspect", "--type", "container", id}
	out, err := cmd.buildDockerCmd(ctx, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", WrapCommandError(out, err))
	}

	containerDetails := []*ContainerDetails{}
	err = json.Unmarshal(out, &containerDetails)
	if err != nil {
		return nil, fmt.Errorf("parse inspect output: %w", err)
	} else if len(containerDetails) == 0 {
		return nil, fmt.Errorf("coudln't find container %s", id)
	}

	return containerDetails[0], nil
}

func (cmd *StartCmd) removeContainer(ctx context.Context, id string) error {
	args := []string{"rm", id}
	out, err := cmd.buildDockerCmd(ctx, args...).Output()
	if err != nil {
		return fmt.Errorf("remove container: %w", WrapCommandError(out, err))
	}

	return nil
}

func (cmd *StartCmd) findLoftContainer(ctx context.Context, name string, onlyRunning bool) (string, error) {
	args := []string{"ps", "-q", "-a", "-f", "name=^" + name + "$"}
	out, err := cmd.buildDockerCmd(ctx, args...).Output()
	if err != nil {
		// fallback to manual search
		return "", fmt.Errorf("error finding container: %w", WrapCommandError(out, err))
	}

	arr := []string{}
	scan := scanner.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		arr = append(arr, strings.TrimSpace(scan.Text()))
	}
	if len(arr) == 0 {
		return "", nil
	}

	// remove the failed / exited containers
	runningContainerID := ""
	for _, containerID := range arr {
		containerState, err := cmd.inspectContainer(ctx, containerID)
		if err != nil {
			return "", err
		} else if onlyRunning && strings.ToLower(containerState.State.Status) != "running" {
			err = cmd.removeContainer(ctx, containerID)
			if err != nil {
				return "", err
			}
		} else {
			runningContainerID = containerID
		}
	}

	return runningContainerID, nil
}

func (cmd *StartCmd) buildDockerCmd(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "docker", args...)
}

func (cmd *StartCmd) prepareInstall(ctx context.Context) error {
	// delete admin user & secret
	return uninstall(ctx, cmd.KubeClient, cmd.RestConfig, cmd.Context, cmd.Namespace, log.Discard)
}

func (cmd *StartCmd) prepare() error {
	loader, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}
	loftConfig := loader.Config()

	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	// load the raw config
	kubeConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	// we switch the context to the install config
	contextToLoad := kubeConfig.CurrentContext
	if cmd.Context != "" {
		contextToLoad = cmd.Context
	} else if loftConfig.LastInstallContext != "" && loftConfig.LastInstallContext != contextToLoad {
		contextToLoad, err = cmd.Log.Question(&survey.QuestionOptions{
			Question:     "Seems like you try to use 'devpod pro start' with a different kubernetes context than before. Please choose which kubernetes context you want to use",
			DefaultValue: contextToLoad,
			Options:      []string{contextToLoad, loftConfig.LastInstallContext},
		})
		if err != nil {
			return err
		}
	}
	cmd.Context = contextToLoad

	loftConfig.LastInstallContext = contextToLoad
	_ = loader.Save()

	// kube client config
	kubeClientConfig = clientcmd.NewNonInteractiveClientConfig(kubeConfig, contextToLoad, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules())

	// test for helm and kubectl
	_, err = exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("seems like helm is not installed. Helm is required for the installation of loft. Please visit https://helm.sh/docs/intro/install/ for install instructions")
	}

	output, err := exec.Command("helm", "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("seems like there are issues with your helm client: \n\n%s", output)
	}

	_, err = exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("seems like kubectl is not installed. Kubectl is required for the installation of loft. Please visit https://kubernetes.io/docs/tasks/tools/install-kubectl/ for install instructions")
	}

	output, err = exec.Command("kubectl", "version", "--context", contextToLoad).CombinedOutput()
	if err != nil {
		return fmt.Errorf("seems like kubectl cannot connect to your Kubernetes cluster: \n\n%s", output)
	}

	cmd.RestConfig, err = kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	cmd.KubeClient, err = kubernetes.NewForConfig(cmd.RestConfig)
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	// Check if cluster has RBAC correctly configured
	_, err = cmd.KubeClient.RbacV1().ClusterRoles().Get(context.Background(), "cluster-admin", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error retrieving cluster role 'cluster-admin': %w. Please make sure RBAC is correctly configured in your cluster", err)
	}

	return nil
}

func (cmd *StartCmd) handleAlreadyExistingInstallation(ctx context.Context) error {
	enableIngress := false

	// Only ask if ingress should be enabled if --upgrade flag is not provided
	if !cmd.Upgrade && term.IsTerminal(os.Stdin) {
		cmd.Log.Info("Existing instance found.")

		// Check if Loft is installed in a local cluster
		isLocal := isInstalledLocally(ctx, cmd.KubeClient, cmd.Namespace)

		// Skip question if --host flag is provided
		if cmd.Host != "" {
			enableIngress = true
		}

		if enableIngress {
			if isLocal {
				// Confirm with user if this is a local cluster
				const (
					YesOption = "Yes"
					NoOption  = "No, my cluster is running not locally (GKE, EKS, Bare Metal, etc.)"
				)

				answer, err := cmd.Log.Question(&survey.QuestionOptions{
					Question:     "Seems like your cluster is running locally (docker desktop, minikube, kind etc.). Is that correct?",
					DefaultValue: YesOption,
					Options: []string{
						YesOption,
						NoOption,
					},
				})
				if err != nil {
					return err
				}

				isLocal = answer == YesOption
			}

			if isLocal {
				// Confirm with user if ingress should be installed in local cluster
				var (
					YesOption = "Yes, enable the ingress anyway"
					NoOption  = "No"
				)

				answer, err := cmd.Log.Question(&survey.QuestionOptions{
					Question:     "Enabling ingress is usually only useful for remote clusters. Do you still want to deploy the ingress to your local cluster?",
					DefaultValue: NoOption,
					Options: []string{
						NoOption,
						YesOption,
					},
				})
				if err != nil {
					return err
				}

				enableIngress = answer == YesOption
			}
		}

		// Check if we need to enable ingress
		if enableIngress {
			// Ask for hostname if --host flag is not provided
			if cmd.Host == "" {
				host, err := enterHostNameQuestion(cmd.Log)
				if err != nil {
					return err
				}

				cmd.Host = host
			} else {
				cmd.Log.Info("Will enable an ingress with hostname: " + cmd.Host)
			}

			if term.IsTerminal(os.Stdin) {
				err := ensureIngressController(ctx, cmd.KubeClient, cmd.Context, cmd.Log)
				if err != nil {
					return errors.Wrap(err, "install ingress controller")
				}
			}
		}
	}

	// Only upgrade if --upgrade flag is present or user decided to enable ingress
	if cmd.Upgrade || enableIngress {
		err := cmd.upgrade()
		if err != nil {
			return err
		}
	}

	return cmd.success(ctx)
}

func (cmd *StartCmd) waitForDeployment(ctx context.Context) (*corev1.Pod, error) {
	// wait for loft pod to start
	cmd.Log.Info("Waiting for DevPod Pro pod to be running...")
	loftPod, err := platform.WaitForPodReady(ctx, cmd.KubeClient, cmd.Namespace, cmd.Log)
	cmd.Log.Donef("Release Pod successfully started")
	if err != nil {
		return nil, err
	}

	// ensure user admin secret is there
	isNewPassword, err := ensureAdminPassword(ctx, cmd.KubeClient, cmd.RestConfig, cmd.Password, cmd.Log)
	if err != nil {
		return nil, err
	}

	// If password is different than expected
	if isNewPassword {
		cmd.Password = ""
	}

	return loftPod, nil
}

func (cmd *StartCmd) pingLoftRouter(ctx context.Context, loftPod *corev1.Pod) (string, error) {
	loftRouterSecret, err := cmd.KubeClient.CoreV1().Secrets(loftPod.Namespace).Get(ctx, LoftRouterDomainSecret, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return "", nil
		}

		return "", fmt.Errorf("find loft router domain secret: %w", err)
	} else if loftRouterSecret.Data == nil || len(loftRouterSecret.Data["domain"]) == 0 {
		return "", nil
	}

	// get the domain from secret
	loftRouterDomain := string(loftRouterSecret.Data["domain"])

	// wait until loft is reachable at the given url
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	cmd.Log.Infof("Waiting until DevPod Pro is reachable at https://%s", loftRouterDomain)
	err = wait.PollUntilContextTimeout(ctx, time.Second*3, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+loftRouterDomain+"/version", nil)
		if err != nil {
			return false, nil
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return false, nil
		}

		return resp.StatusCode == http.StatusOK, nil
	})
	if err != nil {
		return "", err
	}

	return loftRouterDomain, nil
}

func (cmd *StartCmd) successLoftRouter(url string) error {
	if !cmd.NoLogin {
		err := cmd.login(url)
		if err != nil {
			return err
		}
	}

	url = "https://" + url

	password := cmd.Password
	if password == "" {
		password = passwordChangedHint
	}

	cmd.Log.WriteString(logrus.InfoLevel, fmt.Sprintf(`


##########################   LOGIN   ############################

Username: `+ansi.Color("admin", "green+b")+`
Password: `+ansi.Color(password, "green+b")+`  # Change via UI or via: `+ansi.Color("devpod pro reset password", "green+b")+`

Login via UI:  %s
Login via CLI: %s

#################################################################

DevPod Pro was successfully installed and can now be reached at: %s

Thanks for using DevPod Pro!
`,
		ansi.Color(url, "green+b"),
		ansi.Color("devpod pro login"+" "+url, "green+b"),
		url,
	))
	return nil
}

func (cmd *StartCmd) login(url string) error {
	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// check if we are already logged in
	if cmd.isLoggedIn(url) {
		// still open the UI
		err := open.Run(url)
		if err != nil {
			return fmt.Errorf("couldn't open the login page in a browser: %w", err)
		}

		return nil
	}

	// log into the CLI
	err := cmd.loginViaCLI(url)
	if err != nil {
		return err
	}

	// log into the UI
	err = cmd.loginUI(url)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *StartCmd) loginViaCLI(url string) error {
	loginPath := "%s/auth/password/login"

	loginRequest := auth.PasswordLoginRequest{
		Username: defaultUser,
		Password: cmd.Password,
	}
	loginRequestBytes, err := json.Marshal(loginRequest)
	if err != nil {
		return err
	}
	loginRequestBuf := bytes.NewBuffer(loginRequestBytes)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	resp, err := httpClient.Post(fmt.Sprintf(loginPath, url), "application/json", loginRequestBuf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	accessKey := &auth.AccessKey{}
	err = json.Unmarshal(body, accessKey)
	if err != nil {
		return err
	}

	// log into loft
	loader, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	url = strings.TrimSuffix(url, "/")
	err = loader.LoginWithAccessKey(url, accessKey.AccessKey, true, false)
	if err != nil {
		return err
	}

	cmd.Log.WriteString(logrus.InfoLevel, "\n")
	cmd.Log.Donef("Successfully logged in via CLI into %s", ansi.Color(url, "white+b"))

	return nil
}

func (cmd *StartCmd) loginUI(url string) error {
	queryString := fmt.Sprintf("username=%s&password=%s", defaultUser, netUrl.QueryEscape(cmd.Password))
	loginURL := fmt.Sprintf("%s/login#%s", url, queryString)

	err := open.Run(loginURL)
	if err != nil {
		return fmt.Errorf("couldn't open the login page in a browser: %w", err)
	}

	cmd.Log.Infof("If the browser does not open automatically, please navigate to %s", loginURL)

	return nil
}

func (cmd *StartCmd) isLoggedIn(url string) bool {
	url = strings.TrimPrefix(url, "https://")

	c, err := client.NewClientFromPath(cmd.Config)
	return err == nil && strings.TrimPrefix(strings.TrimSuffix(c.Config().Host, "/"), "https://") == strings.TrimSuffix(url, "/")
}

func uninstall(ctx context.Context, kubeClient kubernetes.Interface, restConfig *rest.Config, kubeContext, namespace string, log log.Logger) error {
	releaseName := "devpod-pro"
	deploy, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, defaultDeploymentName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	} else if deploy != nil && deploy.Labels != nil && deploy.Labels["release"] != "" {
		releaseName = deploy.Labels["release"]
	}

	args := []string{
		"uninstall",
		releaseName,
		"--kube-context",
		kubeContext,
		"--namespace",
		namespace,
	}
	log.Infof("Executing command: helm %s", strings.Join(args, " "))
	output, err := exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		log.Errorf("error during helm command: %s (%v)", string(output), err)
	}

	// we also cleanup the validating webhook configuration and apiservice
	apiRegistrationClient, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	err = apiRegistrationClient.ApiregistrationV1().APIServices().Delete(ctx, "v1.management.loft.sh", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = deleteUser(ctx, restConfig, "admin")
	if err != nil {
		return err
	}

	err = kubeClient.CoreV1().Secrets(namespace).Delete(context.Background(), "loft-user-secret-admin", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = kubeClient.CoreV1().Secrets(namespace).Delete(context.Background(), LoftRouterDomainSecret, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	// we also cleanup the validating webhook configuration and apiservice
	err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, "loft-agent", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = apiRegistrationClient.ApiregistrationV1().APIServices().Delete(ctx, "v1alpha1.tenancy.kiosk.sh", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = apiRegistrationClient.ApiregistrationV1().APIServices().Delete(ctx, "v1.cluster.loft.sh", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, "loft-agent-controller", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, "loft-applied-defaults", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	log.WriteString(logrus.InfoLevel, "\n")
	log.Done("Successfully uninstalled DevPod Pro")
	log.WriteString(logrus.InfoLevel, "\n")

	return nil
}

func isAlreadyInstalled(ctx context.Context, kubeClient kubernetes.Interface, namespace string) (bool, error) {
	_, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, defaultDeploymentName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("error accessing kubernetes cluster: %w", err)
	}

	return true, nil
}

func getDefaultPassword(ctx context.Context, kubeClient kubernetes.Interface, namespace string) (string, error) {
	loftNamespace, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			loftNamespace, err := kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return "", err
			}

			return string(loftNamespace.UID), nil
		}

		return "", err
	}

	return string(loftNamespace.UID), nil
}

func isInstalledLocally(ctx context.Context, kubeClient kubernetes.Interface, namespace string) bool {
	_, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, "loft-ingress", metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		_, err = kubeClient.NetworkingV1beta1().Ingresses(namespace).Get(ctx, "loft-ingress", metav1.GetOptions{})
		return kerrors.IsNotFound(err)
	}

	return kerrors.IsNotFound(err)
}

func enterHostNameQuestion(log log.Logger) (string, error) {
	return log.Question(&survey.QuestionOptions{
		Question: fmt.Sprintf("Enter a hostname for your %s instance (e.g. loft.my-domain.tld): \n ", "DevPod Pro"),
		ValidationFunc: func(answer string) error {
			u, err := netUrl.Parse("https://" + answer)
			if err != nil || u.Path != "" || u.Port() != "" || len(strings.Split(answer, ".")) < 2 {
				return fmt.Errorf("please enter a valid hostname without protocol (https://), without path and without port, e.g. loft.my-domain.tld")
			}
			return nil
		},
	})
}

func ensureIngressController(ctx context.Context, kubeClient kubernetes.Interface, kubeContext string, log log.Logger) error {
	// first create an ingress controller
	const (
		YesOption = "Yes"
		NoOption  = "No, I already have an ingress controller installed."
	)

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Ingress controller required. Should the nginx-ingress controller be installed?",
		DefaultValue: YesOption,
		Options: []string{
			YesOption,
			NoOption,
		},
	})
	if err != nil {
		return err
	}

	if answer == YesOption {
		args := []string{
			"install",
			"ingress-nginx",
			"ingress-nginx",
			"--repository-config=''",
			"--repo",
			"https://kubernetes.github.io/ingress-nginx",
			"--kube-context",
			kubeContext,
			"--namespace",
			"ingress-nginx",
			"--create-namespace",
			"--set-string",
			"controller.config.hsts=false",
			"--wait",
		}
		log.WriteString(logrus.InfoLevel, "\n")
		log.Infof("Executing command: helm %s\n", strings.Join(args, " "))
		log.Info("Waiting for ingress controller deployment, this can take several minutes...")
		helmCmd := exec.Command("helm", args...)
		output, err := helmCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error during helm command: %s (%w)", string(output), err)
		}

		list, err := kubeClient.CoreV1().Secrets("ingress-nginx").List(ctx, metav1.ListOptions{
			LabelSelector: "name=ingress-nginx,owner=helm,status=deployed",
		})
		if err != nil {
			return err
		}

		if len(list.Items) == 1 {
			secret := list.Items[0]
			originalSecret := secret.DeepCopy()
			secret.Labels["loft.sh/app"] = "true"
			if secret.Annotations == nil {
				secret.Annotations = map[string]string{}
			}

			secret.Annotations["loft.sh/url"] = "https://kubernetes.github.io/ingress-nginx"
			originalJSON, err := json.Marshal(originalSecret)
			if err != nil {
				return err
			}
			modifiedJSON, err := json.Marshal(secret)
			if err != nil {
				return err
			}
			data, err := jsonpatch.CreateMergePatch(originalJSON, modifiedJSON)
			if err != nil {
				return err
			}
			_, err = kubeClient.CoreV1().Secrets(secret.Namespace).Patch(ctx, secret.Name, types.MergePatchType, data, metav1.PatchOptions{})
			if err != nil {
				return err
			}
		}

		log.Done("Successfully installed ingress-nginx to your kubernetes cluster!")
	}

	return nil
}

func deleteUser(ctx context.Context, restConfig *rest.Config, name string) error {
	loftClient, err := loftclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	user, err := loftClient.StorageV1().Users().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil
	} else if len(user.Finalizers) > 0 {
		user.Finalizers = nil
		_, err = loftClient.StorageV1().Users().Update(ctx, user, metav1.UpdateOptions{})
		if err != nil {
			if kerrors.IsConflict(err) {
				return deleteUser(ctx, restConfig, name)
			}

			return err
		}
	}

	err = loftClient.StorageV1().Users().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	return nil
}

func ensureAdminPassword(ctx context.Context, kubeClient kubernetes.Interface, restConfig *rest.Config, password string, log log.Logger) (bool, error) {
	loftClient, err := loftclientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	admin, err := loftClient.StorageV1().Users().Get(ctx, "admin", metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return false, err
	} else if admin == nil {
		admin, err = loftClient.StorageV1().Users().Create(ctx, &storagev1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "admin",
			},
			Spec: storagev1.UserSpec{
				Username: "admin",
				Email:    "test@domain.tld",
				Subject:  "admin",
				Groups:   []string{"system:masters"},
				PasswordRef: &storagev1.SecretRef{
					SecretName:      "loft-user-secret-admin",
					SecretNamespace: "loft",
					Key:             "password",
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return false, err
		}
	} else if admin.Spec.PasswordRef == nil || admin.Spec.PasswordRef.SecretName == "" || admin.Spec.PasswordRef.SecretNamespace == "" {
		return false, nil
	}

	key := admin.Spec.PasswordRef.Key
	if key == "" {
		key = "password"
	}

	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))

	secret, err := kubeClient.CoreV1().Secrets(admin.Spec.PasswordRef.SecretNamespace).Get(ctx, admin.Spec.PasswordRef.SecretName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return false, err
	} else if err == nil {
		existingPasswordHash, keyExists := secret.Data[key]
		if keyExists {
			return (string(existingPasswordHash) != passwordHash), nil
		}

		secret.Data[key] = []byte(passwordHash)
		_, err = kubeClient.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "update admin password secret")
		}
		return false, nil
	}

	// create the password secret if it was not found, this can happen if you delete the loft namespace without deleting the admin user
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      admin.Spec.PasswordRef.SecretName,
			Namespace: admin.Spec.PasswordRef.SecretNamespace,
		},
		Data: map[string][]byte{
			key: []byte(passwordHash),
		},
	}
	_, err = kubeClient.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return false, errors.Wrap(err, "create admin password secret")
	}

	log.Info("Successfully recreated admin password secret")
	return false, nil
}

func getIngressHost(ctx context.Context, kubeClient kubernetes.Interface, namespace string) (string, error) {
	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, "loft-ingress", metav1.GetOptions{})
	if err != nil {
		ingress, err := kubeClient.NetworkingV1beta1().Ingresses(namespace).Get(ctx, "loft-ingress", metav1.GetOptions{})
		if err != nil {
			return "", err
		} else {
			// find host
			for _, rule := range ingress.Spec.Rules {
				return rule.Host, nil
			}
		}
	} else {
		// find host
		for _, rule := range ingress.Spec.Rules {
			return rule.Host, nil
		}
	}

	return "", fmt.Errorf("couldn't find any host in loft ingress '%s/loft-ingress', please make sure you have not changed any deployed resources", namespace)
}

type version struct {
	Version string `json:"version"`
}

func isHostReachable(ctx context.Context, host string) (bool, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// we disable http2 as Kubernetes has problems with this
	transport.ForceAttemptHTTP2 = false
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	// wait until loft is reachable at the given url
	client := &http.Client{Transport: transport}
	url := "https://" + host + "/version"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request with context: %w", err)
	}
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		out, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}

		v := &version{}
		err = json.Unmarshal(out, v)
		if err != nil {
			return false, fmt.Errorf("error decoding response from %s: %w. Try running '%s --reset'", url, err, "devpod pro start")
		} else if v.Version == "" {
			return false, fmt.Errorf("unexpected response from %s: %s. Try running '%s --reset'", url, string(out), "devpod pro start")
		}

		return true, nil
	}

	return false, nil
}

func upgradeRelease(chartName, chartRepo, kubeContext, namespace string, extraArgs []string, log log.Logger) error {
	// now we install loft
	args := []string{
		"upgrade",
		defaultReleaseName,
		chartName,
		"--install",
		"--create-namespace",
		"--repository-config=''",
		"--kube-context",
		kubeContext,
		"--namespace",
		namespace,
	}
	if chartRepo != "" {
		args = append(args, "--repo", chartRepo)
	}
	args = append(args, extraArgs...)

	log.WriteString(logrus.InfoLevel, "\n")
	log.Infof("Executing command: helm %s\n", strings.Join(args, " "))
	log.Info("Waiting for helm command, this can take up to several minutes...")
	helmCmd := exec.Command("helm", args...)
	if chartRepo != "" {
		helmWorkDir, err := getHelmWorkdir(chartName)
		if err != nil {
			return err
		}

		helmCmd.Dir = helmWorkDir
	}
	output, err := helmCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error during helm command: %s (%w)", string(output), err)
	}

	log.Donef("DevPod Pro has been deployed to your cluster!")
	return nil
}

func getReleaseManifests(chartName, chartRepo, kubeContext, namespace string, extraArgs []string, _ log.Logger) (string, error) {
	args := []string{
		"template",
		defaultReleaseName,
		chartName,
		"--repository-config=''",
		"--kube-context",
		kubeContext,
		"--namespace",
		namespace,
	}
	if chartRepo != "" {
		args = append(args, "--repo", chartRepo)
	}
	args = append(args, extraArgs...)

	helmCmd := exec.Command("helm", args...)
	if chartRepo != "" {
		helmWorkDir, err := getHelmWorkdir(chartName)
		if err != nil {
			return "", err
		}

		helmCmd.Dir = helmWorkDir
	}
	output, err := helmCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error during helm command: %s (%w)", string(output), err)
	}
	return string(output), nil
}

func getHelmWorkdir(chartName string) (string, error) {
	// If chartName folder exists, check temp dir next
	if _, err := os.Stat(chartName); err == nil {
		tempDir := os.TempDir()

		// If tempDir/chartName folder exists, create temp folder
		if _, err := os.Stat(path.Join(tempDir, chartName)); err == nil {
			tempDir, err = os.MkdirTemp(tempDir, chartName)
			if err != nil {
				return "", errors.New("problematic directory `" + chartName + "` found: please execute command in a different folder")
			}
		}

		// Use tempDir
		return tempDir, nil
	}

	// Use current workdir
	return "", nil
}

var (
	ErrMissingContainer = errors.New("missing container")
	ErrLoftNotReachable = errors.New("DevPod Pro is not reachable")
)

type ContainerDetails struct {
	NetworkSettings ContainerNetworkSettings `json:"NetworkSettings,omitempty"`
	State           ContainerDetailsState    `json:"State,omitempty"`
	ID              string                   `json:"ID,omitempty"`
	Created         string                   `json:"Created,omitempty"`
	Config          ContainerDetailsConfig   `json:"Config,omitempty"`
}

type ContainerNetworkSettings struct {
	Ports map[string][]ContainerPort `json:"ports,omitempty"`
}

type ContainerPort struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

type ContainerDetailsConfig struct {
	Labels map[string]string `json:"Labels,omitempty"`
	Image  string            `json:"Image,omitempty"`
	User   string            `json:"User,omitempty"`
	Env    []string          `json:"Env,omitempty"`
}

type ContainerDetailsState struct {
	Status    string `json:"Status,omitempty"`
	StartedAt string `json:"StartedAt,omitempty"`
}

func WrapCommandError(stdout []byte, err error) error {
	if err == nil {
		return nil
	}

	return &Error{
		stdout: stdout,
		err:    err,
	}
}

type Error struct {
	err    error
	stdout []byte
}

func (e *Error) Error() string {
	message := ""
	if len(e.stdout) > 0 {
		message += string(e.stdout) + "\n"
	}

	var exitError *exec.ExitError
	if errors.As(e.err, &exitError) && len(exitError.Stderr) > 0 {
		message += string(exitError.Stderr) + "\n"
	}

	return message + e.err.Error()
}

func getMachineUID(log log.Logger) string {
	id, err := machineid.ID()
	if err != nil {
		id = "error"
		if log != nil {
			log.Debugf("Error retrieving machine uid: %v", err)
		}
	}
	// get $HOME to distinguish two users on the same machine
	// will be hashed later together with the ID
	home, err := util.UserHomeDir()
	if err != nil {
		home = "error"
		if log != nil {
			log.Debugf("Error retrieving machine home: %v", err)
		}
	}
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(home))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
