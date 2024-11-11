package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	loftclient "github.com/loft-sh/api/v4/pkg/clientset/versioned"
	informers "github.com/loft-sh/api/v4/pkg/informers/externalversions"
	informermanagementv1 "github.com/loft-sh/api/v4/pkg/informers/externalversions/management/v1"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// WorkspacesCmd holds the cmd flags
type WorkspacesCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewWorkspacesCmd creates a new command
func NewWorkspacesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &WorkspacesCmd{
		GlobalFlags: globalFlags,
		Log:         log.Default.ErrorStreamOnly(),
	}
	c := &cobra.Command{
		Use:    "workspaces",
		Short:  "Watches all workspaces for a project",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

type ProWorkspaceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   managementv1.DevPodWorkspaceInstanceSpec `json:"spec,omitempty"`
	Status ProWorkspaceInstanceStatus               `json:"status,omitempty"`
}

type ProWorkspaceInstanceStatus struct {
	managementv1.DevPodWorkspaceInstanceStatus `json:",inline"`

	Source *provider.WorkspaceSource    `json:"source,omitempty"`
	IDE    *provider.WorkspaceIDEConfig `json:"ide,omitempty"`
}

func (cmd *WorkspacesCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if cmd.Context == "" {
		cmd.Context = config.DefaultContext
	}

	projectName := os.Getenv(provider.LOFT_PROJECT)
	if projectName == "" {
		return fmt.Errorf("project name not found")
	}

	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	managementConfig, err := baseClient.ManagementConfig()
	if err != nil {
		return err
	}

	clientset, err := loftclient.NewForConfig(managementConfig)
	if err != nil {
		return err
	}

	factory := informers.NewSharedInformerFactoryWithOptions(clientset, time.Second*60,
		informers.WithNamespace(project.ProjectNamespace(projectName)),
	)
	workspaceInformer := factory.Management().V1().DevPodWorkspaceInstances()

	self := baseClient.Self()
	filterByOwner := os.Getenv(provider.LOFT_FILTER_BY_OWNER) == "true"
	instanceStore := newStore(workspaceInformer, self, cmd.Context, filterByOwner, cmd.Log)

	_, err = workspaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			instance, ok := obj.(*managementv1.DevPodWorkspaceInstance)
			if !ok {
				return
			}
			instanceStore.Add(instance)
			printInstances(stdout, instanceStore.List())
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldInstance, ok := oldObj.(*managementv1.DevPodWorkspaceInstance)
			if !ok {
				return
			}
			newInstance, ok := newObj.(*managementv1.DevPodWorkspaceInstance)
			if !ok {
				return
			}
			instanceStore.Update(oldInstance, newInstance)
			printInstances(stdout, instanceStore.List())
		},
		DeleteFunc: func(obj interface{}) {
			instance, ok := obj.(*managementv1.DevPodWorkspaceInstance)
			if !ok {
				// check for DeletedFinalStateUnknown. Can happen if the informer misses the delete event
				u, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					return
				}
				instance, ok = u.Obj.(*managementv1.DevPodWorkspaceInstance)
				if !ok {
					return
				}
			}
			instanceStore.Delete(instance)
			printInstances(stdout, instanceStore.List())
		},
	})
	if err != nil {
		return err
	}

	stopCh := make(chan struct{})
	defer close(stopCh)
	go func() {
		factory.Start(stopCh)
		factory.WaitForCacheSync(stopCh)

		// Kick off initial message
		printInstances(stdout, instanceStore.List())
	}()

	<-stopCh

	return nil
}

type instanceStore struct {
	informer      informermanagementv1.DevPodWorkspaceInstanceInformer
	self          *managementv1.Self
	context       string
	filterByOwner bool

	m         sync.Mutex
	instances map[string]*ProWorkspaceInstance

	log log.Logger
}

func newStore(informer informermanagementv1.DevPodWorkspaceInstanceInformer, self *managementv1.Self, context string, filterByOwner bool, log log.Logger) *instanceStore {
	return &instanceStore{
		informer:      informer,
		self:          self,
		context:       context,
		filterByOwner: filterByOwner,
		instances:     map[string]*ProWorkspaceInstance{},
		log:           log,
	}
}

func (s *instanceStore) key(meta metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s", meta.Namespace, meta.Name)
}

func (s *instanceStore) Add(instance *managementv1.DevPodWorkspaceInstance) {
	if s.filterByOwner && !platform.IsOwner(s.self, instance.Spec.Owner) {
		return
	}
	var source *provider.WorkspaceSource
	if instance.GetAnnotations() != nil && instance.GetAnnotations()[storagev1.DevPodWorkspaceSourceAnnotation] != "" {
		source = provider.ParseWorkspaceSource(instance.GetAnnotations()[storagev1.DevPodWorkspaceSourceAnnotation])
	}

	var ideConfig *provider.WorkspaceIDEConfig
	if instance.GetLabels() != nil && instance.GetLabels()[storagev1.DevPodWorkspaceIDLabel] != "" {
		id := instance.GetLabels()[storagev1.DevPodWorkspaceIDLabel]
		workspaceConfig, err := provider.LoadWorkspaceConfig(s.context, id)
		if err == nil {
			ideConfig = &workspaceConfig.IDE
		}
	}

	proInstance := &ProWorkspaceInstance{
		TypeMeta:   instance.TypeMeta,
		ObjectMeta: instance.ObjectMeta,
		Spec:       instance.Spec,
		Status: ProWorkspaceInstanceStatus{
			DevPodWorkspaceInstanceStatus: instance.Status,
			Source:                        source,
			IDE:                           ideConfig,
		},
	}

	key := s.key(instance.ObjectMeta)
	s.m.Lock()
	s.instances[key] = proInstance
	s.m.Unlock()
}

func (s *instanceStore) Update(oldInstance *managementv1.DevPodWorkspaceInstance, newInstance *managementv1.DevPodWorkspaceInstance) {
	s.Add(newInstance)
}

func (s *instanceStore) Delete(instance *managementv1.DevPodWorkspaceInstance) {
	if s.filterByOwner && !platform.IsOwner(s.self, instance.Spec.Owner) {
		return
	}

	s.m.Lock()
	defer s.m.Unlock()
	key := s.key(instance.ObjectMeta)
	delete(s.instances, key)
}

func (s *instanceStore) List() []*ProWorkspaceInstance {
	instanceList := []*ProWorkspaceInstance{}
	// Check local imported workspaces
	// Eventually this should be implemented by filtering based on ownership and access on the CRD, for now we're stuck with this approach...
	localWorkspaces, err := workspace.ListLocalWorkspaces(s.context, false, s.log)
	if err == nil {
		for _, workspace := range localWorkspaces {
			if workspace.Imported && workspace.Pro != nil {
				// get instance for imported workspace
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: map[string]string{
						storagev1.DevPodWorkspaceUIDLabel: workspace.UID,
					},
				})
				if err != nil {
					continue
				}

				l, err := s.informer.Lister().
					DevPodWorkspaceInstances(project.ProjectFromNamespace(workspace.Pro.Project)).
					List(selector)
				if err != nil {
					continue
				}
				if len(l) == 0 {
					continue
				}
				instance := l[0]
				s.m.Lock()
				if _, ok := s.instances[s.key(instance.ObjectMeta)]; ok {
					continue
				}
				s.m.Unlock()

				var source *provider.WorkspaceSource
				if instance.GetAnnotations() != nil && instance.GetAnnotations()[storagev1.DevPodWorkspaceSourceAnnotation] != "" {
					source = provider.ParseWorkspaceSource(instance.GetAnnotations()[storagev1.DevPodWorkspaceSourceAnnotation])
				}

				var ideConfig *provider.WorkspaceIDEConfig
				if instance.GetLabels() != nil && instance.GetLabels()[storagev1.DevPodWorkspaceIDLabel] != "" {
					id := instance.GetLabels()[storagev1.DevPodWorkspaceIDLabel]
					workspaceConfig, err := provider.LoadWorkspaceConfig(s.context, id)
					if err == nil {
						ideConfig = &workspaceConfig.IDE
					}
				}

				proInstance := &ProWorkspaceInstance{
					TypeMeta:   instance.TypeMeta,
					ObjectMeta: instance.ObjectMeta,
					Spec:       instance.Spec,
					Status: ProWorkspaceInstanceStatus{
						DevPodWorkspaceInstanceStatus: instance.Status,
						Source:                        source,
						IDE:                           ideConfig,
					},
				}
				instanceList = append(instanceList, proInstance)
			}
		}
	}

	s.m.Lock()
	for _, instance := range s.instances {
		instanceList = append(instanceList, instance)
	}
	s.m.Unlock()

	return instanceList
}

func printInstances(w io.Writer, instances []*ProWorkspaceInstance) {
	out, err := json.Marshal(instances)
	if err != nil {
		return
	}

	fmt.Fprintln(w, string(out))
}
