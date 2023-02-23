package workspace

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	options2 "github.com/loft-sh/devpod/pkg/provider/options"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// GetWorkspace tries to retrieve an already existing workspace
func GetWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, args []string, log log.Logger) (client.WorkspaceClient, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(ctx, devPodConfig, ide, log)
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0], log)

	// convert to id
	workspaceID := ToWorkspaceID(name)

	// already exists?
	if !provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		return nil, fmt.Errorf("workspace %s doesn't exist", workspaceID)
	}

	// load workspace config
	return loadExistingWorkspace(ctx, workspaceID, devPodConfig, ide, log)
}

// ResolveWorkspace tries to retrieve an already existing workspace or creates a new one
func ResolveWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, args []string, desiredID, providerOverride string, log log.Logger) (client.WorkspaceClient, error) {
	// check if we have no args
	if len(args) == 0 {
		if desiredID != "" {
			return GetWorkspace(ctx, devPodConfig, ide, []string{desiredID}, log)
		}

		return selectWorkspace(ctx, devPodConfig, ide, log)
	}

	// check if workspace already exists
	isLocalPath, name := isLocalDir(args[0], log)

	// convert to id
	workspaceID := ToWorkspaceID(name)

	// check if desired id already exists
	if desiredID != "" {
		if provider2.WorkspaceExists(devPodConfig.DefaultContext, desiredID) {
			log.Infof("Workspace %s already exists", desiredID)
			return loadExistingWorkspace(ctx, desiredID, devPodConfig, ide, log)
		}

		// set desired id
		workspaceID = desiredID
	} else if provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		log.Infof("Workspace %s already exists", workspaceID)
		return loadExistingWorkspace(ctx, workspaceID, devPodConfig, ide, log)
	}

	// get default provider
	defaultProvider, allProviders, err := LoadProviders(devPodConfig, log)
	if err != nil {
		return nil, err
	}
	if providerOverride != "" {
		var ok bool
		defaultProvider, ok = allProviders[providerOverride]
		if !ok {
			return nil, fmt.Errorf("couldn't find provider %s", providerOverride)
		}
	}

	// get workspace folder
	workspaceFolder, err := provider2.GetWorkspaceDir(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, err
	}

	// resolve workspace
	workspace, err := resolve(ctx, defaultProvider, devPodConfig, name, workspaceID, workspaceFolder, isLocalPath)
	if err != nil {
		_ = os.RemoveAll(workspaceFolder)
		return nil, err
	}

	// set ide config
	if ide != nil {
		workspace.IDE = *ide
	}

	// save workspace config
	err = saveWorkspaceConfig(workspace)
	if err != nil {
		_ = os.RemoveAll(workspaceFolder)
		return nil, errors.Wrap(err, "save config")
	}

	// create a new client
	return clientimplementation.NewWorkspaceClient(defaultProvider.Config, workspace, log)
}

func resolve(ctx context.Context, defaultProvider *ProviderWithOptions, devPodConfig *config.Config, name, workspaceID, workspaceFolder string, isLocalPath bool) (*provider2.Workspace, error) {
	// resolve options
	workspace, err := options2.ResolveOptions(ctx, "", "", &provider2.Workspace{
		Provider: provider2.WorkspaceProviderConfig{
			Name:    defaultProvider.Config.Name,
			Options: defaultProvider.Options,
		},
	}, defaultProvider.Config)
	if err != nil {
		return nil, errors.Wrap(err, "resolve options")
	}

	// create workspace ssh keys
	_, err = devssh.GetPublicKey(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "create ssh keys")
	}

	// is local folder?
	if isLocalPath {
		return &provider2.Workspace{
			ID:       workspaceID,
			Folder:   workspaceFolder,
			Context:  devPodConfig.DefaultContext,
			Provider: workspace.Provider,
			Source: provider2.WorkspaceSource{
				LocalFolder: name,
			},
		}, nil
	}

	// is git?
	gitRepository := normalizeGitRepository(name)
	if strings.HasSuffix(name, ".git") || pingRepository(gitRepository) {
		return &provider2.Workspace{
			ID:       workspaceID,
			Folder:   workspaceFolder,
			Context:  devPodConfig.DefaultContext,
			Provider: workspace.Provider,
			Source: provider2.WorkspaceSource{
				GitRepository: gitRepository,
			},
		}, nil
	}

	// is image?
	_, err = image.GetImage(name)
	if err == nil {
		return &provider2.Workspace{
			ID:       workspaceID,
			Folder:   workspaceFolder,
			Context:  devPodConfig.DefaultContext,
			Provider: workspace.Provider,
			Source: provider2.WorkspaceSource{
				Image: name,
			},
		}, nil
	}

	return nil, fmt.Errorf("%s is neither a local folder, git repository or docker image", name)
}

func isLocalDir(name string, log log.Logger) (bool, string) {
	_, err := os.Stat(name)
	if err == nil {
		absPath, _ := filepath.Abs(name)
		gitRoot := findGitRoot(name)
		if gitRoot != "" && gitRoot != absPath {
			log.Infof("Found git root at %s", gitRoot)
			return true, gitRoot
		}

		if absPath != "" {
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

func selectWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, log log.Logger) (client.WorkspaceClient, error) {
	if !terminal.IsTerminalIn {
		return nil, provideWorkspaceArgErr
	}

	// ask which workspace to use
	workspacesDir, err := provider2.GetWorkspacesDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	workspaceIDs := []string{}
	workspacesDirs, err := os.ReadDir(workspacesDir)
	for _, workspace := range workspacesDirs {
		workspaceIDs = append(workspaceIDs, workspace.Name())
	}
	if len(workspaceIDs) == 0 {
		return nil, provideWorkspaceArgErr
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please select a workspace from the list below",
		DefaultValue: workspaceIDs[0],
		Options:      workspaceIDs,
		Sort:         true,
	})
	if err != nil {
		return nil, err
	}

	// load workspace
	return loadExistingWorkspace(ctx, answer, devPodConfig, ide, log)
}

func loadExistingWorkspace(ctx context.Context, workspaceID string, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, log log.Logger) (client.WorkspaceClient, error) {
	workspaceConfig, err := provider2.LoadWorkspaceConfig(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, err
	}

	providerWithOptions, err := FindProvider(devPodConfig, workspaceConfig.Provider.Name, log)
	if err != nil {
		return nil, err
	}

	// resolve options
	beforeOptions := workspaceConfig.Provider.Options
	workspaceConfig, err = options2.ResolveOptions(ctx, "", "", workspaceConfig, providerWithOptions.Config)
	if err != nil {
		return nil, errors.Wrap(err, "resolve options")
	}

	// replace ide config
	if ide != nil {
		workspaceConfig.IDE = *ide
	}

	// save workspace config
	if !reflect.DeepEqual(workspaceConfig.Provider.Options, beforeOptions) {
		err = provider2.SaveWorkspaceConfig(workspaceConfig)
		if err != nil {
			return nil, err
		}
	}

	return clientimplementation.NewWorkspaceClient(providerWithOptions.Config, workspaceConfig, log)
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
		return filepath.Dir(path)
	}

	absLocalFolder, err := filepath.Abs(localFolder)
	if err != nil {
		return ""
	}

	return filepath.Dir(filepath.Join(absLocalFolder, path))
}

func saveWorkspaceConfig(workspace *provider2.Workspace) error {
	// save config
	workspace.CreationTimestamp = types.Now()
	err := provider2.SaveWorkspaceConfig(workspace)
	if err != nil {
		return err
	}

	return nil
}
