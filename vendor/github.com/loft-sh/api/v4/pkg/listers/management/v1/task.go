// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TaskLister helps list Tasks.
// All objects returned here must be treated as read-only.
type TaskLister interface {
	// List lists all Tasks in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Task, err error)
	// Get retrieves the Task from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.Task, error)
	TaskListerExpansion
}

// taskLister implements the TaskLister interface.
type taskLister struct {
	indexer cache.Indexer
}

// NewTaskLister returns a new TaskLister.
func NewTaskLister(indexer cache.Indexer) TaskLister {
	return &taskLister{indexer: indexer}
}

// List lists all Tasks in the indexer.
func (s *taskLister) List(selector labels.Selector) (ret []*v1.Task, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Task))
	})
	return ret, err
}

// Get retrieves the Task from the index for a given name.
func (s *taskLister) Get(name string) (*v1.Task, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("task"), name)
	}
	return obj.(*v1.Task), nil
}
