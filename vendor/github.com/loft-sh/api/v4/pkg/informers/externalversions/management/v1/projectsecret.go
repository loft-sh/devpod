// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	versioned "github.com/loft-sh/api/v4/pkg/clientset/versioned"
	internalinterfaces "github.com/loft-sh/api/v4/pkg/informers/externalversions/internalinterfaces"
	v1 "github.com/loft-sh/api/v4/pkg/listers/management/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ProjectSecretInformer provides access to a shared informer and lister for
// ProjectSecrets.
type ProjectSecretInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ProjectSecretLister
}

type projectSecretInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewProjectSecretInformer constructs a new informer for ProjectSecret type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewProjectSecretInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredProjectSecretInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredProjectSecretInformer constructs a new informer for ProjectSecret type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredProjectSecretInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ManagementV1().ProjectSecrets(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ManagementV1().ProjectSecrets(namespace).Watch(context.TODO(), options)
			},
		},
		&managementv1.ProjectSecret{},
		resyncPeriod,
		indexers,
	)
}

func (f *projectSecretInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredProjectSecretInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *projectSecretInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&managementv1.ProjectSecret{}, f.defaultInformer)
}

func (f *projectSecretInformer) Lister() v1.ProjectSecretLister {
	return v1.NewProjectSecretLister(f.Informer().GetIndexer())
}
