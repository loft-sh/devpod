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
	"errors"
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/errdefs"
	"github.com/compose-spec/compose-go/v2/graph"
	"github.com/compose-spec/compose-go/v2/types"
)

// checkConsistency validate a compose model is consistent
func checkConsistency(project *types.Project) error {
	for _, s := range project.Services {
		if s.Build == nil && s.Image == "" {
			return fmt.Errorf("service %q has neither an image nor a build context specified: %w", s.Name, errdefs.ErrInvalid)
		}

		if s.Build != nil {
			if s.Build.DockerfileInline != "" && s.Build.Dockerfile != "" {
				return fmt.Errorf("service %q declares mutualy exclusive dockerfile and dockerfile_inline: %w", s.Name, errdefs.ErrInvalid)
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
					return fmt.Errorf("service.build.platforms MUST include service.platform %q: %w", s.Platform, errdefs.ErrInvalid)
				}
			}
		}

		if s.NetworkMode != "" && len(s.Networks) > 0 {
			return fmt.Errorf("service %s declares mutually exclusive `network_mode` and `networks`: %w", s.Name, errdefs.ErrInvalid)
		}
		for network := range s.Networks {
			if _, ok := project.Networks[network]; !ok {
				return fmt.Errorf("service %q refers to undefined network %s: %w", s.Name, network, errdefs.ErrInvalid)
			}
		}

		if s.HealthCheck != nil && len(s.HealthCheck.Test) > 0 {
			switch s.HealthCheck.Test[0] {
			case "CMD", "CMD-SHELL", "NONE":
			default:
				return errors.New(`healthcheck.test must start either by "CMD", "CMD-SHELL" or "NONE"`)
			}
		}

		for dependedService, cfg := range s.DependsOn {
			if _, err := project.GetService(dependedService); err != nil {
				if errors.Is(err, errdefs.ErrDisabled) && !cfg.Required {
					continue
				}
				return fmt.Errorf("service %q depends on undefined service %q: %w", s.Name, dependedService, errdefs.ErrInvalid)
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
					return fmt.Errorf("service %q refers to undefined volume %s: %w", s.Name, volume.Source, errdefs.ErrInvalid)
				}
			}
		}
		if s.Build != nil {
			for _, secret := range s.Build.Secrets {
				if _, ok := project.Secrets[secret.Source]; !ok {
					return fmt.Errorf("service %q refers to undefined build secret %s: %w", s.Name, secret.Source, errdefs.ErrInvalid)
				}
			}
		}
		for _, config := range s.Configs {
			if _, ok := project.Configs[config.Source]; !ok {
				return fmt.Errorf("service %q refers to undefined config %s: %w", s.Name, config.Source, errdefs.ErrInvalid)
			}
		}

		for _, secret := range s.Secrets {
			if _, ok := project.Secrets[secret.Source]; !ok {
				return fmt.Errorf("service %q refers to undefined secret %s: %w", s.Name, secret.Source, errdefs.ErrInvalid)
			}
		}

		if s.Scale != nil && s.Deploy != nil {
			if s.Deploy.Replicas != nil && *s.Scale != *s.Deploy.Replicas {
				return fmt.Errorf("services.%s: can't set distinct values on 'scale' and 'deploy.replicas': %w",
					s.Name, errdefs.ErrInvalid)
			}
			s.Deploy.Replicas = s.Scale
		}

		if s.CPUS != 0 && s.Deploy != nil {
			if s.Deploy.Resources.Limits != nil && s.Deploy.Resources.Limits.NanoCPUs.Value() != s.CPUS {
				return fmt.Errorf("services.%s: can't set distinct values on 'cpus' and 'deploy.resources.limits.cpus': %w",
					s.Name, errdefs.ErrInvalid)
			}
		}
		if s.MemLimit != 0 && s.Deploy != nil {
			if s.Deploy.Resources.Limits != nil && s.Deploy.Resources.Limits.MemoryBytes != s.MemLimit {
				return fmt.Errorf("services.%s: can't set distinct values on 'mem_limit' and 'deploy.resources.limits.memory': %w",
					s.Name, errdefs.ErrInvalid)
			}
		}
		if s.MemReservation != 0 && s.Deploy != nil {
			if s.Deploy.Resources.Reservations != nil && s.Deploy.Resources.Reservations.MemoryBytes != s.MemReservation {
				return fmt.Errorf("services.%s: can't set distinct values on 'mem_reservation' and 'deploy.resources.reservations.memory': %w",
					s.Name, errdefs.ErrInvalid)
			}
		}
		if s.PidsLimit != 0 && s.Deploy != nil {
			if s.Deploy.Resources.Limits != nil && s.Deploy.Resources.Limits.Pids != s.PidsLimit {
				return fmt.Errorf("services.%s: can't set distinct values on 'pids_limit' and 'deploy.resources.limits.pids': %w",
					s.Name, errdefs.ErrInvalid)
			}
		}

		if s.GetScale() > 1 && s.ContainerName != "" {
			attr := "scale"
			if s.Scale == nil {
				attr = "deploy.replicas"
			}
			return fmt.Errorf("services.%s: can't set container_name and %s as container name must be unique: %w", attr,
				s.Name, errdefs.ErrInvalid)
		}

		if s.Develop != nil && s.Develop.Watch != nil {
			for _, watch := range s.Develop.Watch {
				if watch.Action != types.WatchActionRebuild && watch.Target == "" {
					return fmt.Errorf("services.%s.develop.watch: target is required for non-rebuild actions: %w", s.Name, errdefs.ErrInvalid)
				}
			}

		}
	}

	for name, secret := range project.Secrets {
		if secret.External {
			continue
		}
		if secret.File == "" && secret.Environment == "" {
			return fmt.Errorf("secret %q must declare either `file` or `environment`: %w", name, errdefs.ErrInvalid)
		}
	}

	return graph.CheckCycle(project)
}
