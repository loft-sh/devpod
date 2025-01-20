package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform/labels"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
)

func List(ctx context.Context, devPodConfig *config.Config, skipPro bool, log log.Logger) ([]*providerpkg.Workspace, error) {
	// list local workspaces
	localWorkspaces, err := ListLocalWorkspaces(devPodConfig.DefaultContext, skipPro, log)
	if err != nil {
		return nil, err
	}

	proWorkspaces := []*providerpkg.Workspace{}
	if !skipPro {
		// list remote workspaces
		proWorkspaces, err = listProWorkspaces(ctx, devPodConfig, log)
		if err != nil {
			return nil, err
		}
		proWorkspacesByUID := map[string]*providerpkg.Workspace{}
		for _, w := range proWorkspaces {
			proWorkspacesByUID[w.UID] = w
		}

		// Check if every local file based workspace has a remote counterpart
		// If not, mark `exists` as false and allow consumers of this function to take necessary measures
		cleanedLocalWorkspaces := []*providerpkg.Workspace{}
		for _, localWorkspace := range localWorkspaces {
			if localWorkspace.IsPro() {
				if _, ok := proWorkspacesByUID[localWorkspace.UID]; !ok {
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
	// merge pro into local with pro taking precedence if UID matches
	for _, workspace := range append(localWorkspaces, proWorkspaces...) {
		workspaces[workspace.UID] = workspace
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

func listProWorkspaces(ctx context.Context, devPodConfig *config.Config, log log.Logger) ([]*providerpkg.Workspace, error) {
	retWorkspaces := []*providerpkg.Workspace{}
	// lock around `retWorkspaces`
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
		if !providerConfig.IsProxyProvider() {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			workspaces, err := listProWorkspacesForProvider(ctx, devPodConfig, provider, providerConfig, log)
			if err != nil {
				log.ErrorStreamOnly().Warn(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			retWorkspaces = append(retWorkspaces, workspaces...)
		}()
	}
	wg.Wait()

	return retWorkspaces, nil
}

func listProWorkspacesForProvider(ctx context.Context, devPodConfig *config.Config, provider string, providerConfig *providerpkg.ProviderConfig, log log.Logger) ([]*providerpkg.Workspace, error) {
	opts := devPodConfig.ProviderOptions(provider)
	opts[providerpkg.LOFT_FILTER_BY_OWNER] = config.OptionValue{Value: "true"}
	var buf bytes.Buffer
	if err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"listWorkspaces",
		providerConfig.Exec.Proxy.List.Workspaces,
		devPodConfig.DefaultContext,
		nil,
		nil,
		opts,
		providerConfig,
		nil, nil, &buf, log.ErrorStreamOnly().Writer(logrus.ErrorLevel, false), log,
	); err != nil {
		return nil, fmt.Errorf("list workspaces for provider \"%s\": %w", provider, err)
	}
	if buf.Len() == 0 {
		return nil, nil
	}

	instances := []managementv1.DevPodWorkspaceInstance{}
	if err := json.Unmarshal(buf.Bytes(), &instances); err != nil {
		log.ErrorStreamOnly().Errorf("unmarshal devpod workspace instances: %w", err)
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
		projectName := instance.GetLabels()[labels.ProjectLabel]

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
				Project:     projectName,
				DisplayName: instance.Spec.DisplayName,
			},
		}
		retWorkspaces = append(retWorkspaces, &workspace)
	}

	return retWorkspaces, nil
}
