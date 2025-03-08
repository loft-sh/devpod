package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/remotecommand"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	corev1 "k8s.io/api/core/v1"
)

// UpCmd holds the cmd flags:
type UpCmd struct {
	*flags.GlobalFlags

	Log     log.Logger
	streams streams
}

type streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewUpCmd creates a new command
func NewUpCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	logLevel := logrus.InfoLevel
	if os.Getenv(clientimplementation.DevPodDebug) == "true" || globalFlags.Debug {
		logLevel = logrus.DebugLevel
	}

	cmd := &UpCmd{
		GlobalFlags: globalFlags,
		Log: log.NewStreamLoggerWithFormat( /* we don't use stdout */ nil,
			os.Stderr, logLevel, log.JSONFormat).ErrorStreamOnly(),
		streams: streams{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
	}
	c := &cobra.Command{
		Hidden: true,
		Use:    "up",
		Short:  "Runs up on a workspace",
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *UpCmd) Run(ctx context.Context) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	info, err := platform.GetWorkspaceInfoFromEnv()
	if err != nil {
		return err
	}

	instance, err := platform.FindInstanceInProject(ctx, baseClient, info.UID, info.ProjectName)
	if err != nil {
		return err
	} else if instance == nil {
		return fmt.Errorf("workspace %s not found in project %s. Looks like it does not exist anymore and you can delete it", info.ID, info.ProjectName)
	}

	// Log current workspace information. This is both useful to the user to understand the workspace configuration
	// and to us when we receive troubleshooting logs
	printInstanceInfo(instance, cmd.Log)

	if instance.Spec.TemplateRef != nil && templateUpdateRequired(instance) {
		cmd.Log.Info("Template update required")
		oldInstance := instance.DeepCopy()
		instance.Spec.TemplateRef.SyncOnce = true

		instance, err = platform.UpdateInstance(ctx, baseClient, oldInstance, instance, cmd.Log)
		if err != nil {
			return fmt.Errorf("update instance: %w", err)
		}
		cmd.Log.Info("Successfully updated template")
	}

	return cmd.up(ctx, instance, baseClient)
}

func (cmd *UpCmd) up(ctx context.Context, workspace *managementv1.DevPodWorkspaceInstance, client client.Client) error {
	options := platform.OptionsFromEnv(storagev1.DevPodFlagsUp)
	if options != nil && os.Getenv("DEBUG") == "true" {
		options.Add("debug", "true")
	}

	conn, err := platform.DialInstance(client, workspace, "up", options, cmd.Log)
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, cmd.streams.Stdin, cmd.streams.Stdout, cmd.streams.Stderr, cmd.Log)
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}

func templateUpdateRequired(instance *managementv1.DevPodWorkspaceInstance) bool {
	var templateResolved, templateChangesAvailable bool
	for _, condition := range instance.Status.Conditions {
		if condition.Type == storagev1.InstanceTemplateResolved {
			templateResolved = condition.Status == corev1.ConditionTrue
			continue
		}

		if condition.Type == storagev1.InstanceTemplateSynced {
			templateChangesAvailable = condition.Status == corev1.ConditionFalse &&
				condition.Reason == "TemplateChangesAvailable"
			continue
		}
	}

	return !templateResolved || templateChangesAvailable
}

func printInstanceInfo(instance *managementv1.DevPodWorkspaceInstance, log log.Logger) {
	workspaceConfig, _ := json.Marshal(struct {
		// Cluster    storagev1.WorkspaceTargetNamespace
		Template   *storagev1.TemplateRef
		Parameters string
	}{
		// Cluster:    cluster,
		// FIXME: Bring back runner ref
		Template:   instance.Spec.TemplateRef,
		Parameters: instance.Spec.Parameters,
	})
	log.Debug("Starting pro workspace with configuration", string(workspaceConfig))
}
