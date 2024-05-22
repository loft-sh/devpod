// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	scheme "github.com/loft-sh/agentapi/v4/pkg/client/loft/clientset_generated/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterQuotasGetter has a method to return a ClusterQuotaInterface.
// A group's client should implement this interface.
type ClusterQuotasGetter interface {
	ClusterQuotas() ClusterQuotaInterface
}

// ClusterQuotaInterface has methods to work with ClusterQuota resources.
type ClusterQuotaInterface interface {
	Create(ctx context.Context, clusterQuota *v1.ClusterQuota, opts metav1.CreateOptions) (*v1.ClusterQuota, error)
	Update(ctx context.Context, clusterQuota *v1.ClusterQuota, opts metav1.UpdateOptions) (*v1.ClusterQuota, error)
	UpdateStatus(ctx context.Context, clusterQuota *v1.ClusterQuota, opts metav1.UpdateOptions) (*v1.ClusterQuota, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ClusterQuota, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ClusterQuotaList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterQuota, err error)
	ClusterQuotaExpansion
}

// clusterQuotas implements ClusterQuotaInterface
type clusterQuotas struct {
	client rest.Interface
}

// newClusterQuotas returns a ClusterQuotas
func newClusterQuotas(c *ClusterV1Client) *clusterQuotas {
	return &clusterQuotas{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterQuota, and returns the corresponding clusterQuota object, and an error if there is any.
func (c *clusterQuotas) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ClusterQuota, err error) {
	result = &v1.ClusterQuota{}
	err = c.client.Get().
		Resource("clusterquotas").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterQuotas that match those selectors.
func (c *clusterQuotas) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ClusterQuotaList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ClusterQuotaList{}
	err = c.client.Get().
		Resource("clusterquotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterQuotas.
func (c *clusterQuotas) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("clusterquotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a clusterQuota and creates it.  Returns the server's representation of the clusterQuota, and an error, if there is any.
func (c *clusterQuotas) Create(ctx context.Context, clusterQuota *v1.ClusterQuota, opts metav1.CreateOptions) (result *v1.ClusterQuota, err error) {
	result = &v1.ClusterQuota{}
	err = c.client.Post().
		Resource("clusterquotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterQuota).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a clusterQuota and updates it. Returns the server's representation of the clusterQuota, and an error, if there is any.
func (c *clusterQuotas) Update(ctx context.Context, clusterQuota *v1.ClusterQuota, opts metav1.UpdateOptions) (result *v1.ClusterQuota, err error) {
	result = &v1.ClusterQuota{}
	err = c.client.Put().
		Resource("clusterquotas").
		Name(clusterQuota.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterQuota).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *clusterQuotas) UpdateStatus(ctx context.Context, clusterQuota *v1.ClusterQuota, opts metav1.UpdateOptions) (result *v1.ClusterQuota, err error) {
	result = &v1.ClusterQuota{}
	err = c.client.Put().
		Resource("clusterquotas").
		Name(clusterQuota.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterQuota).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the clusterQuota and deletes it. Returns an error if one occurs.
func (c *clusterQuotas) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusterquotas").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterQuotas) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("clusterquotas").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched clusterQuota.
func (c *clusterQuotas) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterQuota, err error) {
	result = &v1.ClusterQuota{}
	err = c.client.Patch(pt).
		Resource("clusterquotas").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
