package workspace

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/providers"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var provideWorkspaceArgErr = fmt.Errorf("please provide a workspace name. E.g. 'devpod up ./my-folder', 'devpod up github.com/my-org/my-repo' or 'devpod up ubuntu'")

type ProviderWithOptions struct {
	Configured bool
	Config     *provider2.ProviderConfig
	Options    map[string]config.OptionValue
}

// LoadProviders loads all known providers for the given context and
func LoadProviders(devPodConfig *config.Config, log log.Logger) (*ProviderWithOptions, map[string]*ProviderWithOptions, error) {
	defaultContext := devPodConfig.Contexts[devPodConfig.DefaultContext]
	retProviders, err := LoadAllProviders(devPodConfig, log)
	if err != nil {
		return nil, nil, err
	}

	// get default provider
	if defaultContext.DefaultProvider == "" {
		return nil, nil, fmt.Errorf("no default provider found. Please make sure to run 'devpod use provider'")
	} else if retProviders[defaultContext.DefaultProvider] == nil {
		return nil, nil, fmt.Errorf("couldn't find default provider %s. Please make sure to add the provider via 'devpod add provider'", defaultContext.DefaultProvider)
	}

	return retProviders[defaultContext.DefaultProvider], retProviders, nil
}

func AddProvider(devPodConfig *config.Config, provider string, log log.Logger) (*provider2.ProviderConfig, error) {
	// local file?
	_, err := os.Stat(provider)
	if err == nil {
		out, err := os.ReadFile(provider)
		if err != nil {
			return nil, err
		}

		return installProvider(devPodConfig, out)
	}

	// is git?
	gitRepository := normalizeGitRepository(provider)
	if strings.HasSuffix(provider, ".git") || pingRepository(gitRepository) {
		log.Infof("Clone git repository %s...", gitRepository)

		// shallow clone repository
		tempDir, err := os.CreateTemp("", "")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tempDir.Name())

		out, err := exec.Command("git", "clone", "--depth", "1", gitRepository, tempDir.Name()).CombinedOutput()
		if err != nil {
			return nil, errors.Wrapf(err, "git clone %s: %s", gitRepository, string(out))
		}

		filePath := filepath.Join(tempDir.Name(), "provider.yaml")
		_, err = os.Stat(filePath)
		if err != nil {
			filePath = filepath.Join(tempDir.Name(), "provider.yml")
			_, err = os.Stat(filePath)
			if err != nil {
				filePath = filepath.Join(tempDir.Name(), "provider.json")
				_, err = os.Stat(filePath)
				if err != nil {
					return nil, fmt.Errorf("couldn't find provider.yaml, provider.yml or provider.json in git repository")
				}
			}
		}

		providerBytes, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		return installProvider(devPodConfig, providerBytes)
	}

	// url?
	if strings.HasPrefix(provider, "http://") || strings.HasPrefix(provider, "https://") {
		log.Infof("Download provider %s...", provider)

		out, err := downloadProvider(provider)
		if err != nil {
			return nil, err
		}

		return installProvider(devPodConfig, out)
	}

	return nil, fmt.Errorf("unrecognized provider type, please specify either a local file, url or git repository")
}

func downloadProvider(url string) ([]byte, error) {
	// initiate download
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "download binary")
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func installProvider(devPodConfig *config.Config, raw []byte) (*provider2.ProviderConfig, error) {
	providerConfig, err := provider2.ParseProvider(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	} else if devPodConfig.Contexts[devPodConfig.DefaultContext].Providers[providerConfig.Name] != nil {
		return nil, fmt.Errorf("provider %s already exists. Please run 'devpod provider delete %s' before adding the provider", providerConfig.Name, providerConfig.Name)
	}

	err = provider2.SaveProviderConfig(devPodConfig.DefaultContext, providerConfig)
	if err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func FindProvider(devPodConfig *config.Config, name string, log log.Logger) (*ProviderWithOptions, error) {
	retProviders, err := LoadAllProviders(devPodConfig, log)
	if err != nil {
		return nil, err
	} else if retProviders[name] == nil {
		return nil, fmt.Errorf("couldn't find provider with name %s. Please make sure to add the provider via 'devpod add provider'", name)
	}

	return retProviders[name], nil
}

func LoadAllProviders(devPodConfig *config.Config, log log.Logger) (map[string]*ProviderWithOptions, error) {
	builtInProvidersConfig, err := providers.GetBuiltInProviders()
	if err != nil {
		return nil, err
	}

	retProviders := map[string]*ProviderWithOptions{}
	for k := range builtInProvidersConfig {
		retProviders[k] = &ProviderWithOptions{
			Config: builtInProvidersConfig[k],
		}
	}

	defaultContext := devPodConfig.Contexts[devPodConfig.DefaultContext]
	for providerName, providerOptions := range defaultContext.Providers {
		if retProviders[providerName] != nil {
			retProviders[providerName].Configured = true
			retProviders[providerName].Options = providerOptions.Options
			continue
		}

		// try to load provider config
		providerConfig, err := provider2.LoadProviderConfig(devPodConfig.DefaultContext, providerName)
		if err != nil {
			return nil, err
		}

		retProviders[providerName] = &ProviderWithOptions{
			Configured: true,
			Config:     providerConfig,
			Options:    providerOptions.Options,
		}
	}

	// list providers from the dir that are currently not configured
	providerDir, err := provider2.GetProvidersDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(providerDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	for _, entry := range entries {
		if retProviders[entry.Name()] != nil {
			continue
		}

		providerConfig, err := provider2.LoadProviderConfig(devPodConfig.DefaultContext, entry.Name())
		if err != nil {
			return nil, err
		}

		retProviders[providerConfig.Name] = &ProviderWithOptions{
			Config: providerConfig,
		}
	}

	return retProviders, nil
}
