package workspace

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
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

func SingleMachineName(provider string) string {
	return "devpod-machine-" + provider
}

// Exists checks if the given workspace already exists
func Exists(devPodConfig *config.Config, args []string) string {
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

func GetWorkspaceName(args []string) string {
	if len(args) == 0 {
		return ""
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	return workspaceID
}

// GetWorkspace tries to retrieve an already existing workspace
func GetWorkspace(devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, args []string, changeLastUsed bool, log log.Logger) (client.WorkspaceClient, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(devPodConfig, ide, changeLastUsed, log)
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
	return loadExistingWorkspace(workspaceID, devPodConfig, ide, changeLastUsed, log)
}

// ResolveWorkspace tries to retrieve an already existing workspace or creates a new one
func ResolveWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, args []string, desiredID, desiredMachine, providerOverride string, providerUserOptions []string, changeLastUsed bool, log log.Logger) (client.WorkspaceClient, error) {
	workspaceClient, err := resolveWorkspace(ctx, devPodConfig, ide, args, desiredID, desiredMachine, providerOverride, providerUserOptions, changeLastUsed, log)
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

func resolveWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, args []string, desiredID, desiredMachine, providerOverride string, providerUserOptions []string, changeLastUsed bool, log log.Logger) (client.WorkspaceClient, error) {
	// check if we have no args
	if len(args) == 0 {
		if desiredID != "" {
			return GetWorkspace(devPodConfig, ide, []string{desiredID}, changeLastUsed, log)
		}

		return selectWorkspace(devPodConfig, ide, changeLastUsed, log)
	}

	// check if workspace already exists
	isLocalPath, name := isLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	// check if desired id already exists
	if desiredID != "" {
		if provider2.WorkspaceExists(devPodConfig.DefaultContext, desiredID) {
			log.Infof("Workspace %s already exists", desiredID)
			return loadExistingWorkspace(desiredID, devPodConfig, ide, changeLastUsed, log)
		}

		// set desired id
		workspaceID = desiredID
	} else if provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		log.Infof("Workspace %s already exists", workspaceID)
		return loadExistingWorkspace(workspaceID, devPodConfig, ide, changeLastUsed, log)
	}

	// create workspace
	workspaceClient, err := createWorkspace(ctx, devPodConfig, ide, workspaceID, name, desiredMachine, providerOverride, providerUserOptions, isLocalPath, log)
	if err != nil {
		_ = clientimplementation.DeleteWorkspaceFolder(devPodConfig.DefaultContext, workspaceID)
		return nil, err
	}

	return workspaceClient, nil
}

func createWorkspace(ctx context.Context, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, workspaceID, name, desiredMachine, providerOverride string, providerUserOptions []string, isLocalPath bool, log log.Logger) (client.WorkspaceClient, error) {
	// get default provider
	provider, allProviders, err := LoadProviders(devPodConfig, log)
	if err != nil {
		return nil, err
	} else if providerOverride != "" {
		var ok bool
		provider, ok = allProviders[providerOverride]
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
	workspace, err := resolve(provider, devPodConfig, name, workspaceID, workspaceFolder, isLocalPath)
	if err != nil {
		return nil, err
	}

	// set ide config
	if ide != nil {
		workspace.IDE = *ide
	}

	// set server
	if desiredMachine != "" {
		if !provider.Config.IsMachineProvider() {
			return nil, fmt.Errorf("provider %s cannot create servers and cannot be used", provider.Config.Name)
		}

		// check if server exists
		if !provider2.MachineExists(workspace.Context, desiredMachine) {
			return nil, fmt.Errorf("server %s doesn't exist and cannot be used", desiredMachine)
		}

		// configure server for workspace
		workspace.Machine = provider2.WorkspaceMachineConfig{
			ID: desiredMachine,
		}
	}

	// create a new machine
	var machineConfig *provider2.Machine
	if provider.Config.IsMachineProvider() && workspace.Machine.ID == "" {
		// create a new machine
		if provider.State != nil && provider.State.SingleMachine {
			workspace.Machine.ID = SingleMachineName(provider.Config.Name)
		} else {
			workspace.Machine.ID = workspace.ID
			workspace.Machine.AutoDelete = true
		}

		// save workspace config
		err = saveWorkspaceConfig(workspace)
		if err != nil {
			return nil, errors.Wrap(err, "save config")
		}

		// only create machine if it does not exist yet
		if !provider2.MachineExists(devPodConfig.DefaultContext, workspace.Machine.ID) {
			// create machine folder
			machineConfig, err = createMachine(workspace.Context, workspace.Machine.ID, provider.Config.Name)
			if err != nil {
				return nil, err
			}

			// create machine
			machineClient, err := clientimplementation.NewMachineClient(devPodConfig, provider.Config, machineConfig, log)
			if err != nil {
				_ = clientimplementation.DeleteMachineFolder(machineConfig.Context, machineConfig.ID)
				return nil, err
			}

			// refresh options
			err = machineClient.RefreshOptions(ctx, providerUserOptions)
			if err != nil {
				_ = clientimplementation.DeleteMachineFolder(machineConfig.Context, machineConfig.ID)
				return nil, err
			}

			// create machine
			err = machineClient.Create(ctx, client.CreateOptions{})
			if err != nil {
				_ = clientimplementation.DeleteMachineFolder(machineConfig.Context, machineConfig.ID)
				return nil, err
			}
		} else {
			// load machine config
			machineConfig, err = provider2.LoadMachineConfig(workspace.Context, workspace.Machine.ID)
			if err != nil {
				return nil, errors.Wrap(err, "load machine config")
			}
		}
	} else {
		// save workspace config
		err = saveWorkspaceConfig(workspace)
		if err != nil {
			return nil, errors.Wrap(err, "save config")
		}

		// load machine config
		if provider.Config.IsMachineProvider() && workspace.Machine.ID != "" {
			machineConfig, err = provider2.LoadMachineConfig(workspace.Context, workspace.Machine.ID)
			if err != nil {
				return nil, errors.Wrap(err, "load machine config")
			}
		}
	}

	// create a new client
	workspaceClient, err := clientimplementation.NewWorkspaceClient(devPodConfig, provider.Config, workspace, machineConfig, log)
	if err != nil {
		return nil, errors.Wrap(err, "create workspace client")
	}

	return workspaceClient, nil
}

func resolve(defaultProvider *ProviderWithOptions, devPodConfig *config.Config, name, workspaceID, workspaceFolder string, isLocalPath bool) (*provider2.Workspace, error) {
	now := types.Now()
	uid := uuid.New().String()

	// is local folder?
	if isLocalPath {
		return &provider2.Workspace{
			ID:      workspaceID,
			UID:     uid,
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name: defaultProvider.Config.Name,
			},
			Source: provider2.WorkspaceSource{
				LocalFolder: name,
			},
			CreationTimestamp: now,
			LastUsedTimestamp: now,
		}, nil
	}

	// is git?
	gitRepository, gitBranch := normalizeGitRepository(name)
	if strings.HasSuffix(name, ".git") || pingRepository(gitRepository) {
		return &provider2.Workspace{
			ID:      workspaceID,
			UID:     uid,
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name: defaultProvider.Config.Name,
			},
			Source: provider2.WorkspaceSource{
				GitRepository: gitRepository,
				GitBranch:     gitBranch,
			},
			CreationTimestamp: now,
			LastUsedTimestamp: now,
		}, nil
	}

	// is image?
	_, err := image.GetImage(name)
	if err == nil {
		return &provider2.Workspace{
			ID:      workspaceID,
			UID:     uid,
			Folder:  workspaceFolder,
			Context: devPodConfig.DefaultContext,
			Provider: provider2.WorkspaceProviderConfig{
				Name: defaultProvider.Config.Name,
			},
			Source: provider2.WorkspaceSource{
				Image: name,
			},
			CreationTimestamp: now,
			LastUsedTimestamp: now,
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
		str = "https://" + str
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

	str = workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(str, "-"), "")
	if len(str) > 63 {
		str = str[:63]
	}

	return str
}

func selectWorkspace(devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, changeLastUsed bool, log log.Logger) (client.WorkspaceClient, error) {
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
	return loadExistingWorkspace(answer, devPodConfig, ide, changeLastUsed, log)
}

func loadExistingWorkspace(workspaceID string, devPodConfig *config.Config, ide *provider2.WorkspaceIDEConfig, changeLastUsed bool, log log.Logger) (client.WorkspaceClient, error) {
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
	if changeLastUsed || !reflect.DeepEqual(workspaceConfig.IDE, beforeIDE) {
		workspaceConfig.LastUsedTimestamp = types.Now()
		err = provider2.SaveWorkspaceConfig(workspaceConfig)
		if err != nil {
			return nil, err
		}
	}

	// load machine config
	var machineConfig *provider2.Machine
	if workspaceConfig.Machine.ID != "" {
		machineConfig, err = provider2.LoadMachineConfig(workspaceConfig.Context, workspaceConfig.Machine.ID)
		if err != nil {
			return nil, errors.Wrap(err, "load machine config")
		}
	}

	// create client
	return clientimplementation.NewWorkspaceClient(devPodConfig, providerWithOptions.Config, workspaceConfig, machineConfig, log)
}

func saveWorkspaceConfig(workspace *provider2.Workspace) error {
	// save config
	err := provider2.SaveWorkspaceConfig(workspace)
	if err != nil {
		return err
	}

	return nil
}

func createMachine(context, machineID, providerName string) (*provider2.Machine, error) {
	// get the machine dir
	machineDir, err := provider2.GetMachineDir(context, machineID)
	if err != nil {
		return nil, err
	}

	// save machine config
	machine := &provider2.Machine{
		ID:      machineID,
		Folder:  machineDir,
		Context: context,
		Provider: provider2.MachineProviderConfig{
			Name: providerName,
		},
		CreationTimestamp: types.Now(),
	}

	// create machine folder
	err = provider2.SaveMachineConfig(machine)
	if err != nil {
		_ = os.RemoveAll(machineDir)
		return nil, err
	}

	return machine, nil
}
