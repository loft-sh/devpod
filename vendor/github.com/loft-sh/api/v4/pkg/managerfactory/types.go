package managerfactory

import (
	"context"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SharedManagerFactory is the interface for retrieving managers
type SharedManagerFactory interface {
	Cluster(cluster string) ClusterClientAccess
	Management() ManagementClientAccess
}

// ClusterClientAccess holds the functions for cluster access
type ClusterClientAccess interface {
	Config(ctx context.Context) (*rest.Config, error)
	// FIXME(ThomasK33): This is a workaround for the direct connection issue for the
	// remote workload vcluster pro feature. This function should be removed as
	// soon as remote control plane feature is refactored.
	ConfigSkippingDirectConnection(ctx context.Context) (*rest.Config, error)
	UncachedClient(ctx context.Context) (client.Client, error)
}

// ManagementClientAccess holds the functions for management access
type ManagementClientAccess interface {
	Config() *rest.Config
	UncachedClient() client.Client
	CachedClient() client.Client
	Cache() cache.Cache
}
