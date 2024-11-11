package update

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
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/terminal"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
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

	// GUI
	instanceEnv := os.Getenv(platform.WorkspaceInstanceEnv)
	if instanceEnv != "" {
		newInstance := &managementv1.DevPodWorkspaceInstance{}
		err := json.Unmarshal([]byte(instanceEnv), newInstance)
		if err != nil {
			return fmt.Errorf("unmarshal workpace instance %s: %w", instanceEnv, err)
		}
		newInstance.TypeMeta = metav1.TypeMeta{} // ignore

		projectName := project.ProjectFromNamespace(newInstance.GetNamespace())
		oldInstance, err := platform.FindInstanceByName(ctx, baseClient, newInstance.GetName(), projectName)
		if err != nil {
			return err
		}

		updatedInstance, err := updateInstance(ctx, baseClient, oldInstance, newInstance, cmd.Log)
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

	// CLI
	if !terminal.IsTerminalIn {
		return fmt.Errorf("unable to update instance through CLI if stdin is not a terminal")
	}
	workspaceID := os.Getenv(platform.WorkspaceIDEnv)
	workspaceUID := os.Getenv(platform.WorkspaceUIDEnv)
	project := os.Getenv(platform.ProjectEnv)
	if workspaceUID == "" || workspaceID == "" || project == "" {
		return fmt.Errorf("workspaceID, workspaceUID or project not found: %s, %s, %s", workspaceID, workspaceUID, project)
	}

	oldInstance, err := platform.FindInstanceInProject(ctx, baseClient, workspaceUID, project)
	if err != nil {
		return err
	}

	newInstance, err := form.UpdateInstance(ctx, baseClient, oldInstance, cmd.Log)
	if err != nil {
		return err
	}

	_, err = updateInstance(ctx, baseClient, oldInstance, newInstance, cmd.Log)
	if err != nil {
		return err
	}

	return nil
}

func updateInstance(ctx context.Context, client client.Client, oldInstance *managementv1.DevPodWorkspaceInstance, newInstance *managementv1.DevPodWorkspaceInstance, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	patch := ctrlclient.MergeFrom(oldInstance)
	data, err := patch.Data(newInstance)
	if err != nil {
		return nil, err
	} else if len(data) == 0 || string(data) == "{}" {
		return newInstance, nil
	}

	res, err := managementClient.Loft().ManagementV1().
		DevPodWorkspaceInstances(oldInstance.GetNamespace()).
		Patch(ctx, oldInstance.GetName(), patch.Type(), data, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}

	return platform.WaitForInstance(ctx, client, res, log)
}
