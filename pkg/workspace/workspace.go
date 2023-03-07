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

var branchRegEx = regexp.MustCompile(`[^a-zA-Z0-9\.\-]+`)

// Exists checks if the given workspace already exists
func Exists(devPodConfig *config.Config, args []string, log log.Logger) string {
	if len(args) == 0 {
		return ""
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	// already exists?
	if !provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		return ""
	}

	return workspaceID
}

// GetWorkspace tries to retrieve an already existing workspace
func GetWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, args []string, log log.Logger) (client.WorkspaceClient, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(ctx, devPodConfig, ide, log)
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	// already exists?
	if !provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		return nil, fmt.Errorf("workspace %s doesn't exist", workspaceID)
	}

	// load workspace config
	return loadExistingWorkspace(workspaceID, devPodConfig, ide, log)
}

// ResolveWorkspace tries to retrieve an already existing workspace or creates a new one
func ResolveWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, args []string, desiredID, desiredMachine, providerOverride string, providerUserOptions []string, log log.Logger) (client.WorkspaceClient, error) {
	workspaceClient, err := resolveWorkspace(ctx, devPodConfig, ide, args, desiredID, desiredMachine, providerOverride, log)
	if err != nil {
		return nil, err
	}

	// refresh options
	err = workspaceClient.RefreshOptions(ctx, providerUserOptions)
	if err != nil {
		return nil, err
	}

	return workspaceClient, nil
}

func resolveWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, args []string, desiredID, desiredMachine, providerOverride string, log log.Logger) (client.WorkspaceClient, error) {
	// check if we have no args
	if len(args) == 0 {
		if desiredID != "" {
			return GetWorkspace(ctx, devPodConfig, ide, []string{desiredID}, log)
		}

		return selectWorkspace(ctx, devPodConfig, ide, log)
	}

	// check if workspace already exists
	isLocalPath, name := isLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	// check if desired id already exists
	if desiredID != "" {
		if provider2.WorkspaceExists(devPodConfig.DefaultContext, desiredID) {
			log.Infof("Workspace %s already exists", desiredID)
			return loadExistingWorkspace(desiredID, devPodConfig, ide, log)
		}

		// set desired id
		workspaceID = desiredID
	} else if provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		log.Infof("Workspace %s already exists", workspaceID)
		return loadExistingWorkspace(workspaceID, devPodConfig, ide, log)
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
	workspace, err := resolve(defaultProvider, devPodConfig, name, workspaceID, workspaceFolder, isLocalPath)
	if err != nil {
		_ = os.RemoveAll(workspaceFolder)
		return nil, err
	}

	// set ide config
	if ide != nil {
		workspace.IDE = *ide
	}

	// set server
	if desiredMachine != "" {
		if !defaultProvider.Config.IsMachineProvider() {
			_ = os.RemoveAll(workspaceFolder)
			return nil, fmt.Errorf("provider %s cannot create servers and cannot be used", defaultProvider.Config.Name)
		}

		// check if server exists
		if !provider2.MachineExists(workspace.Context, desiredMachine) {
			_ = os.RemoveAll(workspaceFolder)
			return nil, fmt.Errorf("server %s doesn't exist and cannot be used", desiredMachine)
		}

		// configure server for workspace
		workspace.Machine = provider2.WorkspaceMachineConfig{
			ID: desiredMachine,
		}
	}

	// save workspace config
	err = saveWorkspaceConfig(workspace)
	if err != nil {
		_ = os.RemoveAll(workspaceFolder)
		return nil, errors.Wrap(err, "save config")
	}

	// create a new client
	workspaceClient, err := clientimplementation.NewWorkspaceClient(devPodConfig, defaultProvider.Config, workspace, log)
	if err != nil {
		_ = os.RemoveAll(workspaceFolder)
		return nil, errors.Wrap(err, "create workspace client")
	}

	return workspaceClient, nil
}

func resolve(defaultProvider *ProviderWithOptions, devPodConfig *config.Config, name, workspaceID, workspaceFolder string, isLocalPath bool) (*provider2.Workspace, error) {
	// create workspace ssh keys
	_, err := devssh.GetPublicKey(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "create ssh keys")
	}

	// is local folder?
	if isLocalPath {
		return &provider2.Workspace{
			ID:      workspaceID,
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name: defaultProvider.Config.Name,
			},
			Source: provider2.WorkspaceSource{
				LocalFolder: name,
			},
		}, nil
	}

	// is git?
	gitRepository, gitBranch := normalizeGitRepository(name)
	if strings.HasSuffix(name, ".git") || pingRepository(gitRepository) {
		return &provider2.Workspace{
			ID:      workspaceID,
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name: defaultProvider.Config.Name,
			},
			Source: provider2.WorkspaceSource{
				GitRepository: gitRepository,
				GitBranch:     gitBranch,
			},
		}, nil
	}

	// is image?
	_, err = image.GetImage(name)
	if err == nil {
		return &provider2.Workspace{
			ID:      workspaceID,
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name: defaultProvider.Config.Name,
			},
			Source: provider2.WorkspaceSource{
				Image: name,
			},
		}, nil
	}

	return nil, fmt.Errorf("%s is neither a local folder, git repository or docker image", name)
}

func isLocalDir(name string) (bool, string) {
	_, err := os.Stat(name)
	if err == nil {
		absPath, _ := filepath.Abs(name)
		if absPath != "" {
			return true, absPath
		}
	}

	return false, name
}

func normalizeGitRepository(str string) (string, string) {
	if !strings.HasPrefix(str, "git@") && !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		str = "http://" + str
	}

	// resolve branch
	branch := ""
	index := strings.LastIndex(str, "@")
	if index != -1 {
		branch = str[index+1:]
		repo := str[:index]

		// is not a valid tag / branch name?
		if branchRegEx.MatchString(branch) {
			branch = ""
		} else {
			str = repo
		}
	}

	if !strings.HasSuffix(str, ".git") {
		str += ".git"
	}

	return str, branch
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

func ToID(str string) string {
	str = strings.ToLower(filepath.ToSlash(str))

	// get last element if we find a /
	index := strings.LastIndex(str, "/")
	if index != -1 {
		str = str[index+1:]

		// remove .git if there is it
		str = strings.TrimSuffix(str, ".git")

		// remove a potential tag / branch name
		splitted := strings.Split(str, "@")
		if len(splitted) == 2 && !branchRegEx.MatchString(splitted[1]) {
			str = splitted[0]
		}
	}

	return workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(str, "-"), "")
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
	return loadExistingWorkspace(answer, devPodConfig, ide, log)
}

func loadExistingWorkspace(workspaceID string, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, log log.Logger) (client.WorkspaceClient, error) {
	workspaceConfig, err := provider2.LoadWorkspaceConfig(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, err
	}

	providerWithOptions, err := FindProvider(devPodConfig, workspaceConfig.Provider.Name, log)
	if err != nil {
		return nil, err
	}

	// replace ide config
	beforeIDE := workspaceConfig.IDE
	if ide != nil {
		workspaceConfig.IDE = *ide
	}

	// save workspace config
	if !reflect.DeepEqual(workspaceConfig.IDE, beforeIDE) {
		err = provider2.SaveWorkspaceConfig(workspaceConfig)
		if err != nil {
			return nil, err
		}
	}

	return clientimplementation.NewWorkspaceClient(devPodConfig, providerWithOptions.Config, workspaceConfig, log)
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
