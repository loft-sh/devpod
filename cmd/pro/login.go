package pro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	providercmd "github.com/loft-sh/devpod/cmd/provider"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const LOFT_PROVIDER_BINARY = "LOFT_PROVIDER"

// LoginCmd holds the login cmd flags
type LoginCmd struct {
	flags.GlobalFlags

	AccessKey      string
	Provider       string
	Version        string
	ProviderSource string

	Options []string

	Login bool
	Use   bool
}

// NewLoginCmd creates a new command
func NewLoginCmd(flags *flags.GlobalFlags) *cobra.Command {
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

		err = cmd.addLoftProvider(devPodConfig, fullURL, log)
		if err != nil {
			return err
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
		err = cmd.login(ctx, devPodConfig, providerConfig, fullURL, log)
		if err != nil {
			return err
		}
	}

	// 3. Configure provider
	if cmd.Use {
		err := providercmd.ConfigureProvider(ctx, providerConfig, devPodConfig.DefaultContext, cmd.Options, false, false, nil, log)
		if err != nil {
			return errors.Wrap(err, "configure provider")
		}
	}

	log.Donef("Successfully configured Loft DevPod Pro")
	return nil
}

func (cmd *LoginCmd) login(ctx context.Context, devPodConfig *config.Config, providerConfig *provider.ProviderConfig, url string, log log.Logger) error {
	providerBinaries, err := binaries.GetBinaries(devPodConfig.DefaultContext, providerConfig)
	if err != nil {
		return fmt.Errorf("get provider binaries: %w", err)
	} else if providerBinaries[LOFT_PROVIDER_BINARY] == "" {
		return fmt.Errorf("provider is missing %s binary", LOFT_PROVIDER_BINARY)
	}

	providerDir, err := provider.GetProviderDir(devPodConfig.DefaultContext, cmd.Provider)
	if err != nil {
		return err
	}

	args := []string{
		"login",
		"--insecure",
		"--log-output=raw",
		url,
	}
	if cmd.AccessKey != "" {
		args = append(args, "--access-key", cmd.AccessKey)
	}

	extraEnv := []string{
		"LOFT_SKIP_VERSION_CHECK=true",
		"LOFT_CONFIG=" + filepath.Join(providerDir, "loft-config.json"),
	}

	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// start the command
	loginCmd := exec.CommandContext(ctx, providerBinaries[LOFT_PROVIDER_BINARY], args...)
	loginCmd.Env = os.Environ()
	loginCmd.Env = append(loginCmd.Env, extraEnv...)
	loginCmd.Stdout = writer
	loginCmd.Stderr = writer
	err = loginCmd.Run()
	if err != nil {
		return fmt.Errorf("run login command: %w", err)
	}

	log.Donef("Successfully logged into %s", url)
	return nil
}

func (cmd *LoginCmd) addLoftProvider(devPodConfig *config.Config, url string, log log.Logger) error {
	// find out loft version
	err := cmd.getProviderSource(url)
	if err != nil {
		return err
	}

	// add the provider
	log.Infof("Add Loft DevPod Pro provider...")

	// is development?
	if cmd.ProviderSource == "loft-sh/loft@v0.0.0" {
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

func (cmd *LoginCmd) getProviderSource(url string) error {
	if cmd.ProviderSource == "" {
		if cmd.Version == "" {
			resp, err := http.GetHTTPClient().Get(url + "/version")
			if err != nil {
				return fmt.Errorf("get %s: %w", url, err)
			} else if resp.StatusCode != 200 {
				out, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("get %s: %s (Status: %d)", url, string(out), resp.StatusCode)
			}

			versionRaw, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read %s: %w", url, err)
			}

			version := &versionObject{}
			err = json.Unmarshal(versionRaw, version)
			if err != nil {
				return fmt.Errorf("parse %s: %w", url, err)
			} else if version.Version == "" {
				return fmt.Errorf("unexpected version '%s', please use --version to define a provider version", version.Version)
			}

			// make sure it starts with a v
			if !strings.HasPrefix(version.Version, "v") {
				version.Version = "v" + version.Version
			}

			cmd.ProviderSource = "loft-sh/loft@" + version.Version
		} else {
			cmd.ProviderSource = "loft-sh/loft@" + cmd.Version
		}
	}

	return nil
}

type versionObject struct {
	// Version is the loft remote version
	Version string `json:"version,omitempty"`
}

var fallbackProvider = `name: loft
version: v0.0.0
icon: https://devpod.sh/assets/devpod.svg
description: DevPod on Loft DevPod Engine
optionGroups:
  - name: Main Options
    defaultVisible: true
    options:
      - LOFT_PROJECT
      - LOFT_TEMPLATE
      - LOFT_TEMPLATE_VERSION
  - name: Template Options
    defaultVisible: true
    options:
      - "TEMPLATE_OPTION_*"
  - name: Other Options
    options:
      - LOFT_RUNNER
options:
  LOFT_CONFIG:
    global: true
    hidden: true
    required: true
    default: "${PROVIDER_FOLDER}/loft-config.json"
    subOptionsCommand: "${LOFT_PROVIDER} devpod list projects"
exec:
  proxy:
    up: |-
      ${LOFT_PROVIDER} devpod up
    ssh: |-
      ${LOFT_PROVIDER} devpod ssh
    stop: |-
      ${LOFT_PROVIDER} devpod stop
    status: |-
      ${LOFT_PROVIDER} devpod status
    delete: |-
      ${LOFT_PROVIDER} devpod delete
binaries:
  LOFT_PROVIDER:
    - os: linux
      arch: amd64
      path: /usr/local/bin/loft
    - os: linux
      arch: arm64
      path: /usr/local/bin/loft
    - os: darwin
      arch: amd64
      path: /usr/local/bin/loft
    - os: darwin
      arch: arm64
      path: /usr/local/bin/loft
`
