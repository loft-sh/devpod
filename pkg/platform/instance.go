package platform

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkspaceInfo struct {
	ID          string
	UID         string
	ProjectName string
}

func GetWorkspaceInfoFromEnv() (*WorkspaceInfo, error) {
	workspaceInfo := &WorkspaceInfo{}
	// get workspace id
	workspaceID := os.Getenv(WorkspaceIDEnv)
	if workspaceID == "" {
		return nil, fmt.Errorf("%s is missing in environment", WorkspaceIDEnv)
	}
	workspaceInfo.ID = workspaceID

	// get workspace uid
	workspaceUID := os.Getenv(WorkspaceUIDEnv)
	if workspaceUID == "" {
		return nil, fmt.Errorf("%s is missing in environment", WorkspaceUIDEnv)
	}
	workspaceInfo.UID = workspaceUID

	// get project
	projectName := os.Getenv(ProjectEnv)
	if projectName == "" {
		return nil, fmt.Errorf("%s is missing in environment", ProjectEnv)
	}
	workspaceInfo.ProjectName = projectName

	return workspaceInfo, nil
}

func FindInstance(ctx context.Context, baseClient client.Client, uid string) (*managementv1.DevPodWorkspaceInstance, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, fmt.Errorf("create management client: %w", err)
	}

	workspaceList, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances("").List(ctx, metav1.ListOptions{
		LabelSelector: storagev1.DevPodWorkspaceUIDLabel + "=" + uid,
	})
	if err != nil {
		return nil, err
	} else if len(workspaceList.Items) == 0 {
		return nil, nil
	}

	return &workspaceList.Items[0], nil
}
func FindInstanceInProject(ctx context.Context, baseClient client.Client, uid, projectName string) (*managementv1.DevPodWorkspaceInstance, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, fmt.Errorf("create management client: %w", err)
	}

	workspaceList, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(project.ProjectNamespace(projectName)).List(ctx, metav1.ListOptions{
		LabelSelector: storagev1.DevPodWorkspaceUIDLabel + "=" + uid,
	})
	if err != nil {
		return nil, err
	} else if len(workspaceList.Items) == 0 {
		return nil, nil
	}

	return &workspaceList.Items[0], nil
}

func FindInstanceByName(ctx context.Context, baseClient client.Client, name, projectName string) (*managementv1.DevPodWorkspaceInstance, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, fmt.Errorf("create management client: %w", err)
	}

	workspace, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(project.ProjectNamespace(projectName)).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

func OptionsFromEnv(name string) url.Values {
	options := os.Getenv(name)
	if options != "" {
		return url.Values{
			"options": []string{options},
		}
	}

	return nil
}

func URLOptions(options any) url.Values {
	raw, _ := json.Marshal(options)
	if options != "" {
		return url.Values{
			"options": []string{string(raw)},
		}
	}

	return nil
}

func DialInstance(baseClient client.Client, workspace *managementv1.DevPodWorkspaceInstance, subResource string, values url.Values, log log.Logger) (*websocket.Conn, error) {
	restConfig, err := baseClient.ManagementConfig()
	if err != nil {
		return nil, err
	}

	host := restConfig.Host
	parsedURL, _ := url.Parse(host)
	if parsedURL != nil && parsedURL.Host != "" {
		host = parsedURL.Host
	}
	log.Debugf("Connect to workspace using host: %s", host)

	loftURL := "wss://" + host + "/kubernetes/management/apis/management.loft.sh/v1/namespaces/" + workspace.Namespace + "/devpodworkspaceinstances/" + workspace.Name + "/" + subResource
	if len(values) > 0 {
		loftURL += "?" + values.Encode()
	}

	dialer := websocket.Dialer{
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	conn, response, err := dialer.Dial(loftURL, map[string][]string{
		"Authorization": {"Bearer " + restConfig.BearerToken},
	})
	if err != nil {
		if response != nil {
			out, _ := io.ReadAll(response.Body)
			headers, _ := json.Marshal(response.Header)
			return nil, fmt.Errorf("%s: error dialing websocket %s (code %d): headers - %s, error - %w", string(out), loftURL, response.StatusCode, string(headers), err)
		}

		return nil, fmt.Errorf("error dialing websocket %s: %w", loftURL, err)
	}

	return conn, nil
}

// UpdateInstance diffs two versions of a DevPodWorkspaceInstance, applies changes via a patch to reduce conflicts.
// Afterwards it waits until the instance is ready to be used.
func UpdateInstance(ctx context.Context, client client.Client, oldInstance *managementv1.DevPodWorkspaceInstance, newInstance *managementv1.DevPodWorkspaceInstance, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	// we don't want to patch status or metadata
	newInstance = newInstance.DeepCopy()
	newInstance.Status = oldInstance.Status
	newInstance.ObjectMeta = oldInstance.ObjectMeta
	newInstance.TypeMeta = oldInstance.TypeMeta

	// create a patch from the old instance
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
		return nil, fmt.Errorf("patch workspace instance: %w (patch: %s)", err, string(data))
	}

	return WaitForInstance(ctx, client, res, log)
}

func WaitForInstance(ctx context.Context, client client.Client, instance *managementv1.DevPodWorkspaceInstance, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	var updatedInstance *managementv1.DevPodWorkspaceInstance
	// we need to wait until instance is scheduled
	err = wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, func(ctx context.Context) (done bool, err error) {
		updatedInstance, err = managementClient.Loft().ManagementV1().
			DevPodWorkspaceInstances(instance.GetNamespace()).
			Get(ctx, instance.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		name := updatedInstance.GetName()
		status := updatedInstance.Status

		if !isReady(updatedInstance) {
			log.Debugf("Workspace %s is in phase %s, waiting until its ready", name, status.Phase)
			return false, nil
		}

		if !isTemplateSynced(updatedInstance) {
			log.Debugf("Workspace template is not ready yet")
			for _, cond := range updatedInstance.Status.Conditions {
				if cond.Status != corev1.ConditionTrue {
					log.Debugf("%s is %s (%s): %s", cond.Type, cond.Status, cond.Reason, cond.Message)
				}
			}
			return false, nil
		}

		log.Debugf("Workspace %s is ready", name)
		return true, nil
	})
	if err != nil {
		// let's build a proper error message here
		msg := "Timed out waiting for workspace to get ready \n\n "
		// basic status
		msg += fmt.Sprintf("ready: %t\n", isReady(updatedInstance))
		msg += fmt.Sprintf("template synced: %t\n", isTemplateSynced(updatedInstance))
		msg += "\n"

		// CRD conditions
		msg += "Conditions:\n"
		for _, cond := range updatedInstance.Status.Conditions {
			msg += fmt.Sprintf("%s is %s (%s): %s\n", cond.Type, cond.Status, cond.Reason, cond.Message)
		}
		msg += "\n"

		// error message, usually context timeout
		msg += fmt.Sprintf("Error: %s", err.Error())

		return nil, errors.New(msg)
	}

	return updatedInstance, nil
}

func isReady(workspace *managementv1.DevPodWorkspaceInstance) bool {
	// Sleeping is considered ready in this context. The workspace will be woken up as soon as we connect to it
	if workspace.Status.Phase == storagev1.InstanceSleeping {
		return true
	}

	return workspace.Status.Phase == storagev1.InstanceReady
}

func isTemplateSynced(workspace *managementv1.DevPodWorkspaceInstance) bool {
	// We're still waiting for the sync to happen
	// The controller will remove this field once it's done syncing
	if workspace.Spec.TemplateRef != nil && workspace.Spec.TemplateRef.SyncOnce {
		return false
	}

	for _, condition := range workspace.Status.Conditions {
		if condition.Type == storagev1.InstanceTemplateResolved {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}
