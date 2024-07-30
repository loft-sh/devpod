package provider

import (
	"cmp"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/loft"
	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/devpod/pkg/loft/project"
	"github.com/loft-sh/devpod/pkg/random"
	"github.com/loft-sh/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store/mem"
	"tailscale.com/tsnet"
	tslogger "tailscale.com/types/logger"
)

const ProviderBinaryEnv = "PRO_PROVIDER"

type WorkspaceInfo struct {
	ID          string
	UID         string
	ProjectName string
}

func GetWorkspaceInfoFromEnv() (*WorkspaceInfo, error) {
	workspaceInfo := &WorkspaceInfo{}
	// get workspace id
	workspaceID := os.Getenv(loft.WorkspaceIDEnv)
	if workspaceID == "" {
		return nil, fmt.Errorf("%s is missing in environment", loft.WorkspaceIDEnv)
	}
	workspaceInfo.ID = workspaceID

	// get workspace uid
	workspaceUID := os.Getenv(loft.WorkspaceUIDEnv)
	if workspaceUID == "" {
		return nil, fmt.Errorf("%s is missing in environment", loft.WorkspaceUIDEnv)
	}
	workspaceInfo.UID = workspaceUID

	// get project
	projectName := os.Getenv(loft.ProjectEnv)
	if projectName == "" {
		return nil, fmt.Errorf("%s is missing in environment", loft.ProjectEnv)
	}
	workspaceInfo.ProjectName = projectName

	return workspaceInfo, nil
}

func FindWorkspace(ctx context.Context, baseClient client.Client, uid, projectName string) (*managementv1.DevPodWorkspaceInstance, error) {
	// create client
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, fmt.Errorf("create management client: %w", err)
	}

	// get workspace
	workspaceList, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(project.ProjectNamespace(projectName)).List(ctx, metav1.ListOptions{
		LabelSelector: storagev1.DevPodWorkspaceUIDLabel + "=" + uid,
	})
	if err != nil {
		return nil, err
	} else if len(workspaceList.Items) == 0 {
		return nil, nil
	}

	return &workspaceList.Items[0], nil
}

func FindWorkspaceByName(ctx context.Context, baseClient client.Client, name, projectName string) (*managementv1.DevPodWorkspaceInstance, error) {
	// create client
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, fmt.Errorf("create management client: %w", err)
	}

	// get workspace
	workspace, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(project.ProjectNamespace(projectName)).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

func OptionsFromEnv(name string) url.Values {
	options := os.Getenv(name)
	if options != "" {
		return url.Values{
			"options": []string{options},
		}
	}

	return nil
}

func DialWorkspace(baseClient client.Client, workspace *managementv1.DevPodWorkspaceInstance, subResource string, values url.Values) (*websocket.Conn, error) {
	restConfig, err := baseClient.ManagementConfig()
	if err != nil {
		return nil, err
	}

	host := restConfig.Host

	// check if this workspace has been scheduled to run on a specific runner
	if workspace.Annotations != nil && workspace.Annotations[storagev1.DevPodWorkspaceRunnerNetworkPeerAnnotation] != "" {
		networkPeerName := workspace.Annotations[storagev1.DevPodWorkspaceRunnerNetworkPeerAnnotation]
		host = networkPeerName + ".ts.loft" // tailscale.BaseDomain
	}

	parsedURL, _ := url.Parse(host)
	if parsedURL != nil && parsedURL.Host != "" {
		host = parsedURL.Host
	}

	loftURL := "wss://" + host + "/kubernetes/management/apis/management.loft.sh/v1/namespaces/" + workspace.Namespace + "/devpodworkspaceinstances/" + workspace.Name + "/" + subResource
	if len(values) > 0 {
		loftURL += "?" + values.Encode()
	}
	l := log.GetInstance().ErrorStreamOnly()
	l.Info("\n")
	l.Info("******* Dial Workspace ********")
	l.Info("\n")
	defer func() {
		l.Info("\n")
		l.Info("******* Dial Workspace ********")
		l.Info("\n")
	}()

	// TODO: We need to create a server here
	// Then have this server connect to controlplan
	// And dial remote runner
	u, _ := url.Parse(restConfig.Host)
	baseUrl := url.URL{
		Scheme: cmp.Or(os.Getenv("LOFT_TSNET_SCHEME"), "https"),
		Host:   u.Host,
		Path:   "/coordinator/",
	}
	insecure := true
	if insecure {
		if err := os.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true"); err != nil {
			return nil, fmt.Errorf("failed to set insecure env var: %w", err)
		}

		envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	}

	if err := checkDerpConnection(context.TODO(), &baseUrl, l); err != nil {
		return nil, fmt.Errorf("failed to check derp connection: %w", err)
	}

	// podname := os.Getenv("HOSTNAME")
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("retrieve hostname: %w", err)
	}
	// TODO: What would be a good hostname scheme?
	hostname = "pascal" + "." + "workspace"
	store, _ := mem.New(tslogger.Discard, "")
	// TODO: Implement me
	// Which scopes do we need here?
	accessKey := cmp.Or(os.Getenv("TS_ACCESS_KEY"), "BoTKX3sxOQxDwjSxvs91KNUPG5itwurwZoeateQnOxnGNOuFIEegeNkak3COjKbf")

	l.Info("BaseURL", baseUrl.String())
	tsServer := &tsnet.Server{
		Hostname:   hostname,
		Logf:       tsnetLogger(l),
		AuthKey:    accessKey,
		ControlURL: baseUrl.String(),
		// TODO: How to get auth key? Can be loaded from network peer?
		// Can this be access key? Does it need certain permissions?
		Dir:       "/tmp/tailscale/" + random.String(5),
		Store:     store,
		Ephemeral: true,
	}

	err = tsServer.Start()
	if err != nil {
		return nil, fmt.Errorf("start tailscale: %w", err)
	}
	err = waitForServer(context.TODO(), tsServer, l)
	if err != nil {
		return nil, fmt.Errorf("wait for server: %w", err)
	}

	// return nil, fmt.Errorf("bail")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = tsServer.Dial
	client := devpodhttp.GetHTTPClient()
	client.Transport = transport
	req, _ := http.NewRequest(http.MethodGet, host+"/ping", nil)
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("req: %w", err)
	}
	defer res.Body.Close()
	l.Info("code", res.StatusCode)
	o, _ := io.ReadAll(res.Body)
	l.Info(string(o))

	return nil, fmt.Errorf("bail")
	// dialer := websocket.Dialer{
	// 	// TODO: Try different configurations and check if methods are being called
	// 	// TODO: Try to send health check to runner?
	// 	TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	// 	Proxy:            http.ProxyFromEnvironment,
	// 	HandshakeTimeout: 45 * time.Second,
	// 	NetDialContext:   tsServer.Dial,
	// }
	//
	// conn, response, err := dialer.Dial(loftURL, map[string][]string{
	// 	"Authorization": {"Bearer " + restConfig.BearerToken},
	// })
	// if err != nil {
	// 	if response != nil {
	// 		out, _ := io.ReadAll(response.Body)
	// 		headers, _ := json.Marshal(response.Header)
	// 		return nil, fmt.Errorf("error dialing websocket %s (code %d): headers - %s, response - %s, error - %w", loftURL, response.StatusCode, string(headers), string(out), err)
	// 	}
	//
	// 	return nil, fmt.Errorf("error dialing websocket %s: %w", loftURL, err)
	// }

	// return conn, nil
}

func checkDerpConnection(ctx context.Context, baseUrl *url.URL, log log.Logger) error {
	newTransport := http.DefaultTransport.(*http.Transport).Clone()
	newTransport.TLSClientConfig = &tls.Config{
		// TODO: Expose
		// InsecureSkipVerify: os.Getenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY") == "true",
		InsecureSkipVerify: true,
	}

	c := &http.Client{
		Transport: newTransport,
		Timeout:   5 * time.Second,
	}

	derpUrl := *baseUrl
	derpUrl.Path = "/derp/probe"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, derpUrl.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := c.Do(req)
	if err != nil || (res != nil && res.StatusCode != http.StatusOK) {
		// TODO: Adjust logging here
		log.Warn(err, "Failed to reach the coordinator server. Make sure that the agent can reach the control-plane. Also, make sure to try using `insecureSkipVerify` or `additionalCA` in the control-plane's helm values in case of x509 certificate issues.", "derpUrl", derpUrl.String())

		if res != nil {
			body, _ := io.ReadAll(res.Body)
			defer res.Body.Close()

			log.Warn("Details", "error", err, "statusCode", res.StatusCode, "body", string(body))
		}

		return fmt.Errorf("failed to reach the coordinator server: %w", err)
	}

	return nil
}

func waitForServer(ctx context.Context, server *tsnet.Server, log log.Logger) error {
	failCounter := 0
	for {
		select {
		case <-ctx.Done():
			err := server.Close()
			if err != nil && !errors.Is(err, net.ErrClosed) {
				return err
			}

			return nil

		case <-time.After(6 * time.Second):
			if failCounter > 10 {
				return fmt.Errorf("control plane tsnet server is not running")
			}

			lc, err := server.LocalClient()
			if err != nil {
				log.Error(err, "Failed to get local client")
				failCounter++
				continue
			}

			status, err := lc.Status(ctx)
			if err != nil {
				log.Error(err, "Failed to get status from local client")
				failCounter++
				continue
			}

			if status.Self == nil {
				log.Error(err, "Failed to get self status from local client")
				failCounter++
				continue
			}

			if status.Self.Online && status.Self.InNetworkMap {
				failCounter = 0
				// o, _ := json.MarshalIndent(status, "", "  ")
				// log.Info(string(o))
				// return nil
			} else {
				failCounter++
				log.Info("Control plane tsnet server is not online", "status", status)
			}
		}
	}
}

func tsnetLogger(log log.Logger) tslogger.Logf {
	logf := tslogger.Discard
	if os.Getenv("LOFT_LOG_TSNET") == "true" {
		logf = func(s string, a ...any) {
			log.Infof("[tailscale] "+s, a...)
		}
	}

	return logf
}
