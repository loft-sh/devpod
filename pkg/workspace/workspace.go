package workspace

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/encoding"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/ide/ideparse"
	"github.com/loft-sh/devpod/pkg/image"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/pkg/errors"
)

var (
	branchRegEx      = regexp.MustCompile(`[^a-zA-Z0-9\.\-]+`)
	prReferenceRegEx = regexp.MustCompile(git.PullRequestReference)
)

func SingleMachineName(devPodConfig *config.Config, provider string, log log.Logger) string {
	legacyMachineName := "devpod-shared-" + provider
	machines, err := listMachines(devPodConfig, log)
	if err == nil {
		for _, machine := range machines {
			if machine.Provider.Name == provider && machine.ID == legacyMachineName {
				return legacyMachineName
			}
		}
	}

	return encoding.SafeConcatNameMax([]string{"devpod-shared", provider, encoding.GetMachineUIDShort(log)}, encoding.MachineUIDLength)
}

// Exists checks if the given workspace already exists
func Exists(devPodConfig *config.Config, args []string) string {
	if len(args) == 0 {
		return ""
	}

	// check if workspace already exists
	_, name := file.IsLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	// already exists?
	if !provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		return ""
	}

	return workspaceID
}

func ListWorkspaces(devPodConfig *config.Config, log log.Logger) ([]*provider2.Workspace, error) {
	workspaceDir, err := provider2.GetWorkspacesDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	retWorkspaces := []*provider2.Workspace{}
	for _, entry := range entries {
		workspaceConfig, err := provider2.LoadWorkspaceConfig(devPodConfig.DefaultContext, entry.Name())
		if err != nil {
			log.ErrorStreamOnly().Warnf("Couldn't load workspace %s: %v", entry.Name(), err)
			continue
		}

		retWorkspaces = append(retWorkspaces, workspaceConfig)
	}

	return retWorkspaces, nil
}

func GetWorkspaceName(args []string) string {
	if len(args) == 0 {
		return ""
	}

	// check if workspace already exists
	_, name := file.IsLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	return workspaceID
}

// GetWorkspace tries to retrieve an already existing workspace
func GetWorkspace(devPodConfig *config.Config, args []string, changeLastUsed bool, log log.Logger) (client.BaseWorkspaceClient, error) {
	provider, workspace, machine, err := getWorkspace(devPodConfig, args, changeLastUsed, log)
	if err != nil {
		return nil, err
	}

	var workspaceClient client.BaseWorkspaceClient
	if provider.IsProxyProvider() {
		workspaceClient, err = clientimplementation.NewProxyClient(devPodConfig, provider, workspace, log)
		if err != nil {
			return nil, err
		}
	} else {
		workspaceClient, err = clientimplementation.NewWorkspaceClient(devPodConfig, provider, workspace, machine, log)
		if err != nil {
			return nil, err
		}
	}

	return workspaceClient, nil
}

func getWorkspace(devPodConfig *config.Config, args []string, changeLastUsed bool, log log.Logger) (*provider2.ProviderConfig, *provider2.Workspace, *provider2.Machine, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(devPodConfig, changeLastUsed, log)
	}

	// check if workspace already exists
	_, name := file.IsLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	// already exists?
	if !provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		return nil, nil, nil, fmt.Errorf("workspace %s doesn't exist", workspaceID)
	}

	// load workspace config
	return loadExistingWorkspace(workspaceID, devPodConfig, changeLastUsed, log)
}

// ResolveWorkspace tries to retrieve an already existing workspace or creates a new one
func ResolveWorkspace(
	ctx context.Context,
	devPodConfig *config.Config,
	ide string,
	ideOptions []string,
	args []string,
	desiredID,
	desiredMachine string,
	providerUserOptions []string,
	devContainerImage string,
	devContainerPath string,
	sshConfigPath string,
	source *provider2.WorkspaceSource,
	uid string,
	changeLastUsed bool,
	log log.Logger,
) (client.BaseWorkspaceClient, error) {
	// verify desired id
	if desiredID != "" {
		if provider2.ProviderNameRegEx.MatchString(desiredID) {
			return nil, fmt.Errorf("workspace name can only include smaller case letters, numbers or dashes")
		} else if len(desiredID) > 48 {
			return nil, fmt.Errorf("workspace name cannot be longer than 48 characters")
		}
	}

	// resolve workspace
	provider, workspace, machine, err := resolveWorkspace(
		ctx,
		devPodConfig,
		args,
		desiredID,
		desiredMachine,
		providerUserOptions,
		sshConfigPath,
		source,
		uid,
		changeLastUsed,
		log,
	)
	if err != nil {
		return nil, err
	}

	// configure ide
	workspace, err = ideparse.RefreshIDEOptions(devPodConfig, workspace, ide, ideOptions)
	if err != nil {
		return nil, err
	}

	// configure dev container source
	if devContainerImage != "" && workspace.DevContainerImage != devContainerImage {
		workspace.DevContainerImage = devContainerImage

		err = provider2.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, errors.Wrap(err, "save workspace")
		}
	}

	// configure dev container source
	if devContainerPath != "" && workspace.DevContainerPath != devContainerPath {
		workspace.DevContainerPath = devContainerPath

		err = provider2.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, errors.Wrap(err, "save workspace")
		}
	}

	// configure dev container source
	if workspace.Source.Container != "" {
		err = provider2.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, errors.Wrap(err, "save workspace")
		}
	}

	// create workspace client
	var workspaceClient client.BaseWorkspaceClient
	if provider.IsProxyProvider() {
		workspaceClient, err = clientimplementation.NewProxyClient(devPodConfig, provider, workspace, log)
		if err != nil {
			return nil, err
		}
	} else {
		workspaceClient, err = clientimplementation.NewWorkspaceClient(devPodConfig, provider, workspace, machine, log)
		if err != nil {
			return nil, err
		}
	}

	// refresh provider options
	err = workspaceClient.RefreshOptions(ctx, providerUserOptions)
	if err != nil {
		return nil, err
	}

	return workspaceClient, nil
}

func resolveWorkspace(
	ctx context.Context,
	devPodConfig *config.Config,
	args []string,
	desiredID,
	desiredMachine string,
	providerUserOptions []string,
	sshConfigPath string,
	source *provider2.WorkspaceSource,
	uid string,
	changeLastUsed bool,
	log log.Logger,
) (*provider2.ProviderConfig, *provider2.Workspace, *provider2.Machine, error) {
	// check if we have no args
	if len(args) == 0 {
		if desiredID != "" {
			return getWorkspace(devPodConfig, []string{desiredID}, changeLastUsed, log)
		}

		return selectWorkspace(devPodConfig, changeLastUsed, log)
	}

	// check if workspace already exists
	isLocalPath, name := file.IsLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	// check if desired id already exists
	if desiredID != "" {
		if provider2.WorkspaceExists(devPodConfig.DefaultContext, desiredID) {
			log.Infof("Workspace %s already exists", desiredID)
			return loadExistingWorkspace(desiredID, devPodConfig, changeLastUsed, log)
		}

		// set desired id
		workspaceID = desiredID
	} else if provider2.WorkspaceExists(devPodConfig.DefaultContext, workspaceID) {
		log.Infof("Workspace %s already exists", workspaceID)
		return loadExistingWorkspace(workspaceID, devPodConfig, changeLastUsed, log)
	}

	// create workspace
	provider, workspace, machine, err := createWorkspace(
		ctx,
		devPodConfig,
		workspaceID,
		name,
		desiredMachine,
		providerUserOptions,
		sshConfigPath,
		source,
		isLocalPath,
		uid,
		log,
	)
	if err != nil {
		_ = clientimplementation.DeleteWorkspaceFolder(devPodConfig.DefaultContext, workspaceID, sshConfigPath, log)
		return nil, nil, nil, err
	}

	return provider, workspace, machine, nil
}

func createWorkspace(
	ctx context.Context,
	devPodConfig *config.Config,
	workspaceID,
	name,
	desiredMachine string,
	providerUserOptions []string,
	sshConfigPath string,
	source *provider2.WorkspaceSource,
	isLocalPath bool,
	uid string,
	log log.Logger,
) (*provider2.ProviderConfig, *provider2.Workspace, *provider2.Machine, error) {
	// get default provider
	provider, _, err := LoadProviders(devPodConfig, log)
	if err != nil {
		return nil, nil, nil, err
	} else if provider.State == nil || !provider.State.Initialized {
		return nil, nil, nil, fmt.Errorf("provider '%s' is not initialized, please make sure to run 'devpod provider use %s' at least once before using this provider", provider.Config.Name, provider.Config.Name)
	}

	// get workspace folder
	workspaceFolder, err := provider2.GetWorkspaceDir(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, nil, nil, err
	}

	// resolve workspace
	workspace, err := resolve(ctx, provider, devPodConfig, name, workspaceID, workspaceFolder, source, isLocalPath, sshConfigPath, uid)
	if err != nil {
		return nil, nil, nil, err
	}

	// set server
	if desiredMachine != "" {
		if !provider.Config.IsMachineProvider() {
			return nil, nil, nil, fmt.Errorf("provider %s cannot create servers and cannot be used", provider.Config.Name)
		}

		// check if server exists
		if !provider2.MachineExists(workspace.Context, desiredMachine) {
			return nil, nil, nil, fmt.Errorf("server %s doesn't exist and cannot be used", desiredMachine)
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
			workspace.Machine.ID = SingleMachineName(devPodConfig, provider.Config.Name, log)
		} else {
			workspace.Machine.ID = encoding.CreateNewUIDShort(workspace.ID)
			workspace.Machine.AutoDelete = true
		}

		// save workspace config
		err = saveWorkspaceConfig(workspace)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "save config")
		}

		// only create machine if it does not exist yet
		if !provider2.MachineExists(devPodConfig.DefaultContext, workspace.Machine.ID) {
			// create machine folder
			machineConfig, err = createMachine(workspace.Context, workspace.Machine.ID, provider.Config.Name)
			if err != nil {
				return nil, nil, nil, err
			}

			// create machine
			machineClient, err := clientimplementation.NewMachineClient(devPodConfig, provider.Config, machineConfig, log)
			if err != nil {
				_ = clientimplementation.DeleteMachineFolder(machineConfig.Context, machineConfig.ID)
				return nil, nil, nil, err
			}

			// refresh options
			err = machineClient.RefreshOptions(ctx, providerUserOptions)
			if err != nil {
				_ = clientimplementation.DeleteMachineFolder(machineConfig.Context, machineConfig.ID)
				return nil, nil, nil, err
			}

			// create machine
			err = machineClient.Create(ctx, client.CreateOptions{})
			if err != nil {
				_ = clientimplementation.DeleteMachineFolder(machineConfig.Context, machineConfig.ID)
				return nil, nil, nil, err
			}
		} else {
			log.Infof("Reuse existing machine '%s' for workspace '%s'", workspace.Machine.ID, workspace.ID)

			// load machine config
			machineConfig, err = provider2.LoadMachineConfig(workspace.Context, workspace.Machine.ID)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "load machine config")
			}
		}
	} else {
		// save workspace config
		err = saveWorkspaceConfig(workspace)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "save config")
		}

		// load machine config
		if provider.Config.IsMachineProvider() && workspace.Machine.ID != "" {
			machineConfig, err = provider2.LoadMachineConfig(workspace.Context, workspace.Machine.ID)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "load machine config")
			}
		}
	}

	return provider.Config, workspace, machineConfig, nil
}

func resolve(
	ctx context.Context,
	defaultProvider *ProviderWithOptions,
	devPodConfig *config.Config,
	name,
	workspaceID,
	workspaceFolder string,
	source *provider2.WorkspaceSource,
	isLocalPath bool,
	sshConfigPath string,
	uid string,
) (*provider2.Workspace, error) {
	now := types.Now()
	if uid == "" {
		uid = encoding.CreateNewUID(devPodConfig.DefaultContext, workspaceID)
	}
	workspace := &provider2.Workspace{
		ID:      workspaceID,
		UID:     uid,
		Context: devPodConfig.DefaultContext,
		Provider: provider2.WorkspaceProviderConfig{
			Name: defaultProvider.Config.Name,
		},
		CreationTimestamp: now,
		LastUsedTimestamp: now,
		SSHConfigPath:     sshConfigPath,
	}

	// outside source set?
	if source != nil {
		workspace.Source = *source
		return workspace, nil
	}

	// is local folder?
	if isLocalPath {
		workspace.Source = provider2.WorkspaceSource{
			LocalFolder: name,
		}
		return workspace, nil
	}

	// is git?
	gitRepository, gitPRReference, gitBranch, gitCommit, gitSubdir := git.NormalizeRepository(name)
	if strings.HasSuffix(name, ".git") || git.PingRepository(gitRepository) {
		workspace.Picture = getProjectImage(name)
		workspace.Source = provider2.WorkspaceSource{
			GitRepository:  gitRepository,
			GitPRReference: gitPRReference,
			GitBranch:      gitBranch,
			GitCommit:      gitCommit,
			GitSubPath:     gitSubdir,
		}

		return workspace, nil
	}

	// is image?
	_, err := image.GetImage(ctx, name)
	if err == nil {
		workspace.Source = provider2.WorkspaceSource{
			Image: name,
		}
		return workspace, nil
	}

	// fall back to git repository
	workspace.Source = provider2.WorkspaceSource{GitRepository: name}
	if gitRepository != "" {
		workspace.Source.GitRepository = gitRepository
	}
	if gitPRReference != "" {
		workspace.Source.GitPRReference = gitPRReference
	}
	if gitBranch != "" {
		workspace.Source.GitBranch = gitBranch
	}
	if gitCommit != "" {
		workspace.Source.GitCommit = gitCommit
	}
	if gitSubdir != "" {
		workspace.Source.GitSubPath = gitSubdir
	}

	return workspace, nil
}

var contentRegEx = regexp.MustCompile(`content="([^"]+)"`)

var regexes = map[string]*regexp.Regexp{
	"github.com": regexp.MustCompile(`(<meta[^>]+property)="og:image" content="([^"]+)"`),
	"gitlab.com": regexp.MustCompile(`(<meta[^>]+content)="([^"]+)" property="og:image"`),
}

func getProjectImage(link string) string {
	if !strings.HasPrefix(link, "http") &&
		!strings.HasPrefix(link, "https") {
		link = "https://" + link
	}

	baseURL, err := url.Parse(link)
	if err != nil {
		return ""
	}

	res, err := http.Get(link)
	if err != nil {
		return ""
	}

	content, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return ""
	}

	html := string(content)

	// Find github social share image: https://css-tricks.com/essential-meta-tags-social-media/
	regEx := regexes[baseURL.Host]
	if regEx == nil {
		return ""
	}

	meta := regEx.FindString(html)
	parts := strings.Split(
		contentRegEx.FindString(meta),
		`"`,
	)

	if len(parts) >= 2 {
		return parts[1]
	}

	return ""
}

var (
	workspaceIDRegEx1 = regexp.MustCompile(`[^\w\-]`)
	workspaceIDRegEx2 = regexp.MustCompile(`[^0-9a-z\-]+`)
)

func ToID(str string) string {
	str = strings.ToLower(filepath.ToSlash(str))
	splitted := strings.Split(str, "@")
	if len(splitted) == 2 {
		// 1. Check if PR was specified
		if prReferenceRegEx.MatchString(str) {
			str = prReferenceRegEx.ReplaceAllStringFunc(splitted[1], git.GetBranchNameForPR)
		} else {
			// 2. Check if a branch name has been specified, if so use this for the ID
			str = strings.TrimSuffix(splitted[1], ".git")
			// Check if branch name matches expected regex
			if !branchRegEx.MatchString(str) {
				str = splitted[0]
			}
		}
	} else {
		// 3. If not, then parse the repo name as ID
		index := strings.LastIndex(str, "/")
		if index != -1 {
			str = str[index+1:]

			// remove a potential tag / branch name
			if len(splitted) == 2 && !branchRegEx.MatchString(splitted[1]) {
				str = splitted[0]
			}

			// remove .git if there is it
			str = strings.TrimSuffix(str, ".git")
		}
	}

	str = workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(str, "-"), "")
	if len(str) > 48 {
		str = str[:48]
	}

	return strings.Trim(str, "-")
}

func selectWorkspace(devPodConfig *config.Config, changeLastUsed bool, log log.Logger) (*provider2.ProviderConfig, *provider2.Workspace, *provider2.Machine, error) {
	if !terminal.IsTerminalIn {
		return nil, nil, nil, errProvideWorkspaceArg
	}

	// ask which workspace to use
	workspacesDir, err := provider2.GetWorkspacesDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, nil, nil, err
	}

	workspaceIDs := []string{}
	workspacesDirs, err := os.ReadDir(workspacesDir)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, workspace := range workspacesDirs {
		name := workspace.Name()
		// filter out hidden files
		if !strings.HasPrefix(name, ".") {
			workspaceIDs = append(workspaceIDs, name)
		}
	}
	if len(workspaceIDs) == 0 {
		return nil, nil, nil, errProvideWorkspaceArg
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please select a workspace from the list below",
		DefaultValue: workspaceIDs[0],
		Options:      workspaceIDs,
		Sort:         true,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	// load workspace
	return loadExistingWorkspace(answer, devPodConfig, changeLastUsed, log)
}

func loadExistingWorkspace(workspaceID string, devPodConfig *config.Config, changeLastUsed bool, log log.Logger) (*provider2.ProviderConfig, *provider2.Workspace, *provider2.Machine, error) {
	workspaceConfig, err := provider2.LoadWorkspaceConfig(devPodConfig.DefaultContext, workspaceID)
	if err != nil {
		return nil, nil, nil, err
	}

	providerWithOptions, err := FindProvider(devPodConfig, workspaceConfig.Provider.Name, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// save workspace config
	if changeLastUsed {
		workspaceConfig.LastUsedTimestamp = types.Now()
		err = provider2.SaveWorkspaceConfig(workspaceConfig)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// load machine config
	var machineConfig *provider2.Machine
	if workspaceConfig.Machine.ID != "" {
		machineConfig, err = provider2.LoadMachineConfig(workspaceConfig.Context, workspaceConfig.Machine.ID)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "load machine config")
		}
	}

	// create client
	return providerWithOptions.Config, workspaceConfig, machineConfig, nil
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
		Context: context,
		Provider: provider2.MachineProviderConfig{
			Name: providerName,
		},
		CreationTimestamp: types.Now(),
		Origin:            filepath.Join(machineDir, provider2.MachineConfigFile),
	}

	// create machine folder
	err = provider2.SaveMachineConfig(machine)
	if err != nil {
		_ = os.RemoveAll(machineDir)
		return nil, err
	}

	return machine, nil
}
