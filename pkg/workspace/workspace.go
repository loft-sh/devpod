package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation/daemonclient"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/encoding"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/ide/ideparse"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/platform"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/terminal"
)

// Resolve takes the `devpod up|build` CLI input and either finds an existing workspace or creates a new one
func Resolve(
	ctx context.Context,
	devPodConfig *config.Config,
	ide string,
	ideOptions []string,
	args []string,
	desiredID,
	desiredMachine string,
	providerUserOptions []string,
	reconfigureProvider bool,
	devContainerImage string,
	devContainerPath string,
	sshConfigPath string,
	source *providerpkg.WorkspaceSource,
	uid string,
	changeLastUsed bool,
	owner platform.OwnerFilter,
	log log.Logger,
) (client.BaseWorkspaceClient, error) {
	// verify desired id
	if desiredID != "" {
		if providerpkg.ProviderNameRegEx.MatchString(desiredID) {
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
		owner,
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

		err = providerpkg.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, fmt.Errorf("save workspace: %w", err)
		}
	}

	// configure dev container source
	if devContainerPath != "" && workspace.DevContainerPath != devContainerPath {
		workspace.DevContainerPath = devContainerPath

		err = providerpkg.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, fmt.Errorf("save workspace: %w", err)
		}
	}

	// configure dev container source
	if workspace.Source.Container != "" {
		err = providerpkg.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, fmt.Errorf("save workspace: %w", err)
		}
	}

	// create workspace client
	client, err := getWorkspaceClient(devPodConfig, provider, workspace, machine, log)
	if err != nil {
		return nil, err
	}

	// refresh provider options
	err = client.RefreshOptions(ctx, providerUserOptions, reconfigureProvider)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getWorkspaceClient(devPodConfig *config.Config, provider *providerpkg.ProviderConfig, workspace *providerpkg.Workspace, machine *providerpkg.Machine, log log.Logger) (client.BaseWorkspaceClient, error) {
	if provider.IsProxyProvider() {
		return clientimplementation.NewProxyClient(devPodConfig, provider, workspace, log)
	} else if provider.IsDaemonProvider() {
		return daemonclient.New(devPodConfig, provider, workspace, log)
	} else {
		return clientimplementation.NewWorkspaceClient(devPodConfig, provider, workspace, machine, log)
	}
}

// Get tries to retrieve an already existing workspace
func Get(ctx context.Context, devPodConfig *config.Config, args []string, changeLastUsed bool, owner platform.OwnerFilter, localOnly bool, log log.Logger) (client.BaseWorkspaceClient, error) {
	var (
		provider  *providerpkg.ProviderConfig
		workspace *providerpkg.Workspace
		machine   *providerpkg.Machine
		err       error
	)

	// check if we have no args
	if len(args) == 0 {
		provider, workspace, machine, err = selectWorkspace(ctx, devPodConfig, changeLastUsed, "", owner, log)
		if err != nil {
			return nil, err
		}
	} else {
		if localOnly {
			workspace = findLocalWorkspace(ctx, devPodConfig, args, "", log)
		} else {
			workspace = findWorkspace(ctx, devPodConfig, args, "", owner, log)
		}
		if workspace == nil {
			return nil, fmt.Errorf("workspace %s doesn't exist", args[0])
		}

		provider, workspace, machine, err = loadExistingWorkspace(devPodConfig, workspace.ID, changeLastUsed, log)
		if err != nil {
			return nil, err
		}
	}

	client, err := getWorkspaceClient(devPodConfig, provider, workspace, machine, log)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Exists checks if the given workspace already exists
func Exists(ctx context.Context, devPodConfig *config.Config, args []string, workspaceID string, owner platform.OwnerFilter, log log.Logger) string {
	workspace := findWorkspace(ctx, devPodConfig, args, workspaceID, owner, log)
	if workspace == nil {
		return ""
	}

	return workspace.ID
}

func resolveWorkspace(
	ctx context.Context,
	devPodConfig *config.Config,
	args []string,
	desiredID,
	desiredMachine string,
	providerUserOptions []string,
	sshConfigPath string,
	source *providerpkg.WorkspaceSource,
	uid string,
	changeLastUsed bool,
	owner platform.OwnerFilter,
	log log.Logger,
) (*providerpkg.ProviderConfig, *providerpkg.Workspace, *providerpkg.Machine, error) {
	// check if we have no args
	if len(args) == 0 {
		if desiredID != "" {
			workspace := findWorkspace(ctx, devPodConfig, nil, desiredID, owner, log)
			if workspace == nil {
				return nil, nil, nil, fmt.Errorf("workspace %s doesn't exist", desiredID)
			}
			return loadExistingWorkspace(devPodConfig, workspace.ID, changeLastUsed, log)
		}

		return selectWorkspace(ctx, devPodConfig, changeLastUsed, sshConfigPath, owner, log)
	}

	// check if workspace already exists
	isLocalPath, name := file.IsLocalDir(args[0])

	// convert to id
	workspaceID := ToID(name)

	// check if desired id already exists
	if desiredID != "" {
		if Exists(ctx, devPodConfig, nil, desiredID, owner, log) != "" {
			log.Debugf("Workspace %s already exists", desiredID)
			return loadExistingWorkspace(devPodConfig, desiredID, changeLastUsed, log)
		}

		// set desired id
		workspaceID = desiredID
	} else if Exists(ctx, devPodConfig, nil, workspaceID, owner, log) != "" {
		log.Debugf("Workspace %s already exists", workspaceID)
		return loadExistingWorkspace(devPodConfig, workspaceID, changeLastUsed, log)
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
	source *providerpkg.WorkspaceSource,
	isLocalPath bool,
	uid string,
	log log.Logger,
) (*providerpkg.ProviderConfig, *providerpkg.Workspace, *providerpkg.Machine, error) {
	// get default provider
	provider, _, err := LoadProviders(devPodConfig, log)
	if err != nil {
		return nil, nil, nil, err
	} else if provider.State == nil || !provider.State.Initialized {
		return nil, nil, nil, fmt.Errorf("provider '%s' is not initialized, please make sure to run 'devpod provider use %s' at least once before using this provider", provider.Config.Name, provider.Config.Name)
	}

	// resolve workspace
	workspace, err := resolveWorkspaceConfig(ctx, provider, devPodConfig, name, workspaceID, source, isLocalPath, sshConfigPath, uid)
	if err != nil {
		return nil, nil, nil, err
	}

	// set server
	if desiredMachine != "" {
		if !provider.Config.IsMachineProvider() {
			return nil, nil, nil, fmt.Errorf("provider %s cannot create servers and cannot be used", provider.Config.Name)
		}

		// check if server exists
		if !providerpkg.MachineExists(workspace.Context, desiredMachine) {
			return nil, nil, nil, fmt.Errorf("server %s doesn't exist and cannot be used", desiredMachine)
		}

		// configure server for workspace
		workspace.Machine = providerpkg.WorkspaceMachineConfig{
			ID: desiredMachine,
		}
	}

	// create a new machine
	var machineConfig *providerpkg.Machine
	if provider.Config.IsMachineProvider() && workspace.Machine.ID == "" {
		// create a new machine
		if provider.State != nil && provider.State.SingleMachine {
			workspace.Machine.ID = SingleMachineName(devPodConfig, provider.Config.Name, log)
		} else {
			workspace.Machine.ID = encoding.CreateNewUIDShort(workspace.ID)
			workspace.Machine.AutoDelete = true
		}

		// save workspace config
		err = providerpkg.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("save config: %w", err)
		}

		// only create machine if it does not exist yet
		if !providerpkg.MachineExists(devPodConfig.DefaultContext, workspace.Machine.ID) {
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
			err = machineClient.RefreshOptions(ctx, providerUserOptions, false)
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
			machineConfig, err = providerpkg.LoadMachineConfig(workspace.Context, workspace.Machine.ID)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("load machine config: %w", err)
			}
		}
	} else if provider.Config.IsProxyProvider() || provider.Config.IsDaemonProvider() {
		// We'll do have to do a bit of mumbo jumbo here because the pro process can't communicate with us directly.
		// It needs os i/o to render the form in CLI mode so we can't go with our typical setup.
		// Instead we first save the config, tell the provider where it lives, it updates it,
		// then we read it again and update to workspace state here
		err = providerpkg.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("save config: %w", err)
		}

		err := resolveProInstance(ctx, devPodConfig, provider.Config.Name, workspace, log)
		if err != nil {
			return nil, nil, nil, err
		}

		workspace, err = providerpkg.LoadWorkspaceConfig(workspace.Context, workspace.ID)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		// save workspace config
		err = providerpkg.SaveWorkspaceConfig(workspace)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("save config: %w", err)
		}

		// load machine config
		if provider.Config.IsMachineProvider() && workspace.Machine.ID != "" {
			machineConfig, err = providerpkg.LoadMachineConfig(workspace.Context, workspace.Machine.ID)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("load machine config: %w", err)
			}
		}
	}

	return provider.Config, workspace, machineConfig, nil
}

func resolveWorkspaceConfig(
	ctx context.Context,
	defaultProvider *ProviderWithOptions,
	devPodConfig *config.Config,
	name,
	workspaceID string,
	source *providerpkg.WorkspaceSource,
	isLocalPath bool,
	sshConfigPath string,
	uid string,
) (*providerpkg.Workspace, error) {
	now := types.Now()
	if uid == "" {
		uid = encoding.CreateNewUID(devPodConfig.DefaultContext, workspaceID)
	}
	workspace := &providerpkg.Workspace{
		ID:      workspaceID,
		UID:     uid,
		Context: devPodConfig.DefaultContext,
		Provider: providerpkg.WorkspaceProviderConfig{
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
		workspace.Source = providerpkg.WorkspaceSource{
			LocalFolder: name,
		}
		return workspace, nil
	}

	// is git?
	gitRepository, gitPRReference, gitBranch, gitCommit, gitSubdir := git.NormalizeRepository(name)
	if strings.HasSuffix(name, ".git") || git.PingRepository(gitRepository, git.GetDefaultExtraEnv(false)) {
		workspace.Picture = getProjectImage(name)
		workspace.Source = providerpkg.WorkspaceSource{
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
		workspace.Source = providerpkg.WorkspaceSource{
			Image: name,
		}
		return workspace, nil
	}

	// fall back to git repository
	workspace.Source = providerpkg.WorkspaceSource{GitRepository: name}
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

func ensureWorkspaceID(args []string, workspaceID string) string {
	if len(args) == 0 && workspaceID == "" {
		return ""
	}

	if workspaceID == "" {
		// check if workspace already exists
		_, name := file.IsLocalDir(args[0])

		// convert to id
		workspaceID = ToID(name)
	}

	return workspaceID
}

func findLocalWorkspace(ctx context.Context, devPodConfig *config.Config, args []string, workspaceID string, log log.Logger) *providerpkg.Workspace {
	workspaceID = ensureWorkspaceID(args, workspaceID)
	if workspaceID == "" {
		return nil
	}

	allWorkspaces, err := ListLocalWorkspaces(devPodConfig.DefaultContext, false, log)
	if err != nil {
		log.Debugf("failed to list workspaces: %v", err)
		return nil
	}

	for _, workspace := range allWorkspaces {
		if workspace.ID != workspaceID {
			continue
		}
		return workspace
	}

	return nil
}

func findWorkspace(ctx context.Context, devPodConfig *config.Config, args []string, workspaceID string, owner platform.OwnerFilter, log log.Logger) *providerpkg.Workspace {
	workspaceID = ensureWorkspaceID(args, workspaceID)
	if workspaceID == "" {
		return nil
	}

	allWorkspaces, err := List(ctx, devPodConfig, false, owner, log)
	if err != nil {
		log.Debugf("failed to list workspaces: %v", err)
		return nil
	}

	var retWorkspace *providerpkg.Workspace
	// already exists in all workspaces (including remote)?
	for _, workspace := range allWorkspaces {
		if workspace.ID != workspaceID {
			continue
		}

		if workspace.IsPro() {
			workspace.Imported = true
			err = providerpkg.SaveWorkspaceConfig(workspace)
			if err != nil {
				log.Debugf("failed to save workspace config for workspace \"%s\" with provider \"%s\": %v", workspace.ID, workspace.Provider.Name, err)
				return nil
			}
		}

		retWorkspace = workspace
		break
	}

	return retWorkspace
}

func selectWorkspace(ctx context.Context, devPodConfig *config.Config, changeLastUsed bool, sshConfigPath string, owner platform.OwnerFilter, log log.Logger) (*providerpkg.ProviderConfig, *providerpkg.Workspace, *providerpkg.Machine, error) {
	if !terminal.IsTerminalIn {
		return nil, nil, nil, errProvideWorkspaceArg
	}

	workspaces, err := List(ctx, devPodConfig, false, owner, log)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list workspaces: %w", err)
	}

	// sort by last used
	sort.SliceStable(workspaces, func(i, j int) bool {
		return workspaces[i].LastUsedTimestamp.Time.Unix() > workspaces[j].LastUsedTimestamp.Time.Unix()
	})

	// prepare form options
	options := []huh.Option[*providerpkg.Workspace]{}
	for _, workspace := range workspaces {
		key := workspace.ID
		if workspace.IsPro() && workspace.Pro.DisplayName != "" {
			key = fmt.Sprintf("%s (%s)", workspace.Pro.DisplayName, workspace.ID)
		}
		options = append(options, huh.NewOption(key, workspace))
	}
	if len(workspaces) == 0 {
		return nil, nil, nil, errors.Join(ErrNoWorkspaceFound, errProvideWorkspaceArg)
	}

	// create terminal form
	var selectedWorkspace *providerpkg.Workspace
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[*providerpkg.Workspace]().
				Title("Please select a workspace from the list below").
				Options(options...).
				Value(&selectedWorkspace),
		),
	)
	if err := form.Run(); err != nil {
		return nil, nil, nil, err
	}
	if selectedWorkspace == nil {
		return nil, nil, nil, fmt.Errorf("no workspace selected")
	}

	// if selected workspace is pro, save config locally
	for _, workspace := range workspaces {
		if workspace.ID == selectedWorkspace.ID && workspace.IsPro() {
			if workspace.SSHConfigPath == "" && sshConfigPath != "" {
				workspace.SSHConfigPath = sshConfigPath
			}
			workspace.Imported = true
			if err := providerpkg.SaveWorkspaceConfig(workspace); err != nil {
				return nil, nil, nil, fmt.Errorf("save workspace config for workspace \"%s\": %w", workspace.ID, err)
			}

			providerConfig, err := providerpkg.LoadProviderConfig(devPodConfig.DefaultContext, workspace.Provider.Name)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("load provider config for workspace \"%s\" with provider \"%s\": %w", workspace.ID, workspace.Provider.Name, err)
			}

			return providerConfig, workspace, nil, nil
		}
	}

	// load workspace
	return loadExistingWorkspace(devPodConfig, selectedWorkspace.ID, changeLastUsed, log)
}

func loadExistingWorkspace(devPodConfig *config.Config, workspaceID string, changeLastUsed bool, log log.Logger) (*providerpkg.ProviderConfig, *providerpkg.Workspace, *providerpkg.Machine, error) {
	workspaceConfig, err := providerpkg.LoadWorkspaceConfig(devPodConfig.DefaultContext, workspaceID)
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
		err = providerpkg.SaveWorkspaceConfig(workspaceConfig)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// load machine config
	var machineConfig *providerpkg.Machine
	if workspaceConfig.Machine.ID != "" {
		machineConfig, err = providerpkg.LoadMachineConfig(workspaceConfig.Context, workspaceConfig.Machine.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("load machine config: %w", err)
		}
	}

	// create client
	return providerWithOptions.Config, workspaceConfig, machineConfig, nil
}

func resolveProInstance(ctx context.Context, devPodConfig *config.Config, providerName string, workspace *providerpkg.Workspace, log log.Logger) error {
	provider, err := FindProvider(devPodConfig, providerName, log)
	if err != nil {
		return err
	}

	workspaceClient, err := getWorkspaceClient(devPodConfig, provider.Config, workspace, nil, log)
	if err != nil {
		return err
	}

	switch c := workspaceClient.(type) {
	case client.ProxyClient:
		return c.Create(ctx, os.Stdin, os.Stdout, os.Stderr)
	case client.DaemonClient:
		return c.Create(ctx, os.Stdin, os.Stdout, os.Stderr)
	default:
		return fmt.Errorf("client does not support remote workspaces")
	}
}
