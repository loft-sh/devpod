package cluster

import (
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Config will be injected during startup and then passed to the rest storages
var Config *rest.Config

// CachedClient will be injected during startup and then passed to the rest storages
var CachedClient client.Client

// UncachedClient will be injected during startup and then passed to the rest storages
var UncachedClient client.Client

// CachedManagementClient will be injected during startup and then passed to the rest storages
var CachedManagementClient client.Client

// UncachedManagementClient will be injected during startup and then passed to the rest storages
var UncachedManagementClient client.Client
