package daemon

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	loftclient "github.com/loft-sh/api/v4/pkg/clientset/versioned"
	typedmanagementv1 "github.com/loft-sh/api/v4/pkg/clientset/versioned/typed/management/v1"
	informers "github.com/loft-sh/api/v4/pkg/informers/externalversions"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

type ProWorkspaceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   managementv1.DevPodWorkspaceInstanceSpec `json:"spec,omitempty"`
	Status ProWorkspaceInstanceStatus               `json:"status,omitempty"`
}

type ProWorkspaceInstanceStatus struct {
	managementv1.DevPodWorkspaceInstanceStatus `json:",inline"`

	Source  *provider.WorkspaceSource       `json:"source,omitempty"`
	IDE     *provider.WorkspaceIDEConfig    `json:"ide,omitempty"`
	Metrics *WorkspaceNetworkMetricsSummary `json:"metrics,omitempty"`
}

type watchConfig struct {
	Context        string
	Project        string
	PlatformClient client.Client
	TsClient       *tailscale.LocalClient
	OwnerFilter    platform.OwnerFilter
	Log            log.Logger
}

type changeFn func(instanceList []*ProWorkspaceInstance)

func startWorkspaceWatcher(ctx context.Context, config watchConfig, onChange changeFn) error {
	self := config.PlatformClient.Self()
	managementConfig, err := config.PlatformClient.ManagementConfig()
	if err != nil {
		return err
	}

	clientset, err := getClientSet(managementConfig)
	if err != nil {
		return err
	}

	started := &atomic.Bool{}
	factory := informers.NewSharedInformerFactoryWithOptions(clientset, time.Second*10,
		informers.WithNamespace(project.ProjectNamespace(config.Project)),
	)
	workspaceInformer := factory.Management().V1().DevPodWorkspaceInstances()
	instanceStore := newStore(self, config.Context, config.OwnerFilter, config.TsClient, config.Log)
	_, err = workspaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			instance, ok := obj.(*managementv1.DevPodWorkspaceInstance)
			if !ok {
				return
			}
			instanceStore.Add(instance)
			if started.Load() {
				onChange(instanceStore.List())
			}
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
			if started.Load() {
				onChange(instanceStore.List())
			}
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
			if started.Load() {
				onChange(instanceStore.List())
			}
		},
	})
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				config.Log.Errorf("panic in workspace watcher: %v\n%s", err, debug.Stack())
			}
		}()

		config.Log.Info("starting workspace watcher")
		factory.Start(ctx.Done())
		factory.WaitForCacheSync(ctx.Done())
		started.Store(true)

		// Kick off initial message
		onChange(instanceStore.List())

		// periodically collect workspace metrics
		instanceStore.collectWorkspaceMetrics(ctx, onChange)
	}()

	<-ctx.Done()
	config.Log.Debug("workspace watcher done")
	return nil
}

type instanceStore struct {
	self        *managementv1.Self
	context     string
	ownerFilter platform.OwnerFilter

	m         sync.Mutex
	instances map[string]*ProWorkspaceInstance
	tsClient  *tailscale.LocalClient

	metricsMu        sync.RWMutex
	metrics          map[string][]WorkspaceNetworkMetrics
	maxMetricSamples int

	log log.Logger
}

type ConnectionType string

const (
	ConnectionTypeDirect ConnectionType = "direct"
	ConnectionTypeDERP   ConnectionType = "DERP"
)

type WorkspaceNetworkMetricsSummary struct {
	LatencyMs          float64        `json:"latencyMs,omitempty"`
	LastConnectionType ConnectionType `json:"connectionType,omitempty"`
	LastDERPRegion     string         `json:"derpRegion,omitempty"`
}

type WorkspaceNetworkMetrics struct {
	LatencyMs      float64        `json:"latencyMs,omitempty"`
	ConnectionType ConnectionType `json:"connectionType,omitempty"`
	DERPRegion     string         `json:"derpRegion,omitempty"`
	Timestamp      int64          `json:"timestamp,omitempty"`
}

func newStore(self *managementv1.Self, context string, ownerFilter platform.OwnerFilter, tsClient *tailscale.LocalClient, log log.Logger) *instanceStore {
	return &instanceStore{
		self:             self,
		context:          context,
		instances:        map[string]*ProWorkspaceInstance{},
		ownerFilter:      ownerFilter,
		tsClient:         tsClient,
		metrics:          map[string][]WorkspaceNetworkMetrics{},
		maxMetricSamples: 6,
		log:              log,
	}
}

func (s *instanceStore) key(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func (s *instanceStore) Add(instance *managementv1.DevPodWorkspaceInstance) {
	if s.ownerFilter == platform.SelfOwnerFilter && !platform.IsOwner(s.self, instance.GetOwner()) {
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

	key := s.key(instance.ObjectMeta.Namespace, instance.ObjectMeta.Name)
	s.m.Lock()
	s.instances[key] = proInstance
	s.m.Unlock()
}

func (s *instanceStore) Update(oldInstance *managementv1.DevPodWorkspaceInstance, newInstance *managementv1.DevPodWorkspaceInstance) {
	if s.ownerFilter == platform.SelfOwnerFilter && !platform.IsOwner(s.self, newInstance.GetOwner()) {
		return
	}
	s.Add(newInstance)
}

func (s *instanceStore) Delete(instance *managementv1.DevPodWorkspaceInstance) {
	if s.ownerFilter == platform.SelfOwnerFilter && !platform.IsOwner(s.self, instance.GetOwner()) {
		return
	}
	s.m.Lock()
	defer s.m.Unlock()
	key := s.key(instance.ObjectMeta.Namespace, instance.ObjectMeta.Name)
	delete(s.instances, key)

	// delete from metrics as well
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	delete(s.metrics, key)
}

func (s *instanceStore) List() []*ProWorkspaceInstance {
	s.m.Lock()
	defer s.m.Unlock()

	instanceList := []*ProWorkspaceInstance{}
	for _, instance := range s.instances {
		instanceList = append(instanceList, s.convert(instance))
	}

	return instanceList
}

func (s *instanceStore) convert(instance *ProWorkspaceInstance) *ProWorkspaceInstance {
	if instance == nil {
		return nil
	}

	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()

	metrics := s.metrics[s.key(instance.ObjectMeta.Namespace, instance.ObjectMeta.Name)]
	if len(metrics) > 0 {
		totalMetrics := len(metrics)
		// calculate average latency
		var totalLatency float64
		for _, metric := range metrics {
			totalLatency += metric.LatencyMs
		}
		avgLatency := totalLatency / float64(totalMetrics)

		instance.Status.Metrics = &WorkspaceNetworkMetricsSummary{
			LatencyMs:          avgLatency,
			LastConnectionType: metrics[totalMetrics-1].ConnectionType,
			LastDERPRegion:     metrics[totalMetrics-1].DERPRegion,
		}
	}

	return instance
}

func (s *instanceStore) collectWorkspaceMetrics(ctx context.Context, onChange changeFn) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// let's kick it off once
	s.updateWorkspaceLatencies(ctx)
	onChange(s.List())

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateWorkspaceLatencies(ctx)
			onChange(s.List())
		}
	}
}

func (s *instanceStore) updateWorkspaceLatencies(ctx context.Context) {
	status, err := s.tsClient.Status(ctx)
	if err != nil {
		s.log.Errorf("Failed to get tailscale status: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, peer := range status.Peer {
		if len(peer.TailscaleIPs) == 0 {
			continue
		}
		instanceName, projectName, err := ts.ParseWorkspaceHostname(peer.HostName)
		if err != nil {
			s.log.Debugf("failed to parse hostname for peer %s: %v", peer.HostName, err)
			continue
		}
		key := fmt.Sprintf("%s/%s", project.ProjectNamespace(projectName), instanceName)
		s.m.Lock()
		instance := s.instances[key]
		s.m.Unlock()
		if instance == nil {
			continue
		}

		wg.Add(1)
		go func(peer *ipnstate.PeerStatus, key string, instance *ProWorkspaceInstance) {
			defer wg.Done()

			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			s.log.Debugf("pinging workspace %s/%s", instance.GetNamespace(), instance.GetName())
			pingResult, err := s.tsClient.Ping(timeoutCtx, peer.TailscaleIPs[0], tailcfg.PingDisco)
			if err != nil {
				s.log.Debugf("Failed to ping workspace %s/%s: %v", instance.GetNamespace(), instance.GetName(), err)
				return
			}
			if pingResult.Err != "" {
				s.log.Debugf("Failed to ping workspace %s/%s: %v", instance.GetNamespace(), instance.GetName(), pingResult.Err)
				return
			}

			// Determine connection type
			connectionType := ConnectionTypeDirect
			derpRegion := ""
			if pingResult.DERPRegionID != 0 {
				connectionType = ConnectionTypeDERP
				derpRegion = pingResult.DERPRegionCode
			}

			s.metricsMu.Lock()
			s.metrics[key] = append(
				s.metrics[key],
				WorkspaceNetworkMetrics{
					LatencyMs:      pingResult.LatencySeconds * 1000,
					ConnectionType: connectionType,
					DERPRegion:     derpRegion,
					Timestamp:      time.Now().Unix(),
				},
			)
			// trim down to max samples if necessary
			if len(s.metrics[key]) > s.maxMetricSamples {
				s.metrics[key] = s.metrics[key][1:]
			}
			s.metricsMu.Unlock()
		}(peer, key, instance)
	}

	wg.Wait()
}

func getClientSet(config *rest.Config) (loftclient.Interface, error) {
	clientset, err := loftclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	mv1 := clientset.ManagementV1()
	c := typedmanagementv1.New(&extendedRESTClient{Interface: mv1.RESTClient()})

	return &extendedClientset{
		Clientset:        clientset,
		ManagementClient: c,
	}, nil
}

var _ rest.Interface = (*extendedRESTClient)(nil)

type extendedClientset struct {
	*loftclient.Clientset
	ManagementClient typedmanagementv1.ManagementV1Interface
}

func (c *extendedClientset) ManagementV1() typedmanagementv1.ManagementV1Interface {
	return c.ManagementClient
}

type extendedRESTClient struct {
	rest.Interface
}

func (e *extendedRESTClient) Get() *rest.Request {
	req := e.Interface.Get()
	// We need to pass this to the backend for more information on the management CRD status
	req.Param("extended", "true")
	req.Param("resync", "10") // resync every 10 seconds in the watch request

	return req
}
