package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/pkg/encoding"
	"github.com/loft-sh/devpod/pkg/random"
)

func (k *kubernetesDriver) TargetArchitecture(ctx context.Context, workspaceId string) (string, error) {
	// namespace
	if k.namespace != "" && k.config.CreateNamespace == "true" {
		k.Log.Debugf("Create namespace '%s'", k.namespace)
		buf := &bytes.Buffer{}
		err := k.runCommand(ctx, []string{"create", "ns", k.namespace}, nil, buf, buf)
		if err != nil {
			k.Log.Debugf("Error creating namespace: %v", err)
		}
	}

	// get target architecture
	k.Log.Infof("Find out cluster architecture...")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := k.runCommand(ctx, []string{"run", "-i", encoding.SafeConcatNameMax([]string{"devpod", workspaceId, random.String(6)}, 32), "-q", "--pod-running-timeout=10m0s", "--rm", "--restart=Never", "--image", k.helperImage(), "--", "sh"}, strings.NewReader("uname -a; exit 0"), stdout, stderr)
	if err != nil {
		return "", fmt.Errorf("find out cluster architecture: %s %s %w", stdout.String(), stderr.String(), err)
	}

	unameOutput := stdout.String()
	if strings.Contains(unameOutput, "arm") || strings.Contains(unameOutput, "aarch") {
		return "arm64", nil
	}

	return "amd64", nil
}

func (k *kubernetesDriver) helperImage() string {
	if k.config.HelperImage != "" {
		return k.config.HelperImage
	}

	return "busybox"
}
