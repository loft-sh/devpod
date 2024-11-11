package add

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterCmd struct {
	Log log.Logger
	*proflags.GlobalFlags

	Namespace        string
	ServiceAccount   string
	DisplayName      string
	KubeContext      string
	Insecure         bool
	Wait             bool
	HelmChartPath    string
	HelmChartVersion string
	HelmSet          []string
	HelmValues       []string
	Host             string
}

// NewClusterCmd creates a new command
func NewClusterCmd(globalFlags *proflags.GlobalFlags) *cobra.Command {
	cmd := &ClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	c := &cobra.Command{
		Use:   "cluster <cluster-name>",
		Short: "add current cluster to DevPod Pro",
		Args:  cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.Namespace, "namespace", "loft", "The namespace to generate the service account in. The namespace will be created if it does not exist")
	c.Flags().StringVar(&cmd.ServiceAccount, "service-account", "loft-admin", "The service account name to create")
	c.Flags().StringVar(&cmd.DisplayName, "display-name", "", "The display name to show in the UI for this cluster")
	c.Flags().BoolVar(&cmd.Wait, "wait", false, "If true, will wait until the cluster is initialized")
	c.Flags().BoolVar(&cmd.Insecure, "insecure", false, "If true, deploys the agent in insecure mode")
	c.Flags().StringVar(&cmd.HelmChartVersion, "helm-chart-version", "", "The agent chart version to deploy")
	c.Flags().StringVar(&cmd.HelmChartPath, "helm-chart-path", "", "The agent chart to deploy")
	c.Flags().StringArrayVar(&cmd.HelmSet, "helm-set", []string{}, "Extra helm values for the agent chart")
	c.Flags().StringArrayVar(&cmd.HelmValues, "helm-values", []string{}, "Extra helm values for the agent chart")
	c.Flags().StringVar(&cmd.KubeContext, "kube-context", "", "The kube context to use for installation")
	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")

	return c
}

func (cmd *ClusterCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, "")
	if err != nil {
		return err
	}

	cmd.Host, err = ensureHost(devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return err
	}

	// Get clusterName from command argument
	clusterName := args[0]

	baseClient, err := platform.InitClientFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	loftVersion, err := baseClient.Version()
	if err != nil {
		return fmt.Errorf("get pro version: %w", err)
	}

	user, team := getUserOrTeam(ctx, baseClient)

	_, err = managementClient.Loft().ManagementV1().Clusters().Create(ctx, &managementv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: managementv1.ClusterSpec{
			ClusterSpec: storagev1.ClusterSpec{
				DisplayName: cmd.DisplayName,
				Owner: &storagev1.UserOrTeam{
					User: user,
					Team: team,
				},
				NetworkPeer: true,
				Access:      getAccess(user, team),
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create cluster: %w", err)
	}

	accessKey, err := managementClient.Loft().ManagementV1().Clusters().GetAccessKey(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get cluster access key: %w", err)
	}

	namespace := cmd.Namespace

	helmArgs := []string{
		"upgrade", "loft",
	}

	if os.Getenv("DEVELOPMENT") == "true" {
		helmArgs = []string{
			"upgrade", "--install", "loft", cmp.Or(os.Getenv("DEVELOPMENT_CHART_DIR"), "./chart"),
			"--create-namespace",
			"--namespace", namespace,
			"--set", "agentOnly=true",
			"--set", "image=" + cmp.Or(os.Getenv("DEVELOPMENT_IMAGE"), "ghcr.io/loft-sh/enterprise:release-test"),
		}
	} else {
		if cmd.HelmChartPath != "" {
			helmArgs = append(helmArgs, cmd.HelmChartPath)
		} else {
			helmArgs = append(helmArgs, "loft", "--repo", "https://charts.loft.sh")
		}

		if loftVersion.Version != "" {
			helmArgs = append(helmArgs, "--version", loftVersion.Version)
		}

		if cmd.HelmChartVersion != "" {
			helmArgs = append(helmArgs, "--version", cmd.HelmChartVersion)
		}

		// general arguments
		helmArgs = append(helmArgs, "--install", "--create-namespace", "--namespace", cmd.Namespace, "--set", "agentOnly=true")
	}

	for _, set := range cmd.HelmSet {
		helmArgs = append(helmArgs, "--set", set)
	}
	for _, values := range cmd.HelmValues {
		helmArgs = append(helmArgs, "--values", values)
	}

	if accessKey.LoftHost != "" {
		helmArgs = append(helmArgs, "--set", "url="+accessKey.LoftHost)
	}

	if accessKey.AccessKey != "" {
		helmArgs = append(helmArgs, "--set", "token="+accessKey.AccessKey)
	}

	if cmd.Insecure || accessKey.Insecure {
		helmArgs = append(helmArgs, "--set", "insecureSkipVerify=true")
	}

	if accessKey.CaCert != "" {
		helmArgs = append(helmArgs, "--set", "additionalCA="+accessKey.CaCert)
	}

	if cmd.Wait {
		helmArgs = append(helmArgs, "--wait")
	}

	if cmd.KubeContext != "" {
		helmArgs = append(helmArgs, "--kube-context", cmd.KubeContext)
	}

	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	if cmd.KubeContext != "" {
		kubeConfig, err := kubeClientConfig.RawConfig()
		if err != nil {
			return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
		}

		kubeClientConfig = clientcmd.NewNonInteractiveClientConfig(kubeConfig, cmd.KubeContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules())
	}

	config, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create kube client: %w", err)
	}

	errChan := make(chan error)

	go func() {
		helmCmd := exec.CommandContext(ctx, "helm", helmArgs...)

		helmCmd.Stdout = cmd.Log.Writer(logrus.DebugLevel, true)
		helmCmd.Stderr = cmd.Log.Writer(logrus.DebugLevel, true)
		helmCmd.Stdin = os.Stdin

		cmd.Log.Info("Installing agent...")
		cmd.Log.Debugf("Running helm command: %v", helmCmd.Args)

		err = helmCmd.Run()
		if err != nil {
			errChan <- fmt.Errorf("failed to install chart: %w", err)
		}

		close(errChan)
	}()

	_, err = platform.WaitForPodReady(ctx, clientset, namespace, cmd.Log)
	if err = errors.Join(err, <-errChan); err != nil {
		return fmt.Errorf("wait for pod: %w", err)
	}

	if cmd.Wait {
		cmd.Log.Info("Waiting for the cluster to be initialized...")
		waitErr := wait.PollUntilContextTimeout(ctx, time.Second, 5*time.Minute, false, func(ctx context.Context) (done bool, err error) {
			clusterInstance, err := managementClient.Loft().ManagementV1().Clusters().Get(ctx, clusterName, metav1.GetOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return false, err
			}

			return clusterInstance != nil && clusterInstance.Status.Phase == storagev1.ClusterStatusPhaseInitialized, nil
		})
		if waitErr != nil {
			return fmt.Errorf("get cluster: %w", waitErr)
		}
	}

	cmd.Log.Donef("Successfully added cluster %s", clusterName)

	return nil
}

func ensureHost(devPodConfig *config.Config, host string, log log.Logger) (string, error) {
	if host != "" {
		return host, nil
	}

	proInstances, err := workspace.ListProInstances(devPodConfig, log)
	if err != nil {
		return "", fmt.Errorf("list pro instances: %w", err)
	}
	options := []string{}
	for _, pro := range proInstances {
		options = append(options, pro.Host)
	}
	h, err := log.Question(&survey.QuestionOptions{
		Question:     "Select Pro instance to connect your cluster to",
		Options:      options,
		DefaultValue: options[0],
	})
	if err != nil {
		return "", fmt.Errorf("select pro instance: %w", err)
	}

	return h, nil
}

func getUserOrTeam(ctx context.Context, baseClient client.Client) (string, string) {
	var user, team string

	self := baseClient.Self()
	userName := self.Status.User
	teamName := self.Status.Team

	if userName != nil {
		user = userName.Name
	} else {
		team = teamName.Name
	}

	return user, team
}

func getAccess(user, team string) []storagev1.Access {
	access := []storagev1.Access{
		{
			Verbs:        []string{"*"},
			Subresources: []string{"*"},
		},
	}

	if team != "" {
		access[0].Teams = []string{team}
	} else {
		access[0].Users = []string{user}
	}

	return access
}
