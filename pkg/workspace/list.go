package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	daemon "github.com/loft-sh/devpod/pkg/daemon/platform"
	"github.com/loft-sh/devpod/pkg/platform"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const ProjectLabel = "loft.sh/project"

func List(ctx context.Context, devPodConfig *config.Config, skipPro bool, owner platform.OwnerFilter, log log.Logger) ([]*providerpkg.Workspace, error) {
	// list local workspaces
	localWorkspaces, err := ListLocalWorkspaces(devPodConfig.DefaultContext, skipPro, log)
	if err != nil {
		return nil, err
	}

	proWorkspaces := []*providerpkg.Workspace{}
	if !skipPro {
		// list remote workspaces
		proWorkspaceResults, err := listProWorkspaces(ctx, devPodConfig, owner, log)
		if err != nil {
			return nil, err
		}
		// extract pure workspace list first
		for _, result := range proWorkspaceResults {
			proWorkspaces = append(proWorkspaces, result.workspaces...)
		}

		// Check if every local file based workspace has a remote counterpart
		// If not, delete it
		// However, we need to differentiate between workspaces that are legitimately not available anymore
		// and the ones where we were temporarily not able to reach the host
		cleanedLocalWorkspaces := []*providerpkg.Workspace{}
		for _, localWorkspace := range localWorkspaces {
			if localWorkspace.IsPro() {
				if shouldDeleteLocalWorkspace(ctx, localWorkspace, proWorkspaceResults) {
					err = clientimplementation.DeleteWorkspaceFolder(devPodConfig.DefaultContext, localWorkspace.ID, "", log)
					if err != nil {
						log.Debugf("failed to delete local workspace %s: %v", localWorkspace.ID, err)
					}
					continue
				}
			}

			cleanedLocalWorkspaces = append(cleanedLocalWorkspaces, localWorkspaces...)
		}
		localWorkspaces = cleanedLocalWorkspaces
	}

	// Set indexed by UID for deduplication
	workspaces := map[string]*providerpkg.Workspace{}

	// set local workspaces
	for _, workspace := range localWorkspaces {
		workspaces[workspace.UID] = workspace
	}

	// merge pro into local with pro taking precedence if UID matches
	for _, proWorkspace := range proWorkspaces {
		localWorkspace, ok := workspaces[proWorkspace.UID]
		if ok {
			// we want to use the local workspace IDE configuration
			proWorkspace.IDE = localWorkspace.IDE
		}

		workspaces[proWorkspace.UID] = proWorkspace
	}

	retWorkspaces := []*providerpkg.Workspace{}
	for _, v := range workspaces {
		retWorkspaces = append(retWorkspaces, v)
	}

	return retWorkspaces, nil
}

func ListLocalWorkspaces(contextName string, skipPro bool, log log.Logger) ([]*providerpkg.Workspace, error) {
	workspaceDir, err := providerpkg.GetWorkspacesDir(contextName)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	retWorkspaces := []*providerpkg.Workspace{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		workspaceConfig, err := providerpkg.LoadWorkspaceConfig(contextName, entry.Name())
		if err != nil {
			log.ErrorStreamOnly().Warnf("Couldn't load workspace %s: %v", entry.Name(), err)
			continue
		}

		if skipPro && workspaceConfig.IsPro() {
			continue
		}

		retWorkspaces = append(retWorkspaces, workspaceConfig)
	}

	return retWorkspaces, nil
}

var errListProWorkspaces = errors.New("list pro workspaces")

type listProWorkspacesResult struct {
	workspaces []*providerpkg.Workspace
	err        error
}

func listProWorkspaces(ctx context.Context, devPodConfig *config.Config, owner platform.OwnerFilter, log log.Logger) (map[string]listProWorkspacesResult, error) {
	results := map[string]listProWorkspacesResult{}
	// lock around `results`
	var mu sync.Mutex
	wg := sync.WaitGroup{}

	for provider, providerContextConfig := range devPodConfig.Current().Providers {
		if !providerContextConfig.Initialized {
			continue
		}
		l := log.ErrorStreamOnly()
		providerConfig, err := providerpkg.LoadProviderConfig(devPodConfig.DefaultContext, provider)
		if err != nil {
			l.Warnf("load provider config for provider \"%s\": %v", provider, err)
			continue
		}
		// only get pro providers
		if !providerConfig.IsProxyProvider() && !providerConfig.IsDaemonProvider() {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			workspaces, err := listProWorkspacesForProvider(ctx, devPodConfig, provider, providerConfig, owner, log)
			mu.Lock()
			defer mu.Unlock()
			results[provider] = listProWorkspacesResult{
				workspaces: workspaces,
				err:        err,
			}
		}()
	}
	wg.Wait()

	return results, nil
}

func listProWorkspacesForProvider(ctx context.Context, devPodConfig *config.Config, provider string, providerConfig *providerpkg.ProviderConfig, owner platform.OwnerFilter, log log.Logger) ([]*providerpkg.Workspace, error) {
	var (
		instances []managementv1.DevPodWorkspaceInstance
		err       error
	)
	if providerConfig.IsProxyProvider() {
		instances, err = listInstancesProxyProvider(ctx, devPodConfig, provider, providerConfig, log)
	} else if providerConfig.IsDaemonProvider() {
		instances, err = listInstancesDaemonProvider(ctx, devPodConfig, provider, providerConfig, owner, log)
	} else {
		return nil, fmt.Errorf("cannot list pro workspaces with provider %s", provider)
	}
	if err != nil {
		if log.GetLevel() < logrus.DebugLevel {
			log.Warnf("Failed to list pro workspaces for provider %s", provider)
		} else {
			log.Warnf("Failed to list pro workspaces for provider %s: %v", provider, err)
		}
		return nil, err
	}

	retWorkspaces := []*providerpkg.Workspace{}
	for _, instance := range instances {
		if instance.GetLabels() == nil {
			log.Debugf("no labels for pro workspace \"%s\" found, skipping", instance.GetName())
			continue
		}

		// id
		id := instance.GetLabels()[storagev1.DevPodWorkspaceIDLabel]
		if id == "" {
			log.Debugf("no ID label for pro workspace \"%s\" found, skipping", instance.GetName())
			continue
		}

		// uid
		uid := instance.GetLabels()[storagev1.DevPodWorkspaceUIDLabel]
		if uid == "" {
			log.Debugf("no UID label for pro workspace \"%s\" found, skipping", instance.GetName())
			continue
		}

		// project
		projectName := instance.GetLabels()[ProjectLabel]

		// source
		source := providerpkg.WorkspaceSource{}
		if instance.Annotations != nil && instance.Annotations[storagev1.DevPodWorkspaceSourceAnnotation] != "" {
			// source to workspace config source
			rawSource := instance.Annotations[storagev1.DevPodWorkspaceSourceAnnotation]
			s := providerpkg.ParseWorkspaceSource(rawSource)
			if s == nil {
				log.ErrorStreamOnly().Warnf("unable to parse workspace source \"%s\": %v", rawSource)
			} else {
				source = *s
			}
		}

		// last used timestamp
		var lastUsedTimestamp types.Time
		sleepModeConfig := instance.Status.SleepModeConfig
		if sleepModeConfig != nil {
			lastUsedTimestamp = types.Unix(sleepModeConfig.Status.LastActivity, 0)
		} else {
			var ts int64
			if instance.Annotations != nil {
				if val, ok := instance.Annotations["sleepmode.loft.sh/last-activity"]; ok {
					var err error
					if ts, err = strconv.ParseInt(val, 10, 64); err != nil {
						log.Warn("received invalid sleepmode.loft.sh/last-activity from ", instance.GetName())
					}
				}
			}
			lastUsedTimestamp = types.Unix(ts, 0)
		}

		// creation timestamp
		creationTimestamp := types.Time{}
		if !instance.CreationTimestamp.IsZero() {
			creationTimestamp = types.NewTime(instance.CreationTimestamp.Time)
		}

		workspace := providerpkg.Workspace{
			ID:      id,
			UID:     uid,
			Context: devPodConfig.DefaultContext,
			Source:  source,
			Provider: providerpkg.WorkspaceProviderConfig{
				Name: provider,
			},
			LastUsedTimestamp: lastUsedTimestamp,
			CreationTimestamp: creationTimestamp,
			Pro: &providerpkg.ProMetadata{
				InstanceName: instance.GetName(),
				Project:      projectName,
				DisplayName:  instance.Spec.DisplayName,
			},
		}
		retWorkspaces = append(retWorkspaces, &workspace)
	}

	return retWorkspaces, nil
}

func shouldDeleteLocalWorkspace(ctx context.Context, localWorkspace *providerpkg.Workspace, proWorkspaceResults map[string]listProWorkspacesResult) bool {
	// get the correct result for this local workspace
	res, ok := proWorkspaceResults[localWorkspace.Provider.Name]
	if !ok {
		return false
	}
	// Don't delete the workspace if we encountered any error fetching the remote workspaces.
	// This could potentially be destructive so we err or the side of caution and only allow
	// deletion if fetching the remote workspace was successful
	if res.err != nil {
		return false
	}

	if localWorkspace.Imported {
		// does remote still exist?
		if ok := checkInstanceExists(ctx, localWorkspace); ok {
			return false
		}
	}

	hasProCounterpart := slices.ContainsFunc(res.workspaces, func(w *providerpkg.Workspace) bool {
		return localWorkspace.UID == w.UID
	})
	return !hasProCounterpart
}

func listInstancesProxyProvider(ctx context.Context, devPodConfig *config.Config, provider string, providerConfig *providerpkg.ProviderConfig, log log.Logger) ([]managementv1.DevPodWorkspaceInstance, error) {
	opts := devPodConfig.ProviderOptions(provider)
	opts[providerpkg.LOFT_FILTER_BY_OWNER] = config.OptionValue{Value: "true"}
	var stdout bytes.Buffer

	if err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"listWorkspaces",
		providerConfig.Exec.Proxy.List.Workspaces,
		devPodConfig.DefaultContext,
		nil,
		nil,
		opts,
		providerConfig,
		nil, nil, &stdout, log.ErrorStreamOnly().Writer(logrus.ErrorLevel, false), log,
	); err != nil {
		return nil, errListProWorkspaces
	}
	if stdout.Len() == 0 {
		return nil, nil
	}

	instances := []managementv1.DevPodWorkspaceInstance{}
	if err := json.Unmarshal(stdout.Bytes(), &instances); err != nil {
		return nil, err
	}

	return instances, nil
}

func listInstancesDaemonProvider(ctx context.Context, devPodConfig *config.Config, provider string, providerConfig *providerpkg.ProviderConfig, owner platform.OwnerFilter, log log.Logger) ([]managementv1.DevPodWorkspaceInstance, error) {
	dir, err := providerpkg.GetDaemonDir(devPodConfig.DefaultContext, provider)
	if err != nil {
		return nil, err
	}

	return daemon.NewLocalClient(dir, provider).ListWorkspaces(ctx, owner)
}

func checkInstanceExists(ctx context.Context, workspace *providerpkg.Workspace) bool {
	provider := workspace.Provider.Name
	context := workspace.Context
	dir, err := providerpkg.GetDaemonDir(context, provider)
	if err != nil {
		return false
	}

	instance, err := daemon.NewLocalClient(dir, provider).GetWorkspace(ctx, workspace.UID)
	if err != nil || instance == nil {
		return false
	}

	return true
}
