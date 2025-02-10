package project

import (
	"strings"
	"sync"
)

var DefaultProjectNamespacePrefix = "loft-p-"

// having a nil value means the prefix is unset and things should panic and not fail silently
var prefix *string
var prefixMux sync.RWMutex

// SetProjectNamespacePrefix sets the global project namespace prefix
// Defaulting should be handled when reading the config via ParseProjectNamespacePrefix
func SetProjectNamespacePrefix(newPrefix string) {
	prefixMux.Lock()
	defer prefixMux.Unlock()

	prefix = &newPrefix
}

func GetProjectNamespacePrefix() string {
	prefixMux.Lock()
	defer prefixMux.Unlock()

	if prefix == nil {
		panic("Seems like you forgot to init the project namespace prefix. This is a requirement as otherwise resolving a project namespace is not possible.")
	}

	return *prefix
}

// ParseConfiguredProjectNSPrefix handles the defaulting for a configured prefix and returns the prefix to be used
func ParseConfiguredProjectNSPrefix(configuredPrefix *string) string {
	if configuredPrefix == nil {
		return DefaultProjectNamespacePrefix
	}

	return *configuredPrefix
}

// ProjectFromNamespace returns the project associated with the namespace
func ProjectFromNamespace(namespace string) string {
	prefixMux.RLock()
	defer prefixMux.RUnlock()

	if prefix == nil {
		panic("Seems like you forgot to init the project namespace prefix. This is a requirement as otherwise resolving a project namespace is not possible.")
	}

	return strings.TrimPrefix(namespace, *prefix)
}

// ProjectNamespace returns the namespace associated with the project
func ProjectNamespace(projectName string) string {
	prefixMux.RLock()
	defer prefixMux.RUnlock()

	if prefix == nil {
		panic("Seems like you forgot to init the project namespace prefix. This is a requirement as otherwise resolving a project namespace is not possible.")
	}

	return *prefix + projectName
}
