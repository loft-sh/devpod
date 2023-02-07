package build

import "io"

type BuildOptions struct {
	BuildArgs map[string]string
	Labels    map[string]string

	Images    []string
	CacheFrom []string

	Dockerfile    string
	Context       string
	ContextReader io.Reader
	Contexts      map[string]string

	Target string

	Load bool
	Push bool
}
