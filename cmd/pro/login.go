package pro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Name           string
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
				return fmt.Errorf("please specify the DevPod Pro url, e.g. devpod pro login my-pro.my-domain.com")
			}

			return cmd.Run(context.Background(), args[0], log.Default)
		},
	}

	loginCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "If defined will use the given access key to login")
	loginCmd.Flags().BoolVar(&cmd.Login, "login", true, "If enabled will automatically try to log into the Loft DevPod Pro")
	loginCmd.Flags().BoolVar(&cmd.Use, "use", true, "If enabled will automatically activate the provider")
	loginCmd.Flags().StringVar(&cmd.Name, "name", "", "Optional name how this DevPod Pro will be referenced as")
	loginCmd.Flags().StringVar(&cmd.Version, "version", "", "The version to use for the DevPod provider")
	loginCmd.Flags().StringArrayVarP(&cmd.Options, "option", "o", []string{}, "Provider option in the form KEY=VALUE")

	loginCmd.Flags().StringVar(&cmd.ProviderSource, "provider-source", "", "The source of the provider")
	_ = loginCmd.Flags().MarkHidden("provider-source")
	return loginCmd
}

// Run runs the command logic
func (cmd *LoginCmd) Run(ctx context.Context, url string, log log.Logger) error {
	if strings.HasPrefix(url, "http://") {
		return fmt.Errorf("http is not supported for DevPod Pro, please use https:// instead")
	} else if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	} else if cmd.Name != "" && len(cmd.Name) > 32 {
		return fmt.Errorf("cannot use a name greater than 32 characters")
	}
	url = strings.TrimSuffix(url, "/")
	if cmd.Name == "" {
		cmd.Name = url
	}
	cmd.Name = workspace.ToProInstanceID(cmd.Name)

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
		if proInstance.URL == url || proInstance.ID == cmd.Name {
			if proInstance.URL != url {
				return fmt.Errorf("pro instance %s already exists with a different url %s != %s", cmd.Name, proInstance.URL, url)
			} else if proInstance.ID != cmd.Name {
				return fmt.Errorf("pro instance with url %s already exists with a different name %s != %s", url, proInstance.ID, cmd.Name)
			}

			currentInstance = proInstance
			break
		}
	}

	// 1. Add provider
	if currentInstance == nil {
		currentInstance = &provider.ProInstance{
			ID:                cmd.Name,
			URL:               url,
			CreationTimestamp: types.Now(),
		}

		err = cmd.addLoftProvider(devPodConfig, url, log)
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
	providerConfig, err := provider.LoadProviderConfig(devPodConfig.DefaultContext, cmd.Name)
	if err != nil {
		return err
	}

	// 2. Login to Loft
	if cmd.Login {
		err = cmd.login(ctx, devPodConfig, providerConfig, url, log)
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

	log.Donef("Successfully configured Loft DevPod Pro %s", cmd.Name)
	return nil
}

func (cmd *LoginCmd) login(ctx context.Context, devPodConfig *config.Config, providerConfig *provider.ProviderConfig, url string, log log.Logger) error {
	providerBinaries, err := binaries.GetBinaries(devPodConfig.DefaultContext, providerConfig)
	if err != nil {
		return fmt.Errorf("get provider binaries: %w", err)
	} else if providerBinaries[LOFT_PROVIDER_BINARY] == "" {
		return fmt.Errorf("provider is missing %s binary", LOFT_PROVIDER_BINARY)
	}

	providerDir, err := provider.GetProviderDir(devPodConfig.DefaultContext, cmd.Name)
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
	_, err = workspace.AddProvider(devPodConfig, cmd.Name, cmd.ProviderSource, log)
	if err != nil {
		return err
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
			} else if version.Version == "" || version.Version == "v0.0.0" {
				return fmt.Errorf("unexpected version '%s', please use --version to define a provider version", version.Version)
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
