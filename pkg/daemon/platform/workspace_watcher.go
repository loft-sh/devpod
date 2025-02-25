package daemon

import (
	"context"
	"fmt"
	"sync"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	loftclient "github.com/loft-sh/api/v4/pkg/clientset/versioned"
	informers "github.com/loft-sh/api/v4/pkg/informers/externalversions"
	informermanagementv1 "github.com/loft-sh/api/v4/pkg/informers/externalversions/management/v1"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

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

type watchConfig struct {
	Context        string
	Project        string
	PlatformClient client.Client
	Log            log.Logger
}

type changeFn func(instanceList []*ProWorkspaceInstance)

func startWorkspaceWatcher(ctx context.Context, config watchConfig, onChange changeFn) error {
	self := config.PlatformClient.Self()
	managementConfig, err := config.PlatformClient.ManagementConfig()
	if err != nil {
		return err
	}

	clientset, err := loftclient.NewForConfig(managementConfig)
	if err != nil {
		return err
	}

	factory := informers.NewSharedInformerFactoryWithOptions(clientset, time.Second*60,
		informers.WithNamespace(project.ProjectNamespace(config.Project)),
	)
	workspaceInformer := factory.Management().V1().DevPodWorkspaceInstances()

	instanceStore := newStore(workspaceInformer, self, config.Context, config.Log)

	_, err = workspaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			instance, ok := obj.(*managementv1.DevPodWorkspaceInstance)
			if !ok {
				return
			}
			instanceStore.Add(instance)
			onChange(instanceStore.List())
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
			onChange(instanceStore.List())
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
			onChange(instanceStore.List())
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
		onChange(instanceStore.List())
	}()
	go func() {
		<-ctx.Done()
		stopCh <- struct{}{}
	}()

	<-stopCh
	config.Log.Debug("workspace watcher done")

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

func newStore(informer informermanagementv1.DevPodWorkspaceInstanceInformer, self *managementv1.Self, context string, log log.Logger) *instanceStore {
	return &instanceStore{
		informer:  informer,
		self:      self,
		context:   context,
		instances: map[string]*ProWorkspaceInstance{},
		log:       log,
	}
}

func (s *instanceStore) key(meta metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s", meta.Namespace, meta.Name)
}

func (s *instanceStore) Add(instance *managementv1.DevPodWorkspaceInstance) {
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
	s.m.Lock()
	defer s.m.Unlock()

	instanceList := []*ProWorkspaceInstance{}
	for _, instance := range s.instances {
		instanceList = append(instanceList, instance)
	}

	return instanceList
}
