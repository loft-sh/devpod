// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"

	v1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	scheme "github.com/loft-sh/api/v4/pkg/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// SelvesGetter has a method to return a SelfInterface.
// A group's client should implement this interface.
type SelvesGetter interface {
	Selves() SelfInterface
}

// SelfInterface has methods to work with Self resources.
type SelfInterface interface {
	Create(ctx context.Context, self *v1.Self, opts metav1.CreateOptions) (*v1.Self, error)
	Update(ctx context.Context, self *v1.Self, opts metav1.UpdateOptions) (*v1.Self, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, self *v1.Self, opts metav1.UpdateOptions) (*v1.Self, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Self, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.SelfList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Self, err error)
	SelfExpansion
}

// selves implements SelfInterface
type selves struct {
	*gentype.ClientWithList[*v1.Self, *v1.SelfList]
}

// newSelves returns a Selves
func newSelves(c *ManagementV1Client) *selves {
	return &selves{
		gentype.NewClientWithList[*v1.Self, *v1.SelfList](
			"selves",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v1.Self { return &v1.Self{} },
			func() *v1.SelfList { return &v1.SelfList{} }),
	}
}
