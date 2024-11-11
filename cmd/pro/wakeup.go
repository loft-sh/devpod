package pro

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// WakeupCmd holds the cmd flags
type WakeupCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Project string
	Host    string
}

// NewWakeupCmd creates a new command
func NewWakeupCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &WakeupCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "wakeup",
		Short: "Wake a workspace up",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			log.Default.SetFormat(log.TextFormat)

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.Project, "project", "", "The project to use")
	_ = c.MarkFlagRequired("project")
	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")

	return c
}

func (cmd *WakeupCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please provide a workspace name")
	}
	targetWorkspace := args[0]

	devPodConfig, err := config.LoadConfig(cmd.Context, "")
	if err != nil {
		return err
	}

	baseClient, err := platform.InitClientFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return err
	}

	workspaceInstance, err := platform.FindInstanceByName(ctx, baseClient, targetWorkspace, cmd.Project)
	if err != nil {
		return err
	}

	if workspaceInstance.Status.Phase != storagev1.InstanceSleeping {
		cmd.Log.Infof("Workspace %s is not sleeping", targetWorkspace, workspaceInstance.Name)
		return nil
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	patch := ctrlclient.MergeFrom(workspaceInstance.DeepCopy())

	if workspaceInstance.Annotations == nil {
		workspaceInstance.Annotations = map[string]string{}
	}
	delete(workspaceInstance.Annotations, clusterv1.SleepModeForceAnnotation)
	delete(workspaceInstance.Annotations, clusterv1.SleepModeForceDurationAnnotation)
	workspaceInstance.Annotations[clusterv1.SleepModeLastActivityAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)

	patchData, err := patch.Data(workspaceInstance)
	if err != nil {
		return err
	}

	_, err = managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(project.ProjectNamespace(cmd.Project)).Patch(ctx, workspaceInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// wait for sleeping
	cmd.Log.Info("Wait until workspace wakes up...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, platform.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		workspaceInstance, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(project.ProjectNamespace(cmd.Project)).Get(ctx, workspaceInstance.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return workspaceInstance.Status.Phase == storagev1.InstanceReady, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for workspace to wake up: %w", err)
	}

	cmd.Log.Donef("Successfully woke up workspace %s", workspaceInstance.Name)
	return nil
}
