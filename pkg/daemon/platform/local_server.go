package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/devpod/pkg/platform"
	platformclient "github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/labels"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/net/memnet"
)

type localServer struct {
	devPodContext  string
	httpServer     *http.Server
	lc             *tailscale.LocalClient
	listener       *memnet.Listener
	pc             platformclient.Client
	platformStatus *platformStatus
	log            log.Logger
	stopChan       chan struct{}
}

type Status struct {
	State         DaemonState  `json:"state,omitempty"`
	LoginRequired bool         `json:"loginRequired,omitempty"`
	Debug         *DebugStatus `json:"debug,omitempty"`
}

type DebugStatus struct {
	Tailscale *ipnstate.Status   `json:"tailscale,omitempty"`
	Self      *managementv1.Self `json:"self,omitempty"`
}

type DaemonState string

var (
	DaemonStateRunning DaemonState = "running"
	DaemonStateStopped DaemonState = "stopped"
	DaemonStatePending DaemonState = "pending"
)

const platformStatusCheckInterval = 10 * time.Second

type platformStatus struct {
	mu            sync.RWMutex
	authenticated bool
}

var (
	routeHealth           = "/health"
	routeMetrics          = "/metrics"
	routeStatus           = "/status"
	routeVersion          = "/version"
	routeShutdown         = "/shutdown"
	routeSelf             = "/self"
	routeProjects         = "/projects"
	routeProjectTemplates = "/projects/:project/templates"
	routeProjectClusters  = "/projects/:project/clusters"
	routeGetWorkspace     = "/workspace"
	routeWatchWorkspaces  = "/watch-workspaces"
	routeListWorkspaces   = "/list-workspaces"
	routeCreateWorkspace  = "/create-workspace"
	routeUpdateWorkspace  = "/update-workspace"
)

func newLocalServer(lc *tailscale.LocalClient, pc platformclient.Client, devPodContext string, log log.Logger) (*localServer, error) {
	l := &localServer{
		lc:             lc,
		pc:             pc,
		log:            log,
		devPodContext:  devPodContext,
		listener:       memnet.Listen("localclient.devpod:80"),
		platformStatus: &platformStatus{authenticated: true},
		stopChan:       make(chan struct{}, 1),
	}

	router := httprouter.New()
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, i interface{}) {
		http.Error(w, fmt.Errorf("panic: %s", i).Error(), http.StatusInternalServerError)
		l.log.Error(fmt.Errorf("panic: %s", i), debug.Stack())
	}
	router.GET(routeHealth, l.health)
	router.GET(routeStatus, l.status)
	router.GET(routeVersion, l.version)
	router.GET(routeShutdown, l.shutdown)
	router.GET(routeMetrics, l.metrics)
	router.GET(routeSelf, l.self)
	router.GET(routeProjects, l.projects)
	router.GET(routeProjectTemplates, l.projectTemplates)
	router.GET(routeProjectClusters, l.projectClusters)
	router.GET(routeGetWorkspace, l.getWorkspace)
	router.GET(routeWatchWorkspaces, l.watchWorkspaces)
	router.GET(routeListWorkspaces, l.listWorkspace)
	router.POST(routeCreateWorkspace, l.createWorkspace)
	router.POST(routeUpdateWorkspace, l.updateWorkspace)

	l.httpServer = &http.Server{Handler: handlers.LoggingHandler(log.Writer(logrus.DebugLevel, true), router)}

	return l, nil
}

func (l *localServer) ListenAndServe() error {
	errChan := make(chan error, 1)
	go func() {
		l.log.Info("Start config watcher")
		err := l.watchPlatform(l.stopChan)
		errChan <- err
	}()
	go func() {
		err := l.httpServer.Serve(l.listener)
		errChan <- err
	}()
	return <-errChan
}

func (l *localServer) Close() error {
	l.log.Info("shutting down local server")
	l.stopChan <- struct{}{}
	_ = l.listener.Close()
	return nil
}

func (l *localServer) Addr() string {
	return l.listener.Addr().String()
}

func (l *localServer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	return l.listener.Dial(ctx, network, addr)
}

func (l *localServer) watchPlatform(stopChan <-chan struct{}) error {
	for {
		l.log.Debug("Check platform status")

		managementClient, err := l.pc.Management()
		if err != nil {
			l.log.Error(fmt.Errorf("create mangement client: %w", err))
		} else {
			_, err = managementClient.Loft().ManagementV1().Selves().Create(context.Background(), &managementv1.Self{}, metav1.CreateOptions{})
			l.platformStatus.mu.Lock()
			if err != nil {
				if IsAccessKeyNotFound(err) {
					l.log.Warnf("client not authenticated: %s", err)
					l.platformStatus.authenticated = false
				} else {
					l.log.Errorf("failed to create self: %v", err)
				}
			} else {
				// We don't want to be too restrictive in case the error
				// is transient and doesn't impact existing connections
				l.platformStatus.authenticated = true
			}
			l.platformStatus.mu.Unlock()
		}

		select {
		case <-stopChan:
			return nil
		case <-time.After(platformStatusCheckInterval):
		}
	}
}

func (l *localServer) health(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func (l *localServer) status(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	st, err := l.lc.Status(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status := &Status{}
	// overall state
	switch st.BackendState {
	case ipn.Starting.String():
		status.State = DaemonStatePending
	case ipn.Running.String():
		status.State = DaemonStateRunning
	default:
		// we consider all other states as `stopped`
		status.State = DaemonStateStopped
	}

	// authentication info
	l.platformStatus.mu.RLock()
	if !l.platformStatus.authenticated {
		status.LoginRequired = true
	}
	l.platformStatus.mu.RUnlock()

	// debug info
	self := l.pc.Self()
	self.Status.AccessKey = "*********" // redact access key
	if r.URL.Query().Has("debug") {
		status.Debug = &DebugStatus{
			Tailscale: st,
			Self:      self,
		}
	}

	tryJSON(w, status)
}

func (l *localServer) metrics(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// TODO: Get from tailscale local client
}

func (l *localServer) shutdown(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	err := l.Close()
	if err != nil {
		http.Error(w, fmt.Errorf("shut down daemon server: %v", err).Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type VersionInfo struct {
	ServerVersion string `json:"serverVersion,omitempty"`
}

func (l *localServer) version(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	platformVersion, err := platform.GetPlatformVersion(l.pc.Config().Host)
	if err != nil {
		http.Error(w, fmt.Errorf("get platform version: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	tryJSON(w, VersionInfo{
		ServerVersion: platformVersion.Version,
	})
}

func (l *localServer) self(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	tryJSON(w, l.pc.Self())
}

func (l *localServer) projects(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	managementClient, err := l.pc.Management()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(r.Context(), metav1.ListOptions{})
	if err != nil {
		http.Error(w, fmt.Errorf("list projects: %w", err).Error(), http.StatusInternalServerError)
		return
	} else if len(projectList.Items) == 0 {
		err := fmt.Errorf("you don't have access to any projects, please make sure you have at least access to 1 project")
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	tryJSON(w, projectList.Items)
}

func (l *localServer) projectTemplates(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	projectName := params.ByName("project")
	managementClient, err := l.pc.Management()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templateList, err := managementClient.Loft().ManagementV1().Projects().ListTemplates(r.Context(), projectName, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Errorf("list templates: %w", err).Error(), http.StatusInternalServerError)
		return
	} else if len(templateList.DevPodWorkspaceTemplates) == 0 {
		err := fmt.Errorf("seems like there is no template allowed in project %s, please make sure to at least have a single template available", projectName)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tryJSON(w, templateList)
}
func (l *localServer) projectClusters(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	projectName := params.ByName("project")
	managementClient, err := l.pc.Management()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	clusterList, err := managementClient.Loft().ManagementV1().Projects().ListClusters(r.Context(), projectName, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Errorf("list cluster: %w", err).Error(), http.StatusInternalServerError)
		return
	} else if len(clusterList.Clusters) == 0 {
		err := fmt.Errorf("seems like there is no cluster allowed in project %s, please make sure to at least have a single cluster available", projectName)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tryJSON(w, clusterList)
}

func (l *localServer) listWorkspace(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	managementClient, err := l.pc.Management()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(r.Context(), metav1.ListOptions{})
	if err != nil {
		http.Error(w, fmt.Errorf("list projects: %w", err).Error(), http.StatusInternalServerError)
		return
	} else if len(projectList.Items) == 0 {
		err := fmt.Errorf("you don't have access to any projects, please make sure you have at least access to 1 project")
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	instances := []managementv1.DevPodWorkspaceInstance{}
	for _, p := range projectList.Items {
		ns := project.ProjectNamespace(p.GetName())
		workspaceList, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(ns).List(r.Context(), metav1.ListOptions{})
		if err != nil {
			http.Error(w, fmt.Errorf("list workspaces in project %s: %w", p.GetName(), err).Error(), http.StatusNoContent)
			return
		}

		for _, instance := range workspaceList.Items {
			if instance.GetLabels() == nil {
				instance.Labels = map[string]string{}
			}
			instance.Labels[labels.ProjectLabel] = p.GetName()

			instances = append(instances, instance)
		}
	}

	tryJSON(w, instances)
}

func (l *localServer) getWorkspace(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	uid := r.URL.Query().Get("uid")
	if uid == "" {
		http.Error(w, "missing required query parameter \"uid\"", http.StatusInternalServerError)
		return
	}

	instance, err := platform.FindInstance(r.Context(), l.pc, uid)
	if err != nil {
		http.Error(w, fmt.Errorf("failed to get workspace with uid %s: %w", uid, err).Error(), http.StatusInternalServerError)
		return
	}
	if instance == nil {
		// send OK but don't try to marshal nil instance
		w.WriteHeader(http.StatusOK)
		return
	}

	tryJSON(w, instance)
}

func (l *localServer) watchWorkspaces(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "not a flusher", http.StatusInternalServerError)
		return
	}

	project := r.URL.Query().Get("project")
	if project == "" {
		http.Error(w, "missing required query parameter \"project\"", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err := startWorkspaceWatcher(r.Context(), watchConfig{
		Project:        project,
		Context:        l.devPodContext,
		PlatformClient: l.pc,
		Log:            l.log},
		func(instanceList []*ProWorkspaceInstance) {
			if instanceList != nil {
				err := enc.Encode(instanceList)
				if err != nil {
					http.Error(w, "decode workspace list", http.StatusInternalServerError)
					return
				}
				f.Flush()
			}
		},
	)
	if err != nil {
		http.Error(w, fmt.Errorf("failed to watch workspaces: %w", err).Error(), http.StatusInternalServerError)
		l.log.Error("watch workspaces: %w", err)
		return
	}
}

func (l *localServer) createWorkspace(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	instance := &managementv1.DevPodWorkspaceInstance{}
	err := json.NewDecoder(r.Body).Decode(instance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	instance.TypeMeta = metav1.TypeMeta{}

	updatedInstance, err := createInstance(r.Context(), l.pc, instance, l.log)
	if err != nil {
		http.Error(w, fmt.Errorf("create workspace: %w", err).Error(), http.StatusBadRequest)
		return
	}

	tryJSON(w, updatedInstance)
}

func (l *localServer) updateWorkspace(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	newInstance := &managementv1.DevPodWorkspaceInstance{}
	err := json.NewDecoder(r.Body).Decode(newInstance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newInstance.TypeMeta = metav1.TypeMeta{}

	projectName := project.ProjectFromNamespace(newInstance.GetNamespace())
	oldInstance, err := platform.FindInstanceByName(r.Context(), l.pc, newInstance.GetName(), projectName)
	if err != nil {
		http.Error(w, fmt.Errorf("find old workspace: %w", err).Error(), http.StatusBadRequest)
		return
	}

	updatedInstance, err := updateInstance(r.Context(), l.pc, oldInstance, newInstance, l.log)
	if err != nil {
		http.Error(w, fmt.Errorf("update workspace: %w", err).Error(), http.StatusBadRequest)
		return
	}

	tryJSON(w, updatedInstance)
}

func tryJSON(w http.ResponseWriter, obj interface{}) {
	out, err := json.Marshal(obj)
	if err != nil {
		http.Error(w, fmt.Errorf("marshal: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func createInstance(ctx context.Context, client platformclient.Client, instance *managementv1.DevPodWorkspaceInstance, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
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

func updateInstance(ctx context.Context, client platformclient.Client, oldInstance *managementv1.DevPodWorkspaceInstance, newInstance *managementv1.DevPodWorkspaceInstance, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	// This ensures the template is kept up to date with configuration changes
	if newInstance.Spec.TemplateRef != nil {
		newInstance.Spec.TemplateRef.SyncOnce = true
	}

	return platform.UpdateInstance(ctx, client, oldInstance, newInstance, log)
}
