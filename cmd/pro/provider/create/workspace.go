package create

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/form"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/terminal"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkspaceCmd holds the cmd flags
type WorkspaceCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewWorkspaceCmd creates a new command
func NewWorkspaceCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &WorkspaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance().ErrorStreamOnly(),
	}
	c := &cobra.Command{
		Use:    "workspace",
		Short:  "Create a workspace",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *WorkspaceCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	// fully serialized intance, right now only used by GUI
	instanceEnv := os.Getenv(platform.WorkspaceInstanceEnv)
	if instanceEnv != "" {
		instance := &managementv1.DevPodWorkspaceInstance{} // init pointer
		err := json.Unmarshal([]byte(instanceEnv), instance)
		if err != nil {
			return fmt.Errorf("unmarshal workpace instance %s: %w", instanceEnv, err)
		}

		updatedInstance, err := createInstance(ctx, baseClient, instance, cmd.Log)
		if err != nil {
			return err
		}

		out, err := json.Marshal(updatedInstance)
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	}

	// Info through env, right now only used by CLI
	workspaceID := os.Getenv(provider.WORKSPACE_ID)
	workspaceUID := os.Getenv(provider.WORKSPACE_UID)
	workspaceFolder := os.Getenv(provider.WORKSPACE_FOLDER)
	workspaceContext := os.Getenv(provider.WORKSPACE_CONTEXT)
	if workspaceUID == "" || workspaceID == "" || workspaceFolder == "" {
		return fmt.Errorf("workspaceID, workspaceUID or workspace folder not found: %s, %s, %s", workspaceID, workspaceUID, workspaceFolder)
	}
	instance, err := platform.FindInstance(ctx, baseClient, workspaceUID)
	if err != nil {
		return err
	}
	// Nothing left to do if we already have an instance
	if instance != nil {
		return nil
	}
	if !terminal.IsTerminalIn {
		return fmt.Errorf("unable to create new instance through CLI if stdin is not a terminal")
	}

	instance, err = form.CreateInstance(ctx, baseClient, workspaceID, workspaceUID, cmd.Log)
	if err != nil {
		return err
	}

	_, err = createInstance(ctx, baseClient, instance, cmd.Log)
	if err != nil {
		return err
	}

	// once we have the instance, update workspace and save config
	// TODO: Do we need a file lock?
	workspaceConfig, err := provider.LoadWorkspaceConfig(workspaceContext, workspaceID)
	if err != nil {
		return fmt.Errorf("load workspace config: %w", err)
	}
	workspaceConfig.Pro = &provider.ProMetadata{
		Project:     project.ProjectFromNamespace(instance.GetNamespace()),
		DisplayName: instance.Spec.DisplayName,
	}

	err = provider.SaveWorkspaceConfig(workspaceConfig)
	if err != nil {
		return fmt.Errorf("save workspace config: %w", err)
	}

	return nil
}

func createInstance(ctx context.Context, client client.Client, instance *managementv1.DevPodWorkspaceInstance, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	updatedInstance, err := managementClient.Loft().ManagementV1().
		DevPodWorkspaceInstances(instance.GetNamespace()).
		Create(ctx, instance, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create workspace instance: %w", err)
	}

	return platform.WaitForInstance(ctx, client, updatedInstance, log)
}
