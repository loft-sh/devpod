// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// DevPodEnvironmentTemplateLister helps list DevPodEnvironmentTemplates.
// All objects returned here must be treated as read-only.
type DevPodEnvironmentTemplateLister interface {
	// List lists all DevPodEnvironmentTemplates in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.DevPodEnvironmentTemplate, err error)
	// Get retrieves the DevPodEnvironmentTemplate from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.DevPodEnvironmentTemplate, error)
	DevPodEnvironmentTemplateListerExpansion
}

// devPodEnvironmentTemplateLister implements the DevPodEnvironmentTemplateLister interface.
type devPodEnvironmentTemplateLister struct {
	listers.ResourceIndexer[*v1.DevPodEnvironmentTemplate]
}

// NewDevPodEnvironmentTemplateLister returns a new DevPodEnvironmentTemplateLister.
func NewDevPodEnvironmentTemplateLister(indexer cache.Indexer) DevPodEnvironmentTemplateLister {
	return &devPodEnvironmentTemplateLister{listers.New[*v1.DevPodEnvironmentTemplate](indexer, v1.Resource("devpodenvironmenttemplate"))}
}
