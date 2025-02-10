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

// LicensesGetter has a method to return a LicenseInterface.
// A group's client should implement this interface.
type LicensesGetter interface {
	Licenses() LicenseInterface
}

// LicenseInterface has methods to work with License resources.
type LicenseInterface interface {
	Create(ctx context.Context, license *v1.License, opts metav1.CreateOptions) (*v1.License, error)
	Update(ctx context.Context, license *v1.License, opts metav1.UpdateOptions) (*v1.License, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, license *v1.License, opts metav1.UpdateOptions) (*v1.License, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.License, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.LicenseList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.License, err error)
	LicenseRequest(ctx context.Context, licenseName string, licenseRequest *v1.LicenseRequest, opts metav1.CreateOptions) (*v1.LicenseRequest, error)

	LicenseExpansion
}

// licenses implements LicenseInterface
type licenses struct {
	*gentype.ClientWithList[*v1.License, *v1.LicenseList]
}

// newLicenses returns a Licenses
func newLicenses(c *ManagementV1Client) *licenses {
	return &licenses{
		gentype.NewClientWithList[*v1.License, *v1.LicenseList](
			"licenses",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v1.License { return &v1.License{} },
			func() *v1.LicenseList { return &v1.LicenseList{} }),
	}
}

// LicenseRequest takes the representation of a licenseRequest and creates it.  Returns the server's representation of the licenseRequest, and an error, if there is any.
func (c *licenses) LicenseRequest(ctx context.Context, licenseName string, licenseRequest *v1.LicenseRequest, opts metav1.CreateOptions) (result *v1.LicenseRequest, err error) {
	result = &v1.LicenseRequest{}
	err = c.GetClient().Post().
		Resource("licenses").
		Name(licenseName).
		SubResource("request").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(licenseRequest).
		Do(ctx).
		Into(result)
	return
}
