package tunnelserver

import (
	"github.com/loft-sh/api/v4/pkg/devpod"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/netstat"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
)

type Option func(*tunnelServer) *tunnelServer

func WithWorkspace(workspace *provider2.Workspace) Option {
	return func(s *tunnelServer) *tunnelServer {
		s.workspace = workspace
		return s
	}
}

func WithForwarder(forwarder netstat.Forwarder) Option {
	return func(s *tunnelServer) *tunnelServer {
		s.forwarder = forwarder
		return s
	}
}

func WithAllowGitCredentials(allowGitCredentials bool) Option {
	return func(s *tunnelServer) *tunnelServer {
		s.allowGitCredentials = allowGitCredentials
		return s
	}
}

func WithAllowDockerCredentials(allowDockerCredentials bool) Option {
	return func(s *tunnelServer) *tunnelServer {
		s.allowDockerCredentials = allowDockerCredentials
		return s
	}
}

func WithAllowKubeConfig(allow bool) Option {
	return func(s *tunnelServer) *tunnelServer {
		s.allowKubeConfig = allow
		return s
	}
}

func WithMounts(mounts []*config.Mount) Option {
	return func(s *tunnelServer) *tunnelServer {
		s.mounts = mounts
		return s
	}
}

func WithPlatformOptions(options *devpod.PlatformOptions) Option {
	return func(s *tunnelServer) *tunnelServer {
		s.platformOptions = options
		return s
	}
}
