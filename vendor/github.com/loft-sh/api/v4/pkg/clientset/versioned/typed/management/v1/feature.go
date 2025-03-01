// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	context "context"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	scheme "github.com/loft-sh/api/v4/pkg/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// FeaturesGetter has a method to return a FeatureInterface.
// A group's client should implement this interface.
type FeaturesGetter interface {
	Features() FeatureInterface
}

// FeatureInterface has methods to work with Feature resources.
type FeatureInterface interface {
	Create(ctx context.Context, feature *managementv1.Feature, opts metav1.CreateOptions) (*managementv1.Feature, error)
	Update(ctx context.Context, feature *managementv1.Feature, opts metav1.UpdateOptions) (*managementv1.Feature, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, feature *managementv1.Feature, opts metav1.UpdateOptions) (*managementv1.Feature, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*managementv1.Feature, error)
	List(ctx context.Context, opts metav1.ListOptions) (*managementv1.FeatureList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *managementv1.Feature, err error)
	FeatureExpansion
}

// features implements FeatureInterface
type features struct {
	*gentype.ClientWithList[*managementv1.Feature, *managementv1.FeatureList]
}

// newFeatures returns a Features
func newFeatures(c *ManagementV1Client) *features {
	return &features{
		gentype.NewClientWithList[*managementv1.Feature, *managementv1.FeatureList](
			"features",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *managementv1.Feature { return &managementv1.Feature{} },
			func() *managementv1.FeatureList { return &managementv1.FeatureList{} },
		),
	}
}
