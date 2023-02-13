package workspace

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	options2 "github.com/loft-sh/devpod/pkg/provider/options"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
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
func GetWorkspace(ctx context.Context, devPodConfig *config.Config, args []string, log log.Logger) (*provider2.Workspace, provider2.Provider, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(ctx, devPodConfig, log)
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
	return loadExistingWorkspace(ctx, workspaceID, devPodConfig, log)
}

// ResolveWorkspace tries to retrieve an already existing workspace or creates a new one
func ResolveWorkspace(ctx context.Context, devPodConfig *config.Config, args []string, desiredID string, log log.Logger) (*provider2.Workspace, provider2.Provider, error) {
	// check if we have no args
	if len(args) == 0 {
		if desiredID != "" {
			return GetWorkspace(ctx, devPodConfig, []string{desiredID}, log)
		}

		return selectWorkspace(ctx, devPodConfig, log)
	}

	// check if workspace already exists
	isLocalPath, name := isLocalDir(args[0], log)

	// convert to id
	workspaceID := ToWorkspaceID(name)

	// check if desired id already exists
	if desiredID != "" {
		if config.WorkspaceExists(devPodConfig.DefaultContext, desiredID) {
			log.Infof("Workspace %s already exists", desiredID)
			return loadExistingWorkspace(ctx, desiredID, devPodConfig, log)
		}

		// set desired id
		workspaceID = desiredID
	} else if config.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		log.Infof("Workspace %s already exists", workspaceID)
		return loadExistingWorkspace(ctx, workspaceID, devPodConfig, log)
	}

	// get default provider
	defaultProvider, _, err := LoadProviders(devPodConfig, log)
	if err != nil {
		return nil, nil, err
	}

	// get workspace folder
	workspaceFolder, err := config.GetWorkspaceDir(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, nil, err
	}

	// resolve options
	options, err := options2.ResolveOptions(ctx, "", "", &provider2.Workspace{
		Provider: provider2.WorkspaceProviderConfig{
			Options: defaultProvider.Options,
		},
	}, defaultProvider.Provider.Options())
	if err != nil {
		return nil, nil, errors.Wrap(err, "resolve options")
	}

	// create workspace ssh keys
	_, err = devssh.GetPublicKey(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "create ssh keys")
	}

	// is local folder?
	if isLocalPath {
		return &provider2.Workspace{
			ID:      workspaceID,
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name:    defaultProvider.Provider.Name(),
				Options: options,
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
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name:    defaultProvider.Provider.Name(),
				Options: options,
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
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name:    defaultProvider.Provider.Name(),
				Options: options,
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

func selectWorkspace(ctx context.Context, devPodConfig *config.Config, log log.Logger) (*provider2.Workspace, provider2.Provider, error) {
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

	// load workspace
	return loadExistingWorkspace(ctx, answer, devPodConfig, log)
}

func loadExistingWorkspace(ctx context.Context, workspaceID string, devPodConfig *config.Config, log log.Logger) (*provider2.Workspace, provider2.Provider, error) {
	workspaceConfig, err := config.LoadWorkspaceConfig(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, nil, err
	}

	providerWithOptions, err := FindProvider(devPodConfig, workspaceConfig.Provider.Name, log)
	if err != nil {
		return nil, nil, err
	}

	// resolve options
	beforeOptions := workspaceConfig.Provider.Options
	workspaceConfig.Provider.Options, err = options2.ResolveOptions(ctx, "", "", workspaceConfig, providerWithOptions.Provider.Options())
	if err != nil {
		return nil, nil, errors.Wrap(err, "resolve options")
	}

	// save workspace config
	if !reflect.DeepEqual(workspaceConfig.Provider.Options, beforeOptions) {
		err = config.SaveWorkspaceConfig(workspaceConfig)
		if err != nil {
			return nil, nil, err
		}
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
		return filepath.Dir(path)
	}

	absLocalFolder, err := filepath.Abs(localFolder)
	if err != nil {
		return ""
	}

	return filepath.Dir(filepath.Join(absLocalFolder, path))
}
