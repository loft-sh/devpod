package image

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	kubernetesauth "github.com/google/go-containerregistry/pkg/authn/kubernetes"
	"gopkg.in/square/go-jose.v2/jwt"
)

const tokenFileLocation = "/var/run/secrets/kubernetes.io/serviceaccount/token"

// See https://github.com/kubernetes/kubernetes/blob/30ae12d018697d3c5f04e225b11f242f5310e097/pkg/serviceaccount/claims.go#L55
type privateClaims struct {
	Kubernetes kubernetesClaim `json:"kubernetes.io,omitempty"`
}

type kubernetesClaim struct {
	Namespace string           `json:"namespace,omitempty"`
	Svcacct   ref              `json:"serviceaccount,omitempty"`
	Pod       *ref             `json:"pod,omitempty"`
	Secret    *ref             `json:"secret,omitempty"`
	Node      *ref             `json:"node,omitempty"`
	WarnAfter *jwt.NumericDate `json:"warnafter,omitempty"`
}

type ref struct {
	Name string `json:"name,omitempty"`
	UID  string `json:"uid,omitempty"`
}

func getKeychain(ctx context.Context) (authn.Keychain, error) {
	var keychain authn.Keychain

	tokenBytes, err := os.ReadFile(tokenFileLocation)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// we're not in a kubernetes pod, use default keychain
			return authn.DefaultKeychain, nil
		}

		return nil, fmt.Errorf("failed to read kubernetes service account token: %w", err)
	}

	// in-cluster auth
	m, err := getPodMetadata(tokenBytes)
	if err != nil {
		return nil, err
	}

	keychain, err = k8schain.NewInCluster(ctx, kubernetesauth.Options{
		ServiceAccountName: m.serviceAccountName,
		Namespace:          m.namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	return keychain, nil
}

type podMetadata struct {
	serviceAccountName string
	namespace          string
}

func getPodMetadata(token []byte) (podMetadata, error) {
	t, err := jwt.ParseSigned(string(token))
	if err != nil {
		return podMetadata{}, fmt.Errorf("failed to parse kubernetes service account token: %w", err)
	}

	privateClaims := privateClaims{}
	err = t.UnsafeClaimsWithoutVerification(&privateClaims)
	if err != nil {
		return podMetadata{}, fmt.Errorf("failed to get claims from kubernetes service account token: %w", err)
	}

	kubeClaim := privateClaims.Kubernetes
	// get serviceaccount name and imagepullsecret
	if kubeClaim.Namespace == "" || kubeClaim.Svcacct.Name == "" {
		return podMetadata{}, fmt.Errorf("failed to retrieve pod metadata from kubernetes service account token: %w", err)
	}

	return podMetadata{
		namespace:          kubeClaim.Namespace,
		serviceAccountName: kubeClaim.Svcacct.Name,
	}, nil
}
