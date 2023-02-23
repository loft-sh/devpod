package workspace

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	options2 "github.com/loft-sh/devpod/pkg/provider/options"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
	"github.com/pkg/errors"
	"os"
	"reflect"
)

// GetServerClient creates a server client
func GetServerClient(ctx context.Context, devPodConfig *config.Config, args []string, log log.Logger) (client.Client, error) {
	// check if we have no args
	if len(args) == 0 {
		return selectServer(ctx, devPodConfig, log)
	}

	// check if workspace already exists
	_, name := isLocalDir(args[0], log)

	// convert to id
	serverID := ToWorkspaceID(name)

	// already exists?
	if !provider2.ServerExists(devPodConfig.DefaultContext, serverID) {
		return nil, fmt.Errorf("server %s doesn't exist", serverID)
	}

	// load workspace config
	return loadExistingServer(ctx, serverID, devPodConfig, log)
}

func selectServer(ctx context.Context, devPodConfig *config.Config, log log.Logger) (client.Client, error) {
	if !terminal.IsTerminalIn {
		return nil, provideWorkspaceArgErr
	}

	// ask which server to use
	serversDir, err := provider2.GetServersDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	serverIDs := []string{}
	seversDirs, err := os.ReadDir(serversDir)
	for _, workspace := range seversDirs {
		serverIDs = append(serverIDs, workspace.Name())
	}
	if len(serverIDs) == 0 {
		return nil, provideWorkspaceArgErr
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please select a server from the list below",
		DefaultValue: serverIDs[0],
		Options:      serverIDs,
		Sort:         true,
	})
	if err != nil {
		return nil, err
	}

	// load workspace
	return loadExistingServer(ctx, answer, devPodConfig, log)
}

func loadExistingServer(ctx context.Context, serverID string, devPodConfig *config.Config, log log.Logger) (client.Client, error) {
	serverConfig, err := provider2.LoadServerConfig(devPodConfig.DefaultContext, serverID)
	if err != nil {
		return nil, err
	}

	providerWithOptions, err := FindProvider(devPodConfig, serverConfig.Provider.Name, log)
	if err != nil {
		return nil, err
	}

	// resolve options
	beforeOptions := serverConfig.Provider.Options
	serverConfig, err = options2.ResolveOptionsServer(ctx, "", "", serverConfig, providerWithOptions.Config)
	if err != nil {
		return nil, errors.Wrap(err, "resolve options")
	}

	// save workspace config
	if !reflect.DeepEqual(serverConfig.Provider.Options, beforeOptions) {
		err = provider2.SaveServerConfig(serverConfig)
		if err != nil {
			return nil, err
		}
	}

	return clientimplementation.NewServerClient(providerWithOptions.Config, serverConfig, log), nil
}
