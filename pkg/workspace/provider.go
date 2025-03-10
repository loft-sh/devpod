package workspace

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/devpod/providers"

	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/download"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

var (
	ErrNoWorkspaceFound    = errors.New("no workspace found")
	errProvideWorkspaceArg = errors.New("please provide a workspace name. E.g. 'devpod up ./my-folder', 'devpod up github.com/my-org/my-repo' or 'devpod up ubuntu'")
)

type ProviderWithOptions struct {
	Config *providerpkg.ProviderConfig `json:"config,omitempty"`
	State  *config.ProviderConfig      `json:"state,omitempty"`
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
		return nil, nil, fmt.Errorf("no default provider found. Please make sure to run 'devpod provider use'")
	} else if retProviders[defaultContext.DefaultProvider] == nil {
		return nil, nil, fmt.Errorf("couldn't find default provider %s. Please make sure to add the provider via 'devpod provider add'", defaultContext.DefaultProvider)
	}

	return retProviders[defaultContext.DefaultProvider], retProviders, nil
}

func CloneProvider(devPodConfig *config.Config, providerName, providerSourceRaw string, log log.Logger) (*ProviderWithOptions, error) {
	providerWithOptions, err := FindProvider(devPodConfig, providerSourceRaw, log)
	if err != nil {
		return nil, err
	}
	providerConfig, err := installProvider(devPodConfig, providerWithOptions.Config, providerName, &providerWithOptions.Config.Source, log)
	if err != nil {
		return nil, err
	}
	providerWithOptions.Config = providerConfig

	return providerWithOptions, nil
}

func AddProviderRaw(devPodConfig *config.Config, providerName string, providerSource *providerpkg.ProviderSource, providerRaw []byte, log log.Logger) (*providerpkg.ProviderConfig, error) {
	providerConfig, err := installRawProvider(devPodConfig, providerName, providerRaw, providerSource, log)
	if err != nil {
		return nil, err
	}

	if devPodConfig.Current().Providers == nil {
		devPodConfig.Current().Providers = map[string]*config.ProviderConfig{}
	}
	if devPodConfig.Current().Providers[providerConfig.Name] == nil {
		devPodConfig.Current().Providers[providerConfig.Name] = &config.ProviderConfig{
			CreationTimestamp: types.Now(),
		}
	}
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return nil, errors.Wrap(err, "save config")
	}

	return providerConfig, nil
}

func AddProvider(devPodConfig *config.Config, providerName, providerSourceRaw string, log log.Logger) (*providerpkg.ProviderConfig, error) {
	providerRaw, providerSource, err := ResolveProvider(providerSourceRaw, log)
	if err != nil {
		return nil, err
	}

	return AddProviderRaw(devPodConfig, providerName, providerSource, providerRaw, log)
}

func UpdateProvider(devPodConfig *config.Config, providerName, providerSourceRaw string, log log.Logger) (*providerpkg.ProviderConfig, error) {
	if devPodConfig.Current().Providers[providerName] == nil {
		return nil, fmt.Errorf("provider %s doesn't exist. Please run 'devpod provider add %s' instead", providerName, providerSourceRaw)
	}

	if providerSourceRaw == "" {
		s, err := ResolveProviderSource(devPodConfig, providerName, log)
		if err != nil {
			return nil, err
		}
		providerSourceRaw = s
	}

	providerRaw, providerSource, err := ResolveProvider(providerSourceRaw, log)
	if err != nil {
		return nil, err
	}

	return updateProvider(devPodConfig, providerName, providerRaw, providerSource, log)
}

func ResolveProviderSource(devPodConfig *config.Config, providerName string, log log.Logger) (string, error) {
	source := ""

	providerConfig, err := FindProvider(devPodConfig, providerName, log)
	if err != nil {
		return "", errors.Wrap(err, "find provider")
	}

	if providerConfig.Config.Source.Internal {
		// Name could also be overridden if initial name was already taken, so prefer the raw source if available
		if providerConfig.Config.Source.Raw == "" {
			source = providerConfig.Config.Name
		} else {
			source = providerConfig.Config.Source.Raw
		}
	} else if providerConfig.Config.Source.URL != "" {
		source = providerConfig.Config.Source.URL
	} else if providerConfig.Config.Source.File != "" {
		source = providerConfig.Config.Source.File
	} else if providerConfig.Config.Source.Github != "" {
		source = providerConfig.Config.Source.Github
	} else {
		return "", fmt.Errorf("provider %s is missing a source. Please run `devpod provider update %s SOURCE`", providerName, providerName)
	}

	return source, nil
}

func ResolveProvider(providerSource string, log log.Logger) ([]byte, *providerpkg.ProviderSource, error) {
	retSource := &providerpkg.ProviderSource{Raw: strings.TrimSpace(providerSource)}

	// in-built?
	internalProviders := providers.GetBuiltInProviders()
	if internalProviders[providerSource] != "" {
		retSource.Internal = true
		return []byte(internalProviders[providerSource]), retSource, nil
	}

	// url?
	if strings.HasPrefix(providerSource, "http://") || strings.HasPrefix(providerSource, "https://") {
		log.Infof("Download provider %s...", providerSource)
		out, err := downloadProvider(providerSource)
		if err != nil {
			return nil, nil, err
		}
		retSource.URL = providerSource

		return out, retSource, nil
	}

	// local file?
	if strings.HasSuffix(providerSource, ".yaml") || strings.HasSuffix(providerSource, ".yml") {
		_, err := os.Stat(providerSource)
		if err == nil {
			out, err := os.ReadFile(providerSource)
			if err == nil {
				absPath, err := filepath.Abs(providerSource)
				if err != nil {
					return nil, nil, err
				}
				retSource.File = absPath

				return out, retSource, nil
			}
		}
	}

	// check if github
	out, source, err := DownloadProviderGithub(providerSource, log)
	if err != nil {
		return nil, nil, errors.Wrap(err, "download github")
	} else if len(out) > 0 {
		return out, source, nil
	}

	return nil, nil, fmt.Errorf("unrecognized provider type, please specify either a local file, url or github repository")
}

func DownloadProviderGithub(originalPath string, log log.Logger) ([]byte, *providerpkg.ProviderSource, error) {
	path := strings.TrimPrefix(originalPath, "github.com/")

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
		return nil, nil, nil
	}

	// get latest release
	requestURL := ""
	if release == "" {
		requestURL = fmt.Sprintf("https://github.com/%s/releases/latest/download/provider.yaml", path)
	} else {
		requestURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/provider.yaml", path, release)
	}

	// download
	body, err := download.File(requestURL, log)
	if err != nil {
		return nil, nil, errors.Wrap(err, "download")
	}
	defer body.Close()

	// read body
	out, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}

	return out, &providerpkg.ProviderSource{
		Raw:    originalPath,
		Github: path,
	}, nil
}

func downloadProvider(url string) ([]byte, error) {
	// initiate download
	resp, err := devpodhttp.GetHTTPClient().Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "download binary")
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func updateProvider(devPodConfig *config.Config, providerName string, raw []byte, source *providerpkg.ProviderSource, log log.Logger) (*providerpkg.ProviderConfig, error) {
	providerConfig, err := providerpkg.ParseProvider(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	providerConfig.Source = *source
	if providerName != "" {
		providerConfig.Name = providerName
	}
	if providerConfig.Options == nil {
		providerConfig.Options = map[string]*types.Option{}
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

	binariesDir, err := providerpkg.GetProviderBinariesDir(devPodConfig.DefaultContext, providerConfig.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get binaries dir")
	}

	_, err = binaries.DownloadBinaries(providerConfig.Binaries, binariesDir, log)
	if err != nil {
		_ = os.RemoveAll(binariesDir)
		return nil, errors.Wrap(err, "download binaries")
	}

	err = providerpkg.SaveProviderConfig(devPodConfig.DefaultContext, providerConfig)
	if err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func installRawProvider(devPodConfig *config.Config, providerName string, raw []byte, source *providerpkg.ProviderSource, log log.Logger) (*providerpkg.ProviderConfig, error) {
	providerConfig, err := providerpkg.ParseProvider(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	return installProvider(devPodConfig, providerConfig, providerName, source, log)
}

func installProvider(devPodConfig *config.Config, providerConfig *providerpkg.ProviderConfig, providerName string, source *providerpkg.ProviderSource, log log.Logger) (*providerpkg.ProviderConfig, error) {
	providerConfig.Source = *source
	if providerName != "" {
		providerConfig.Name = providerName
	}
	if devPodConfig.Current().Providers[providerConfig.Name] != nil {
		return nil, fmt.Errorf("provider %s already exists. Please run 'devpod provider delete %s' before adding the provider", providerConfig.Name, providerConfig.Name)
	}

	providerDir, err := providerpkg.GetProviderDir(devPodConfig.DefaultContext, providerConfig.Name)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(providerDir)
	if err == nil {
		return nil, fmt.Errorf("provider %s already exists. Please run 'devpod provider delete %s' before adding the provider", providerConfig.Name, providerConfig.Name)
	}

	binariesDir, err := providerpkg.GetProviderBinariesDir(devPodConfig.DefaultContext, providerConfig.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get binaries dir")
	}

	_, err = binaries.DownloadBinaries(providerConfig.Binaries, binariesDir, log)
	if err != nil {
		_ = os.RemoveAll(providerDir)
		return nil, errors.Wrap(err, "download binaries")
	}

	err = providerpkg.SaveProviderConfig(devPodConfig.DefaultContext, providerConfig)
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
		return nil, fmt.Errorf("couldn't find provider with name %s. Please make sure to add the provider via 'devpod provider add'", name)
	}

	return retProviders[name], nil
}

func LoadAllProviders(devPodConfig *config.Config, log log.Logger) (map[string]*ProviderWithOptions, error) {
	retProviders := map[string]*ProviderWithOptions{}
	defaultContext := devPodConfig.Current()
	for providerName, providerState := range defaultContext.Providers {
		if retProviders[providerName] != nil {
			retProviders[providerName].State = providerState
			continue
		}

		// try to load provider config
		providerConfig, err := providerpkg.LoadProviderConfig(devPodConfig.DefaultContext, providerName)
		if err != nil {
			log.Warnf("Error loading provider '%s': %v", providerName, err)
			continue
		}

		retProviders[providerName] = &ProviderWithOptions{
			Config: providerConfig,
			State:  providerState,
		}
	}

	// list providers from the dir that are currently not configured
	providerDir, err := providerpkg.GetProvidersDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(providerDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	for _, entry := range entries {
		if retProviders[entry.Name()] != nil || !entry.IsDir() || strings.HasPrefix(entry.Name(), ".DS_Store") {
			continue
		}

		providerConfig, err := providerpkg.LoadProviderConfig(devPodConfig.DefaultContext, entry.Name())
		if err != nil {
			return nil, err
		}

		retProviders[providerConfig.Name] = &ProviderWithOptions{
			Config: providerConfig,
		}
	}

	return retProviders, nil
}

func ProviderFromHost(ctx context.Context, devPodConfig *config.Config, proHost string, log log.Logger) (*providerpkg.ProviderConfig, error) {
	proInstanceConfig, err := providerpkg.LoadProInstanceConfig(devPodConfig.DefaultContext, proHost)
	if err != nil {
		return nil, fmt.Errorf("load pro instance %s: %w", proHost, err)
	}

	provider, err := FindProvider(devPodConfig, proInstanceConfig.Provider, log)
	if err != nil {
		return nil, fmt.Errorf("find provider: %w", err)
	} else if !provider.Config.IsProxyProvider() && !provider.Config.IsDaemonProvider() {
		return nil, fmt.Errorf("provider is not a pro provider")
	}

	return provider.Config, nil
}
