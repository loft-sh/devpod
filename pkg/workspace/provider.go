package workspace

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/download"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/providers"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
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
	defaultContext := devPodConfig.Current()
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
	providerRaw, err := resolveProvider(provider, log)
	if err != nil {
		return nil, err
	}

	return installProvider(devPodConfig, providerRaw, log)
}

func UpdateProvider(devPodConfig *config.Config, provider string, log log.Logger) (*provider2.ProviderConfig, error) {
	providerRaw, err := resolveProvider(provider, log)
	if err != nil {
		return nil, err
	}

	return updateProvider(devPodConfig, provider, providerRaw, log)
}

func resolveProvider(provider string, log log.Logger) ([]byte, error) {
	// local file?
	if strings.HasSuffix(provider, ".yaml") || strings.HasSuffix(provider, ".yml") {
		_, err := os.Stat(provider)
		if err == nil {
			out, err := os.ReadFile(provider)
			if err == nil {
				return out, nil
			}
		}
	}

	// url?
	if strings.HasPrefix(provider, "http://") || strings.HasPrefix(provider, "https://") {
		log.Infof("Download provider %s...", provider)

		out, err := downloadProvider(provider)
		if err != nil {
			return nil, err
		}

		return out, nil
	}

	// check if github
	out, err := DownloadProviderGithub(provider)
	if err != nil {
		return nil, errors.Wrap(err, "download github")
	} else if len(out) > 0 {
		return out, nil
	}

	return nil, fmt.Errorf("unrecognized provider type, please specify either a local file, url or github repository")
}

func DownloadProviderGithub(path string) ([]byte, error) {
	path = strings.TrimPrefix(path, "github.com/")

	// resolve release
	release := ""
	index := strings.LastIndex(path, "@")
	if index != -1 {
		release = path[index+1:]
		path = path[:index]
	}

	// split by separator
	splitted := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(splitted) == 1 {
		path = "loft-sh/devpod-provider-" + path
	} else if len(splitted) != 2 {
		return nil, nil
	}

	// get latest release
	requestURL := ""
	if release == "" {
		requestURL = fmt.Sprintf("https://github.com/%s/releases/latest/download/provider.yaml", path)
	} else {
		requestURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/provider.yaml", path, release)
	}

	// download
	body, err := download.File(requestURL)
	if err != nil {
		return nil, errors.Wrap(err, "download")
	}
	defer body.Close()

	// read body
	return io.ReadAll(body)
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

func updateProvider(devPodConfig *config.Config, provider string, raw []byte, log log.Logger) (*provider2.ProviderConfig, error) {
	providerConfig, err := provider2.ParseProvider(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	if devPodConfig.Current().Providers[providerConfig.Name] == nil {
		return nil, fmt.Errorf("provider %s doesn't exist. Please run 'devpod provider add %s' instead", providerConfig.Name, provider)
	}
	if providerConfig.Options == nil {
		providerConfig.Options = map[string]*provider2.ProviderOption{}
	}

	// update options
	for optionName := range devPodConfig.Current().Providers[providerConfig.Name].Options {
		_, ok := providerConfig.Options[optionName]
		if !ok {
			delete(devPodConfig.Current().Providers[providerConfig.Name].Options, optionName)
		}
	}

	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return nil, err
	}

	binariesDir, err := provider2.GetProviderBinariesDir(devPodConfig.DefaultContext, providerConfig.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get binaries dir")
	}

	_, err = binaries.DownloadBinaries(providerConfig.Binaries, binariesDir, log)
	if err != nil {
		_ = os.RemoveAll(binariesDir)
		return nil, errors.Wrap(err, "download binaries")
	}

	err = provider2.SaveProviderConfig(devPodConfig.DefaultContext, providerConfig)
	if err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func installProvider(devPodConfig *config.Config, raw []byte, log log.Logger) (*provider2.ProviderConfig, error) {
	providerConfig, err := provider2.ParseProvider(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	} else if devPodConfig.Current().Providers[providerConfig.Name] != nil {
		return nil, fmt.Errorf("provider %s already exists. Please run 'devpod provider delete %s' before adding the provider", providerConfig.Name, providerConfig.Name)
	}

	providerDir, err := provider2.GetProviderDir(devPodConfig.DefaultContext, providerConfig.Name)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(providerDir)
	if err == nil {
		return nil, fmt.Errorf("provider %s already exists. Please run 'devpod provider delete %s' before adding the provider", providerConfig.Name, providerConfig.Name)
	}

	binariesDir, err := provider2.GetProviderBinariesDir(devPodConfig.DefaultContext, providerConfig.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get binaries dir")
	}

	_, err = binaries.DownloadBinaries(providerConfig.Binaries, binariesDir, log)
	if err != nil {
		_ = os.RemoveAll(providerDir)
		return nil, errors.Wrap(err, "download binaries")
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

	defaultContext := devPodConfig.Current()
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
