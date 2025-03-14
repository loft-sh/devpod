package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type Client struct {
	client *kubernetes.Clientset

	config *rest.Config
}

// NewClient constructs a struct wrapping the kubernetes client that is used by the kubernetes driver
func NewClient(kubeConfig, kubeContext string) (*Client, string, error) {
	if kubeConfig == "" {
		kubeConfig = os.Getenv("KUBECONFIG")
	}

	// create client config loading rules
	var clientConfigLoadingRules *clientcmd.ClientConfigLoadingRules
	if kubeConfig != "" {
		clientConfigLoadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfig}
	} else {
		clientConfigLoadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	// load kubernetes config
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientConfigLoadingRules,
		&clientcmd.ConfigOverrides{CurrentContext: kubeContext},
	)

	clientConfig, err := config.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load kubernetes config: %w", err)
	}

	namespace, _, err := config.Namespace()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load kubernetes namespace from config: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, "", err
	}

	return &Client{
		client: kubeClient,
		config: clientConfig,
	}, namespace, nil
}

func (c *Client) Client() *kubernetes.Clientset {
	return c.client
}

func (c *Client) Config() *rest.Config {
	return c.config
}

func (c *Client) FullLogs(ctx context.Context, namespace, pod, container string) ([]byte, error) {
	logs, err := c.Logs(ctx, namespace, pod, container, true)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(logs)
}

func (c *Client) Logs(ctx context.Context, namespace, pod, container string, follow bool) (io.ReadCloser, error) {
	return c.client.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		Follow:    follow,
	}).Stream(ctx)
}

type ExecStreamOptions struct {
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Pod       string
	Namespace string
	Container string
	Command   []string
}

// Exec executes a kubectl exec with given transport round tripper and upgrader
func (c *Client) Exec(ctx context.Context, options *ExecStreamOptions) error {
	client, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return err
	}

	execRequest := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.Pod).
		Namespace(options.Namespace).
		SubResource(string("exec")).
		VersionedParams(&corev1.PodExecOptions{
			Container: options.Container,
			Command:   options.Command,
			Stdin:     options.Stdin != nil,
			Stdout:    options.Stdout != nil,
			Stderr:    options.Stderr != nil,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", execRequest.URL())
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		errChan <- exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  options.Stdin,
			Stdout: options.Stdout,
			Stderr: options.Stderr,
		})
	}()

	select {
	case <-ctx.Done():
		<-errChan
		return nil
	case err = <-errChan:
		return err
	}
}
