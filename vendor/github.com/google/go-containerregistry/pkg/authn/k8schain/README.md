# `k8schain`

This is an implementation of the [`authn.Keychain`](https://godoc.org/github.com/google/go-containerregistry/authn#Keychain) interface loosely based on the authentication semantics used by the Kubelet when performing the pull of a Pod's images.

This keychain supports passing a Kubernetes Service Account and some ImagePullSecrets which may represent registry credentials.

In addition to those, the keychain also includes cloud-specific credential helpers for Google Container Registry (and Artifact Registry), Azure Container Registry, and Amazon AWS Elasic Container Registry.
This means that if the keychain is used from within Kubernetes services on those clouds (GKE, AKS, EKS), any available service credentials will be discovered and used.

In general this keychain should be used when the code is expected to run in a Kubernetes cluster, and especially when it will run in one of those clouds.
To get a cloud-agnostic keychain, use [`pkg/authn/kubernetes`](../kubernetes) instead.

To get only cloud-aware keychains, use [`google.Keychain`](https://godoc.org/github.com/google/go-containerregistry/pkg/v1/google#Keychain), or [`pkg/authn.NewKeychainFromHelper`](https://godoc.org/github.com/google/go-containerregistry/pkg/authn#NewKeychainFromHelper) with a cloud credential helper implementation -- see the implementation of `k8schain.NewNoClient` for more details.

## Usage

### Creating a keychain

A `k8schain` keychain can be built via one of:

```go
// client is a kubernetes.Interface
kc, err := k8schain.New(ctx, client, k8schain.Options{})
...

// This method is suitable for use by controllers or other in-cluster processes.
kc, err := k8schain.NewInCluster(ctx, k8schain.Options{})
...
```

### Using the keychain

The `k8schain` keychain can be used directly as an `authn.Keychain`, e.g.

```go
auth, err := kc.Resolve(registry)
if err != nil {
	...
}
```

Or, with the [`remote.WithAuthFromKeychain`](https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#WithAuthFromKeychain) option:

```go
img, err := remote.Image(ref, remote.WithAuthFromKeychain(kc))
if err != nil {
	...
}
```
