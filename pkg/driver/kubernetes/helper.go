package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	limitsPrefix   = "limits."
	requestsPrefix = "requests."
)

func parseResources(resourceString string, log log.Logger) corev1.ResourceRequirements {
	resourcesSplitted := strings.Split(resourceString, ",")
	requests := corev1.ResourceList{}
	limits := corev1.ResourceList{}
	for _, resourceName := range resourcesSplitted {
		resourceName = strings.TrimSpace(resourceName)

		// requests
		if strings.HasPrefix(resourceName, requestsPrefix) {
			strippedResource := strings.TrimPrefix(resourceName, requestsPrefix)
			name, quantity, err := parseResource(strippedResource)
			if err != nil {
				log.Error(err.Error())
				continue
			}

			requests[corev1.ResourceName(name)] = quantity
		}

		// limits
		if strings.HasPrefix(resourceName, limitsPrefix) {
			strippedResource := strings.TrimPrefix(resourceName, limitsPrefix)
			name, quantity, err := parseResource(strippedResource)
			if err != nil {
				log.Error(err.Error())
				continue
			}

			limits[corev1.ResourceName(name)] = quantity
		}
	}

	return corev1.ResourceRequirements{
		Limits:   limits,
		Requests: requests,
	}
}

func getPodTemplate(manifest string) (*corev1.Pod, error) {
	// check if manifest is inline yaml
	pod := &corev1.Pod{}
	errInline := yaml.Unmarshal([]byte(manifest), pod)
	if errInline == nil {
		return pod, nil
	}

	// check if manifest is path
	p, err := filepath.Abs(manifest)
	if err != nil {
		return nil, fmt.Errorf("parsing pod tempate failed failed: %w (inline) or %w (file)", errInline, err)
	}
	body, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("parsing pod tempate failed failed: %w (inline) or %w (file)", errInline, err)
	}
	err = yaml.Unmarshal(body, pod)
	if err == nil {
		return pod, nil
	}

	return nil, fmt.Errorf("parsing pod tempate failed failed: %w (inline) or %w (file)", errInline, err)
}

func parseLabels(str string) (map[string]string, error) {
	if str == "" {
		return nil, nil
	}

	labels := strings.Split(str, ",")
	retMap := map[string]string{}
	for _, label := range labels {
		label = strings.TrimSpace(label)
		splitted := strings.SplitN(label, "=", 2)
		if len(splitted) != 2 {
			return nil, fmt.Errorf("invalid label '%s', expected format label=value", label)
		}

		retMap[splitted[0]] = splitted[1]
	}

	return retMap, nil
}

func parseResource(resourceName string) (string, resource.Quantity, error) {
	splittedResource := strings.SplitN(resourceName, "=", 2)
	if len(splittedResource) != 2 {
		return "", resource.Quantity{}, fmt.Errorf("error parsing resource %s: expected form resource=quantity", resourceName)
	}

	quantity, err := resource.ParseQuantity(splittedResource[1])
	if err != nil {
		return "", resource.Quantity{}, fmt.Errorf("error parsing resource %s: %w", resourceName, err)
	}

	return splittedResource[0], quantity, nil
}
