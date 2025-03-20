package pro

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/blang/semver"
	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	providercmd "github.com/loft-sh/devpod/cmd/provider"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	versionpkg "github.com/loft-sh/devpod/pkg/version"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const PROVIDER_BINARY = "PRO_PROVIDER"
const providerRepo = "loft-sh/devpod"

// LoginCmd holds the login cmd flags
type LoginCmd struct {
	proflags.GlobalFlags

	AccessKey      string
	Provider       string
	Version        string
	ProviderSource string

	Options []string

	Login        bool
	Use          bool
	ForceBrowser bool
}

// NewLoginCmd creates a new command
func NewLoginCmd(flags *proflags.GlobalFlags) *cobra.Command {
	cmd := &LoginCmd{
		GlobalFlags: *flags,
	}
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Log into a DevPod Pro instance",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the DevPod Pro host, e.g. devpod pro login my-pro.my-domain.com")
			}

			return cmd.Run(context.Background(), args[0], log.Default)
		},
	}

	loginCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "If defined will use the given access key to login")
	loginCmd.Flags().BoolVar(&cmd.Login, "login", true, "If enabled will automatically try to log into the Loft DevPod Pro")
	loginCmd.Flags().BoolVar(&cmd.Use, "use", true, "If enabled will automatically activate the provider")
	loginCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Optional name how the DevPod Pro provider will be named")
	loginCmd.Flags().StringVar(&cmd.Version, "version", "", "The version to use for the DevPod provider")
	loginCmd.Flags().StringArrayVarP(&cmd.Options, "option", "o", []string{}, "Provider option in the form KEY=VALUE")
	loginCmd.Flags().BoolVar(&cmd.ForceBrowser, "force-browser", false, "Force login through browser")

	loginCmd.Flags().StringVar(&cmd.ProviderSource, "provider-source", "", "The source of the provider")
	_ = loginCmd.Flags().MarkHidden("provider-source")
	return loginCmd
}

// Run runs the command logic
func (cmd *LoginCmd) Run(ctx context.Context, fullURL string, log log.Logger) error {
	if strings.HasPrefix(fullURL, "http://") {
		return fmt.Errorf("http is not supported for DevPod Pro, please use https:// instead")
	} else if !strings.HasPrefix(fullURL, "https://") {
		fullURL = "https://" + fullURL
	} else if cmd.Provider != "" && len(cmd.Provider) > 32 {
		return fmt.Errorf("cannot use a provider name greater than 32 characters")
	}

	// get host from url
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return fmt.Errorf("invalid url %s: %w", fullURL, err)
	}

	// extract host
	host := parsedURL.Host

	// load devpod config
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	// check if there is already a pro instance with that url
	proInstances, err := workspace.ListProInstances(devPodConfig, log)
	if err != nil {
		return err
	}

	// check if url is found somewhere
	var currentInstance *provider.ProInstance
	for _, proInstance := range proInstances {
		if proInstance.Host == host {
			currentInstance = proInstance
			break
		}
	}
	if currentInstance != nil {
		cmd.Provider = currentInstance.Provider
	} else {
		// find a provider name
		if cmd.Provider == "" {
			cmd.Provider = "devpod-pro"
		}
		cmd.Provider = provider.ToProInstanceID(cmd.Provider)

		// check if provider already exists
		providers, err := workspace.LoadAllProviders(devPodConfig, log)
		if err != nil {
			return fmt.Errorf("load providers: %w", err)
		}

		// provider already exists?
		if providers[cmd.Provider] != nil {
			// alternative name
			cmd.Provider = provider.ToProInstanceID("devpod-" + host)
			if providers[cmd.Provider] != nil {
				return fmt.Errorf("provider %s already exists, please choose a different name via --provider", cmd.Provider)
			}
		}
	}

	// 1. Add provider
	if currentInstance == nil {
		currentInstance = &provider.ProInstance{
			Provider:          cmd.Provider,
			Host:              host,
			CreationTimestamp: types.Now(),
		}

		remoteVersion, err := platform.GetDevPodVersion(fullURL)
		if err != nil {
			return err
		}
		rv, err := semver.Parse(strings.TrimPrefix(remoteVersion, "v"))
		if err != nil {
			return fmt.Errorf("invalid version %s: %w", remoteVersion, err)
		}
		if rv.LT(semver.Version{Major: 0, Minor: 6, Patch: 999}) && remoteVersion != versionpkg.DevVersion {
			log.Debug("remote version < 0.7.0, installing proxy provider")
			// proxy providers are deprecated and shouldn't be used
			// unless explicitly the server version is below 0.7.0
			err = cmd.addLoftProvider(devPodConfig, fullURL, log)
			if err != nil {
				return err
			}
		} else {
			// add built-in pro (daemon) provider
			_, err = workspace.AddProvider(devPodConfig, cmd.Provider, "pro", log)
			if err != nil {
				return err
			}
		}

		err = provider.SaveProInstanceConfig(devPodConfig.DefaultContext, currentInstance)
		if err != nil {
			return err
		}

		// reload devpod config
		devPodConfig, err = config.LoadConfig(devPodConfig.DefaultContext, cmd.Provider)
		if err != nil {
			return err
		}
	}

	// get provider config
	providerConfig, err := provider.LoadProviderConfig(devPodConfig.DefaultContext, cmd.Provider)
	if err != nil {
		return err
	}

	// 2. Login to Loft
	if cmd.Login {
		err = login(ctx, devPodConfig, fullURL, cmd.Provider, cmd.AccessKey, false, cmd.ForceBrowser, log)
		if err != nil {
			return err
		}
		log.Donef("Successfully logged into DevPod Pro instance %s", ansi.Color(fullURL, "white+b"))
	}

	// 3. Configure provider
	if cmd.Use {
		err := providercmd.ConfigureProvider(ctx, providerConfig, devPodConfig.DefaultContext, cmd.Options, false, false, false, nil, log)
		if err != nil {
			return errors.Wrap(err, "configure provider")
		}
	}

	log.Donef("Successfully configured DevPod Pro")
	return nil
}

func (cmd *LoginCmd) addLoftProvider(devPodConfig *config.Config, url string, log log.Logger) error {
	// find out loft version
	err := cmd.resolveProviderSource(url)
	if err != nil {
		return err
	}

	// add the provider
	log.Infof("Add DevPod Pro provider...")

	// is development?
	if cmd.ProviderSource == providerRepo+"@v0.0.0" {
		log.Debugf("Add development provider")
		_, err = workspace.AddProviderRaw(devPodConfig, cmd.Provider, &provider.ProviderSource{}, []byte(fallbackProvider), log)
		if err != nil {
			return err
		}
	} else {
		_, err = workspace.AddProvider(devPodConfig, cmd.Provider, cmd.ProviderSource, log)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cmd *LoginCmd) resolveProviderSource(url string) error {
	if cmd.ProviderSource != "" {
		return nil
	}
	if cmd.Version != "" {
		cmd.ProviderSource = providerRepo + "@" + cmd.Version
		return nil
	}

	version, err := platform.GetDevPodVersion(url)
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	cmd.ProviderSource = providerRepo + "@" + version

	return nil
}

func login(ctx context.Context, devPodConfig *config.Config, url string, providerName string, accessKey string, skipBrowserLogin, forceBrowser bool, log log.Logger) error {
	configPath, err := platform.LoftConfigPath(devPodConfig.DefaultContext, providerName)
	if err != nil {
		return err
	}
	loader, err := client.NewClientFromPath(configPath)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	if accessKey == "" {
		accessKey = loader.Config().AccessKey
	}

	// log in
	url = strings.TrimSuffix(url, "/")
	if accessKey != "" && !forceBrowser {
		err = loader.LoginWithAccessKey(url, accessKey, true, true)
	} else {
		if skipBrowserLogin {
			return fmt.Errorf("unable to login to loft host")
		}
		err = loader.Login(url, true, log)
	}
	if err != nil {
		return err
	}

	return nil
}

var fallbackProvider = `name: devpod-pro
version: v0.0.0
icon: https://devpod.sh/assets/devpod.svg
description: DevPod Pro
options:
  LOFT_CONFIG:
    global: true
    hidden: true
    required: true
    default: "${PROVIDER_FOLDER}/loft-config.json"
binaries:
  PRO_PROVIDER:
    - os: linux
      arch: amd64
      path: /usr/local/bin/devpod
    - os: linux
      arch: arm64
      path: /usr/local/bin/devpod
    - os: darwin 
      arch: amd64
      path: /usr/local/bin/devpod
    - os: darwin
      arch: arm64
      path: /usr/local/bin/devpod
    - os: windows
      arch: amd64
      path: "C:\\Users\\pasca\\workspace\\devpod\\desktop\\src-tauri\\bin\\devpod-cli-x86_64-pc-windows-msvc.exe"
exec:
  proxy:
    up: |-
      ${PRO_PROVIDER} pro provider up
    ssh: |-
      ${PRO_PROVIDER} pro provider ssh
    stop: |-
      ${PRO_PROVIDER} pro provider stop
    status: |-
      ${PRO_PROVIDER} pro provider status
    delete: |-
      ${PRO_PROVIDER} pro provider delete
    health: |-
      ${PRO_PROVIDER} pro provider health
    daemon:
      start: |-
        ${PRO_PROVIDER} pro provider daemon start
      status: |-
        ${PRO_PROVIDER} pro provider daemon status
    create:
      workspace: |-
        ${PRO_PROVIDER} pro provider create workspace
    get:
      workspace: |-
        ${PRO_PROVIDER} pro provider get workspace
      self: |-
        ${PRO_PROVIDER} pro provider get self
      version: |-
        ${PRO_PROVIDER} pro provider get version
    update:
      workspace: |-
        ${PRO_PROVIDER} pro provider update workspace
    watch:
      workspaces: |-
        ${PRO_PROVIDER} pro provider watch workspaces
    list:
      workspaces: |-
        ${PRO_PROVIDER} pro provider list workspaces
      projects: |-
        ${PRO_PROVIDER} pro provider list projects
      templates: |-
        ${PRO_PROVIDER} pro provider list templates
      clusters: |-
        ${PRO_PROVIDER} pro provider list clusters
`
