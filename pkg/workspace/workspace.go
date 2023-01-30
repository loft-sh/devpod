package workspace

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
	"github.com/loft-sh/devpod/provider/gcloud"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var provideWorkspaceArgErr = fmt.Errorf("please provide a workspace name. E.g. 'devpod up ./my-folder', 'devpod up github.com/my-org/my-repo' or 'devpod up ubuntu'")

func getProvider() types.Provider {
	// TODO: remove hardcode
	// provider := provider2.NewWorkspaceProviderWrapper(docker.NewDockerProvider())
	gcloudProvider, err := gcloud.NewProvider(gcloud.ProviderConfig{}, log.Default)
	if err != nil {
		panic(err)
	}

	provider := provider2.NewServerProviderWrapper(gcloudProvider)
	return provider
}

func GetWorkspace(args []string, log log.Logger) (*config.Workspace, types.Provider, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(log)
	}

	// check if workspace already exists
	localFolder, _ := filepath.Abs(args[0])
	workspaceID := localFolder
	if workspaceID == "" {
		workspaceID = args[0]
	}

	// convert to id
	workspaceID = ToWorkspaceID(workspaceID)

	// already exists?
	if !config.WorkspaceExists(workspaceID) {
		return nil, nil, fmt.Errorf("workspace %s doesn't exist", workspaceID)
	}

	workspaceConfig, err := config.LoadWorkspaceConfig(workspaceID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "load workspace config")
	}

	return workspaceConfig, getProvider(), nil
}

func ResolveWorkspace(args []string, log log.Logger) (*config.Workspace, types.Provider, error) {
	provider := getProvider()

	// check if we have no args
	if len(args) == 0 {
		return selectWorkspace(log)
	}

	// check if workspace already exists
	localFolder, _ := filepath.Abs(args[0])
	workspaceID := localFolder
	if workspaceID == "" {
		workspaceID = args[0]
	}

	// convert to id
	workspaceID = ToWorkspaceID(workspaceID)

	// already exists?
	if config.WorkspaceExists(workspaceID) {
		workspaceConfig, err := config.LoadWorkspaceConfig(workspaceID)
		if err != nil {
			return nil, nil, errors.Wrap(err, "load workspace config")
		}

		return workspaceConfig, provider, nil
	}

	// is local folder?
	if localFolder != "" {
		_, err := os.Stat(localFolder)
		if err == nil {
			return &config.Workspace{
				ID: workspaceID,
				Source: config.WorkspaceSource{
					LocalFolder: localFolder,
				},
			}, provider, nil
		}
	}

	// is git?
	gitRepository := normalizeGitRepository(args[0])
	if strings.HasSuffix(args[0], ".git") || pingRepository(gitRepository) {
		return &config.Workspace{
			ID: workspaceID,
			Source: config.WorkspaceSource{
				GitRepository: gitRepository,
			},
		}, provider, nil
	}

	// is image?
	_, err := image.GetImage(args[0])
	if err == nil {
		return &config.Workspace{
			ID: workspaceID,
			Source: config.WorkspaceSource{
				Image: args[0],
			},
		}, provider, nil
	}

	return nil, nil, fmt.Errorf("")
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

func selectWorkspace(log log.Logger) (*config.Workspace, types.Provider, error) {
	provider := getProvider()
	if !terminal.IsTerminalIn {
		return nil, nil, provideWorkspaceArgErr
	}

	// ask which workspace to use
	workspacesDir, err := config.GetWorkspacesDir()
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

	workspaceConfig, err := config.LoadWorkspaceConfig(answer)
	if err != nil {
		return nil, nil, err
	}

	return workspaceConfig, provider, nil
}
