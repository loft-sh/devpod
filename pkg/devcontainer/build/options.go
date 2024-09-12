package build

type BuildOptions struct {
	BuildArgs map[string]string
	Labels    map[string]string

	CliOpts []string

	Images    []string
	CacheFrom []string
	CacheTo   []string

	Dockerfile string
	Context    string
	Contexts   map[string]string

	Target string

	Load   bool
	Push   bool
	Upload bool
}
