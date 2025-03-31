package client

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/auth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loft-sh/devpod/pkg/platform/kube"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/devpod/pkg/version"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var CacheFolder = ".loft"

// DefaultCacheConfig is the path to the config
var DefaultCacheConfig = "config.json"

// Copied from platform `pkg/authentication/registry.go`
var ErrAccessKeyNotFound = fmt.Errorf("platform access key not found. Please login again")

const (
	VersionPath   = "%s/version"
	LoginPath     = "%s/login?cli=true"
	RedirectPath  = "%s/spaces"
	AccessKeyPath = "%s/profile/access-keys"
	RefreshToken  = time.Minute * 30
)

func init() {
	hd, _ := util.UserHomeDir()
	if folder, ok := os.LookupEnv("LOFT_CACHE_FOLDER"); ok {
		CacheFolder = filepath.Join(hd, folder)
	} else {
		CacheFolder = filepath.Join(hd, CacheFolder)
	}
	DefaultCacheConfig = filepath.Join(CacheFolder, DefaultCacheConfig)
}

type Client interface {
	Management() (kube.Interface, error)
	ManagementConfig() (*rest.Config, error)
	RefreshSelf(ctx context.Context) error
	Self() *managementv1.Self

	Login(host string, insecure bool, log log.Logger) error
	LoginWithAccessKey(host, accessKey string, insecure bool, force bool) error
	LoginRaw(host, accessKey string, insecure bool) error

	Logout(ctx context.Context) error

	Version() (*auth.Version, error)
	Config() *Config
	Save() error
}

func NewClient() Client {
	return &client{
		config: &Config{},
	}
}

func NewClientFromConfig(c *Config) Client {
	return &client{
		config: c,
	}
}

func NewClientFromPath(path string) (Client, error) {
	c := &client{
		configPath: path,
	}

	err := c.initConfig()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func InitClientFromPath(ctx context.Context, path string) (Client, error) {
	c, err := NewClientFromPath(path)
	if err != nil {
		return nil, err
	}

	err = c.RefreshSelf(ctx)
	if err != nil {
		return nil, err
	}

	return c, nil
}

type client struct {
	config     *Config
	configPath string
	configOnce sync.Once

	self *managementv1.Self
}

func (c *client) RefreshSelf(ctx context.Context) error {
	managementClient, err := c.Management()
	if err != nil {
		return fmt.Errorf("create mangement client: %w", err)
	}

	c.self, err = managementClient.Loft().ManagementV1().Selves().Create(ctx, &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error trying to reach platform: %w. This usually indicates you either have no connection to the platform or are not authenticated", err)
	}

	projectNamespacePrefix := project.DefaultProjectNamespacePrefix
	if c.self.Status.ProjectNamespacePrefix != nil {
		projectNamespacePrefix = *c.self.Status.ProjectNamespacePrefix
	}

	project.SetProjectNamespacePrefix(projectNamespacePrefix)

	return nil
}

func (c *client) Self() *managementv1.Self {
	return c.self.DeepCopy()
}

// Logout implements Client.
func (c *client) Logout(ctx context.Context) error {
	managementClient, err := c.Management()
	if err != nil {
		return fmt.Errorf("create management client: %w", err)
	}

	self, err := managementClient.Loft().ManagementV1().Selves().Create(ctx, &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("get self: %w", err)
	}

	if self.Status.AccessKey != "" && self.Status.AccessKeyType == storagev1.AccessKeyTypeLogin {
		err = managementClient.Loft().ManagementV1().OwnedAccessKeys().Delete(ctx, self.Status.AccessKey, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("delete access key: %w", err)
		}
	}

	return nil
}

func (c *client) initConfig() error {
	var retErr error
	c.configOnce.Do(func() {
		// load the config or create new one if not found
		content, err := os.ReadFile(c.configPath)
		if err != nil {
			if os.IsNotExist(err) {
				c.config = NewConfig()
				return
			}

			retErr = err
			return
		}

		config := &Config{
			VirtualClusterAccessPointCertificates: make(map[string]VirtualClusterCertificatesEntry),
		}
		err = json.Unmarshal(content, config)
		if err != nil {
			retErr = err
			return
		}

		c.config = config
	})

	return retErr
}

func (c *client) Save() error {
	if c.configPath == "" {
		return nil
	}
	if c.config == nil {
		return perrors.New("no config to write")
	}
	if c.config.Kind == "" {
		c.config.Kind = "Config"
	}
	if c.config.APIVersion == "" {
		c.config.APIVersion = "storage.loft.sh/v1"
	}

	err := os.MkdirAll(filepath.Dir(c.configPath), 0o755)
	if err != nil {
		return err
	}

	out, err := json.Marshal(c.config)
	if err != nil {
		return err
	}

	return os.WriteFile(c.configPath, out, 0o660)
}

func (c *client) ManagementConfig() (*rest.Config, error) {
	return c.restConfig("/kubernetes/management")
}

func (c *client) Management() (kube.Interface, error) {
	restConfig, err := c.ManagementConfig()
	if err != nil {
		return nil, err
	}

	return kube.NewForConfig(restConfig)
}

func (c *client) Config() *Config {
	return c.config
}

type keyStruct struct {
	Key string
}

func verifyHost(host string) error {
	if !strings.HasPrefix(host, "https") {
		return fmt.Errorf("cannot log into a non https loft instance '%s', please make sure you have TLS enabled", host)
	}

	return nil
}

func (c *client) Version() (*auth.Version, error) {
	restConfig, err := c.restConfig("")
	if err != nil {
		return nil, err
	}

	restClient, err := kube.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	raw, err := restClient.CoreV1().RESTClient().Get().RequestURI("/version").DoRaw(context.Background())
	if err != nil {
		return nil, perrors.New(fmt.Sprintf("%s\n\nYou may need to login again via `%s login %s --insecure` to allow self-signed certificates\n", err.Error(), os.Args[0], restConfig.Host))
	}

	version := &auth.Version{}
	err = json.Unmarshal(raw, version)
	if err != nil {
		return nil, perrors.Wrap(err, "parse version response")
	}

	return version, nil
}

func (c *client) Login(host string, insecure bool, log log.Logger) error {
	var (
		loginUrl   = fmt.Sprintf(LoginPath, host)
		key        keyStruct
		keyChannel = make(chan keyStruct)
	)

	err := verifyHost(host)
	if err != nil {
		return err
	}

	server := startServer(fmt.Sprintf(RedirectPath, host), keyChannel, log)
	err = open.Run(fmt.Sprintf(LoginPath, host))
	if err != nil {
		return fmt.Errorf("couldn't open the login page in a browser: %w. Please use the --access-key flag for the login command. You can generate an access key here: %s", err, fmt.Sprintf(AccessKeyPath, host))
	} else {
		log.Infof("If the browser does not open automatically, please navigate to %s", loginUrl)
		msg := "If you have problems logging in, please navigate to %s/profile/access-keys, click on 'Create Access Key' and then login via '%s %s --access-key ACCESS_KEY"
		if insecure {
			msg += " --insecure"
		}
		msg += "'"
		log.Infof(msg, host, "devpod pro login", host)
		log.Infof("Logging into DevPod Pro...")

		key = <-keyChannel
	}

	go func() {
		err = server.Shutdown(context.Background())
		if err != nil {
			log.Debugf("Error shutting down server: %v", err)
		}
	}()

	close(keyChannel)
	return c.LoginWithAccessKey(host, key.Key, insecure, false)
}

func (c *client) LoginRaw(host, accessKey string, insecure bool) error {
	if c.config.Host == host && c.config.AccessKey == accessKey {
		return nil
	}

	c.config.Host = host
	c.config.Insecure = insecure
	c.config.AccessKey = accessKey
	return c.Save()
}

func (c *client) LoginWithAccessKey(host, accessKey string, insecure bool, force bool) error {
	err := verifyHost(host)
	if err != nil {
		return err
	}
	if !force && c.config.Host == host && c.config.AccessKey == accessKey {
		return nil
	}

	// delete old access key if were logged in before
	if !force && c.config.AccessKey != "" {
		managementClient, err := c.Management()
		if err == nil {
			self, err := managementClient.Loft().ManagementV1().Selves().Create(context.TODO(), &managementv1.Self{}, metav1.CreateOptions{})
			if err == nil && self.Status.AccessKey != "" && self.Status.AccessKeyType == storagev1.AccessKeyTypeLogin {
				_ = managementClient.Loft().ManagementV1().OwnedAccessKeys().Delete(context.TODO(), self.Status.AccessKey, metav1.DeleteOptions{})
			}
		}
	}

	c.config.Host = host
	c.config.Insecure = insecure
	c.config.AccessKey = accessKey

	// verify the connection works
	managementClient, err := c.Management()
	if err != nil {
		return perrors.Wrap(err, "create management client")
	}

	// try to get self
	_, err = managementClient.Loft().ManagementV1().Selves().Create(context.TODO(), &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) {
			var err x509.UnknownAuthorityError
			if errors.As(urlError.Err, &err) {
				return fmt.Errorf("unsafe login endpoint '%s', if you wish to login into an insecure loft endpoint run with the '--insecure' flag", c.config.Host)
			}
		}

		return perrors.Errorf("error logging in: %v", err)
	}

	return c.Save()
}

// VerifyVersion checks if the Loft version is compatible with this CLI version
func VerifyVersion(baseClient Client) error {
	v, err := baseClient.Version()
	if err != nil {
		return err
	} else if v.Version == "v0.0.0" {
		return nil
	}

	backendMajor, err := strconv.Atoi(v.Major)
	if err != nil {
		return perrors.Wrap(err, "parse major version string")
	}

	cliVersionStr := version.GetVersion()
	cliVersionStr = strings.TrimPrefix(cliVersionStr, "v")
	if cliVersionStr == "" {
		return nil
	}

	cliVersion, err := semver.Parse(cliVersionStr)
	if err != nil {
		return err
	}

	if int(cliVersion.Major) > backendMajor {
		return fmt.Errorf("unsupported %[1]s version %[2]s. Please downgrade your CLI to below v%[3]d.0.0 to support this version, as %[1]s v%[3]d.0.0 and newer versions are incompatible with v%[4]d.x.x", "DevPod Pro", v.Version, cliVersion.Major, backendMajor)
	} else if int(cliVersion.Major) < backendMajor {
		return fmt.Errorf("unsupported %[1]s version %[2]s. Please upgrade your CLI to v%[3]d.0.0 or above to support this version, as %[1]s v%[3]d.0.0 and newer versions are incompatible with v%[4]d.x.x", "DevPod Pro", v.Version, backendMajor, cliVersion.Major)
	}

	return nil
}

func (c *client) restConfig(hostSuffix string) (*rest.Config, error) {
	if c.config == nil {
		return nil, perrors.New("no config loaded")
	} else if c.config.Host == "" || c.config.AccessKey == "" {
		return nil, perrors.New(fmt.Sprintf("not logged in, please make sure you have run '%s' to create one or '%s [%s]' if one already exists", "devpod pro start", "devpod pro login", "devpod-pro-url"))
	}

	// build a rest config
	config, err := GetRestConfig(c.config.Host+hostSuffix, c.config.AccessKey, c.config.Insecure)
	if err != nil {
		return nil, err
	}

	return config, err
}

func GetKubeConfig(host, token, namespace string, insecure bool) clientcmd.ClientConfig {
	contextName := "local"
	kubeConfig := clientcmdapi.NewConfig()
	kubeConfig.Contexts = map[string]*clientcmdapi.Context{
		contextName: {
			Cluster:   contextName,
			AuthInfo:  contextName,
			Namespace: namespace,
		},
	}
	kubeConfig.Clusters = map[string]*clientcmdapi.Cluster{
		contextName: {
			Server:                host,
			InsecureSkipTLSVerify: insecure,
		},
	}
	kubeConfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		contextName: {
			Token: token,
		},
	}
	kubeConfig.CurrentContext = contextName
	return clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{})
}

func GetRestConfig(host, token string, insecure bool) (*rest.Config, error) {
	config, err := GetKubeConfig(host, token, "", insecure).ClientConfig()
	if err != nil {
		return nil, err
	}
	config.UserAgent = "devpod/" + version.GetVersion()

	return config, nil
}

func startServer(redirectURI string, keyChannel chan keyStruct, log log.Logger) *http.Server {
	srv := &http.Server{Addr: ":25843"}

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["key"]
		if !ok || len(keys[0]) == 0 {
			log.Warn("Login: the key used to login is not valid")
			return
		}

		keyChannel <- keyStruct{
			Key: keys[0],
		}
		http.Redirect(w, r, redirectURI, http.StatusSeeOther)
	})

	go func() {
		// cannot panic, because this probably is an intentional close
		_ = srv.ListenAndServe()
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
