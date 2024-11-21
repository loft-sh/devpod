// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// DevPodWorkspaceTemplateLister helps list DevPodWorkspaceTemplates.
// All objects returned here must be treated as read-only.
type DevPodWorkspaceTemplateLister interface {
	// List lists all DevPodWorkspaceTemplates in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.DevPodWorkspaceTemplate, err error)
	// Get retrieves the DevPodWorkspaceTemplate from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.DevPodWorkspaceTemplate, error)
	DevPodWorkspaceTemplateListerExpansion
}

// devPodWorkspaceTemplateLister implements the DevPodWorkspaceTemplateLister interface.
type devPodWorkspaceTemplateLister struct {
	listers.ResourceIndexer[*v1.DevPodWorkspaceTemplate]
}

// NewDevPodWorkspaceTemplateLister returns a new DevPodWorkspaceTemplateLister.
func NewDevPodWorkspaceTemplateLister(indexer cache.Indexer) DevPodWorkspaceTemplateLister {
	return &devPodWorkspaceTemplateLister{listers.New[*v1.DevPodWorkspaceTemplate](indexer, v1.Resource("devpodworkspacetemplate"))}
}
