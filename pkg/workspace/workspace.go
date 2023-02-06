package workspace

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/providerimplementation"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
	"github.com/loft-sh/devpod/providers"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var provideWorkspaceArgErr = fmt.Errorf("please provide a workspace name. E.g. 'devpod up ./my-folder', 'devpod up github.com/my-org/my-repo' or 'devpod up ubuntu'")

type ProviderWithOptions struct {
	Provider provider2.Provider
	Options  map[string]string
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
	builtInProviders, err := providers.GetBuiltInProviders()
	if err != nil {
		return nil, err
	}

	retProviders := map[string]*ProviderWithOptions{}
	for k, p := range builtInProviders {
		retProviders[k] = &ProviderWithOptions{
			Provider: p,
		}
	}

	defaultContext := devPodConfig.Contexts[devPodConfig.DefaultContext]
	for providerName, providerOptions := range defaultContext.Providers {
		if retProviders[providerName] != nil {
			retProviders[providerName].Options = providerOptions.Options
			continue
		}

		// try to load provider config
		providerDir, err := config.GetProviderDir(devPodConfig.DefaultContext, providerName)
		if err != nil {
			log.Errorf("Error retrieving provider directory: %v", err)
			continue
		}

		providerConfigFile := filepath.Join(providerDir, config.ProviderConfigFile)
		contents, err := os.ReadFile(providerConfigFile)
		if err != nil {
			log.Errorf("Error reading provider %s config: %v", providerName, err)
			continue
		}

		providerConfig, err := provider2.ParseProvider(bytes.NewReader(contents))
		if err != nil {
			log.Errorf("Error parsing provider %s config: %v", providerName, err)
			continue
		}

		retProviders[providerName] = &ProviderWithOptions{
			Provider: providerimplementation.NewProvider(providerConfig),
			Options:  providerOptions.Options,
		}
	}

	return retProviders, nil
}

func GetWorkspace(devPodConfig *config.Config, args []string, log log.Logger) (*provider2.Workspace, provider2.Provider, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(devPodConfig, log)
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0], log)

	// convert to id
	workspaceID := ToWorkspaceID(name)

	// already exists?
	if !config.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		return nil, nil, fmt.Errorf("workspace %s doesn't exist", workspaceID)
	}

	// load workspace config
	workspaceConfig, err := config.LoadWorkspaceConfig(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "load workspace config")
	}

	// find the matching provider
	providerWithOptions, err := FindProvider(devPodConfig, workspaceConfig.Provider.Name, log)
	if err != nil {
		return nil, nil, err
	}

	return workspaceConfig, providerWithOptions.Provider, nil
}

func ResolveWorkspace(devPodConfig *config.Config, args []string, log log.Logger) (*provider2.Workspace, provider2.Provider, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(devPodConfig, log)
	}

	// check if workspace already exists
	isLocalPath, name := isLocalDir(args[0], log)

	// convert to id
	workspaceID := ToWorkspaceID(name)

	// already exists?
	if config.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		log.Infof("Workspace %s already exists", workspaceID)
		workspaceConfig, err := config.LoadWorkspaceConfig(devPodConfig.DefaultContext, workspaceID)
		if err != nil {
			return nil, nil, errors.Wrap(err, "load workspace config")
		}

		// find the matching provider
		providerWithOptions, err := FindProvider(devPodConfig, workspaceConfig.Provider.Name, log)
		if err != nil {
			return nil, nil, err
		}

		return workspaceConfig, providerWithOptions.Provider, nil
	}

	// get default provider
	defaultProvider, _, err := LoadProviders(devPodConfig, log)

	// is local folder?
	if isLocalPath {
		return &provider2.Workspace{
			ID:      workspaceID,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name:    defaultProvider.Provider.Name(),
				Options: defaultProvider.Options,
			},
			Source: provider2.WorkspaceSource{
				LocalFolder: name,
			},
		}, defaultProvider.Provider, nil
	}

	// is git?
	gitRepository := normalizeGitRepository(name)
	if strings.HasSuffix(name, ".git") || pingRepository(gitRepository) {
		return &provider2.Workspace{
			ID:      workspaceID,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name:    defaultProvider.Provider.Name(),
				Options: defaultProvider.Options,
			},
			Source: provider2.WorkspaceSource{
				GitRepository: gitRepository,
			},
		}, defaultProvider.Provider, nil
	}

	// is image?
	_, err = image.GetImage(name)
	if err == nil {
		return &provider2.Workspace{
			ID:      workspaceID,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name:    defaultProvider.Provider.Name(),
				Options: defaultProvider.Options,
			},
			Source: provider2.WorkspaceSource{
				Image: name,
			},
		}, defaultProvider.Provider, nil
	}

	return nil, nil, fmt.Errorf("%s is neither a local folder, git repository or docker image", name)
}

func isLocalDir(name string, log log.Logger) (bool, string) {
	_, err := os.Stat(name)
	if err == nil {
		gitRoot := findGitRoot(name)
		if gitRoot != "" {
			log.Infof("Found git root at %s, switching working directory", gitRoot)
			return true, gitRoot
		}

		absPath, err := filepath.Abs(name)
		if err == nil {
			return true, absPath
		}
	}

	return false, name
}

func normalizeGitRepository(str string) string {
	if strings.HasSuffix(str, ".git") {
		str += ".git"
	}

	if !strings.HasPrefix(str, "git@") && !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		return "http://" + str
	}

	return str
}

func pingRepository(str string) bool {
	if !command.Exists("git") {
		return false
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := exec.CommandContext(timeoutCtx, "git", "ls-remote", "--quiet", str).CombinedOutput()
	if err != nil {
		return false
	}

	return true
}

var workspaceIDRegEx1 = regexp.MustCompile(`[^\w\-]`)
var workspaceIDRegEx2 = regexp.MustCompile(`[^0-9a-z\-]+`)

func ToWorkspaceID(str string) string {
	str = strings.ToLower(filepath.ToSlash(str))

	// get last element if we find a /
	index := strings.LastIndex(str, "/")
	if index == -1 {
		return workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(str, "-"), "")
	}

	return workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(str[index+1:], "-"), "")
}

func selectWorkspace(devPodConfig *config.Config, log log.Logger) (*provider2.Workspace, provider2.Provider, error) {
	if !terminal.IsTerminalIn {
		return nil, nil, provideWorkspaceArgErr
	}

	// ask which workspace to use
	workspacesDir, err := config.GetWorkspacesDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, nil, err
	}

	workspaceIDs := []string{}
	workspacesDirs, err := os.ReadDir(workspacesDir)
	for _, workspace := range workspacesDirs {
		workspaceIDs = append(workspaceIDs, workspace.Name())
	}
	if len(workspaceIDs) == 0 {
		return nil, nil, provideWorkspaceArgErr
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please select a workspace from the list below",
		DefaultValue: workspaceIDs[0],
		Options:      workspaceIDs,
		Sort:         true,
	})
	if err != nil {
		return nil, nil, err
	}

	workspaceConfig, err := config.LoadWorkspaceConfig(devPodConfig.DefaultContext, answer)
	if err != nil {
		return nil, nil, err
	}

	providerWithOptions, err := FindProvider(devPodConfig, workspaceConfig.Provider.Name, log)
	if err != nil {
		return nil, nil, err
	}

	return workspaceConfig, providerWithOptions.Provider, nil
}

func findGitRoot(localFolder string) string {
	if !command.Exists("git") {
		return ""
	}

	out, err := exec.Command("git", "-C", localFolder, "rev-parse", "--git-dir").Output()
	if err != nil {
		return ""
	}

	path := strings.TrimSpace(string(out))
	_, err = os.Stat(path)
	if err != nil {
		return ""
	}

	if filepath.IsAbs(path) {
		return path
	}

	absLocalFolder, err := filepath.Abs(localFolder)
	if err != nil {
		return ""
	}

	return filepath.Join(absLocalFolder, path)
}
