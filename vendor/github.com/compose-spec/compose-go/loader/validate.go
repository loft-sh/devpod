/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package loader

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/errdefs"
	"github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
)

// checkConsistency validate a compose model is consistent
func checkConsistency(project *types.Project) error {
	for _, s := range project.Services {
		if s.Build == nil && s.Image == "" {
			return errors.Wrapf(errdefs.ErrInvalid, "service %q has neither an image nor a build context specified", s.Name)
		}

		if s.Build != nil {
			if s.Build.DockerfileInline != "" && s.Build.Dockerfile != "" {
				return errors.Wrapf(errdefs.ErrInvalid, "service %q declares mutualy exclusive dockerfile and dockerfile_inline", s.Name)
			}

			if len(s.Build.Platforms) > 0 && s.Platform != "" {
				var found bool
				for _, platform := range s.Build.Platforms {
					if platform == s.Platform {
						found = true
						break
					}
				}
				if !found {
					return errors.Wrapf(errdefs.ErrInvalid, "service.build.platforms MUST include service.platform %q ", s.Platform)
				}
			}
		}

		if s.NetworkMode != "" && len(s.Networks) > 0 {
			return errors.Wrap(errdefs.ErrInvalid, fmt.Sprintf("service %s declares mutually exclusive `network_mode` and `networks`", s.Name))
		}
		for network := range s.Networks {
			if _, ok := project.Networks[network]; !ok {
				return errors.Wrap(errdefs.ErrInvalid, fmt.Sprintf("service %q refers to undefined network %s", s.Name, network))
			}
		}

		if s.HealthCheck != nil && len(s.HealthCheck.Test) > 0 {
			switch s.HealthCheck.Test[0] {
			case "CMD", "CMD-SHELL", "NONE":
			default:
				return errors.New(`healthcheck.test must start either by "CMD", "CMD-SHELL" or "NONE"`)
			}
		}

		for dependedService := range s.DependsOn {
			if _, err := project.GetService(dependedService); err != nil {
				return errors.Wrap(errdefs.ErrInvalid, fmt.Sprintf("service %q depends on undefined service %s", s.Name, dependedService))
			}
		}

		if strings.HasPrefix(s.NetworkMode, types.ServicePrefix) {
			serviceName := s.NetworkMode[len(types.ServicePrefix):]
			if _, err := project.GetServices(serviceName); err != nil {
				return fmt.Errorf("service %q not found for network_mode 'service:%s'", serviceName, serviceName)
			}
		}

		for _, volume := range s.Volumes {
			if volume.Type == types.VolumeTypeVolume && volume.Source != "" { // non anonymous volumes
				if _, ok := project.Volumes[volume.Source]; !ok {
					return errors.Wrap(errdefs.ErrInvalid, fmt.Sprintf("service %q refers to undefined volume %s", s.Name, volume.Source))
				}
			}
		}
		if s.Build != nil {
			for _, secret := range s.Build.Secrets {
				if _, ok := project.Secrets[secret.Source]; !ok {
					return errors.Wrap(errdefs.ErrInvalid, fmt.Sprintf("service %q refers to undefined build secret %s", s.Name, secret.Source))
				}
			}
		}
		for _, config := range s.Configs {
			if _, ok := project.Configs[config.Source]; !ok {
				return errors.Wrap(errdefs.ErrInvalid, fmt.Sprintf("service %q refers to undefined config %s", s.Name, config.Source))
			}
		}

		for _, secret := range s.Secrets {
			if _, ok := project.Secrets[secret.Source]; !ok {
				return errors.Wrap(errdefs.ErrInvalid, fmt.Sprintf("service %q refers to undefined secret %s", s.Name, secret.Source))
			}
		}
	}

	for name, secret := range project.Secrets {
		if secret.External.External {
			continue
		}
		if secret.File == "" && secret.Environment == "" {
			return errors.Wrap(errdefs.ErrInvalid, fmt.Sprintf("secret %q must declare either `file` or `environment`", name))
		}
	}

	return nil
}
