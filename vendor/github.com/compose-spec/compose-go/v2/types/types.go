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

package types

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/go-connections/nat"
)

// ServiceConfig is the configuration of one service
type ServiceConfig struct {
	Name     string   `yaml:"name,omitempty" json:"-"`
	Profiles []string `yaml:"profiles,omitempty" json:"profiles,omitempty"`

	Annotations  Mapping        `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Attach       *bool          `yaml:"attach,omitempty" json:"attach,omitempty"`
	Build        *BuildConfig   `yaml:"build,omitempty" json:"build,omitempty"`
	Develop      *DevelopConfig `yaml:"develop,omitempty" json:"develop,omitempty"`
	BlkioConfig  *BlkioConfig   `yaml:"blkio_config,omitempty" json:"blkio_config,omitempty"`
	CapAdd       []string       `yaml:"cap_add,omitempty" json:"cap_add,omitempty"`
	CapDrop      []string       `yaml:"cap_drop,omitempty" json:"cap_drop,omitempty"`
	CgroupParent string         `yaml:"cgroup_parent,omitempty" json:"cgroup_parent,omitempty"`
	Cgroup       string         `yaml:"cgroup,omitempty" json:"cgroup,omitempty"`
	CPUCount     int64          `yaml:"cpu_count,omitempty" json:"cpu_count,omitempty"`
	CPUPercent   float32        `yaml:"cpu_percent,omitempty" json:"cpu_percent,omitempty"`
	CPUPeriod    int64          `yaml:"cpu_period,omitempty" json:"cpu_period,omitempty"`
	CPUQuota     int64          `yaml:"cpu_quota,omitempty" json:"cpu_quota,omitempty"`
	CPURTPeriod  int64          `yaml:"cpu_rt_period,omitempty" json:"cpu_rt_period,omitempty"`
	CPURTRuntime int64          `yaml:"cpu_rt_runtime,omitempty" json:"cpu_rt_runtime,omitempty"`
	CPUS         float32        `yaml:"cpus,omitempty" json:"cpus,omitempty"`
	CPUSet       string         `yaml:"cpuset,omitempty" json:"cpuset,omitempty"`
	CPUShares    int64          `yaml:"cpu_shares,omitempty" json:"cpu_shares,omitempty"`

	// Command for the service containers.
	// If set, overrides COMMAND from the image.
	//
	// Set to `[]` or an empty string to clear the command from the image.
	Command ShellCommand `yaml:"command,omitempty" json:"command"` // NOTE: we can NOT omitempty for JSON! see ShellCommand type for details.

	Configs           []ServiceConfigObjConfig `yaml:"configs,omitempty" json:"configs,omitempty"`
	ContainerName     string                   `yaml:"container_name,omitempty" json:"container_name,omitempty"`
	CredentialSpec    *CredentialSpecConfig    `yaml:"credential_spec,omitempty" json:"credential_spec,omitempty"`
	DependsOn         DependsOnConfig          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Deploy            *DeployConfig            `yaml:"deploy,omitempty" json:"deploy,omitempty"`
	DeviceCgroupRules []string                 `yaml:"device_cgroup_rules,omitempty" json:"device_cgroup_rules,omitempty"`
	Devices           []DeviceMapping          `yaml:"devices,omitempty" json:"devices,omitempty"`
	DNS               StringList               `yaml:"dns,omitempty" json:"dns,omitempty"`
	DNSOpts           []string                 `yaml:"dns_opt,omitempty" json:"dns_opt,omitempty"`
	DNSSearch         StringList               `yaml:"dns_search,omitempty" json:"dns_search,omitempty"`
	Dockerfile        string                   `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	DomainName        string                   `yaml:"domainname,omitempty" json:"domainname,omitempty"`

	// Entrypoint for the service containers.
	// If set, overrides ENTRYPOINT from the image.
	//
	// Set to `[]` or an empty string to clear the entrypoint from the image.
	Entrypoint ShellCommand `yaml:"entrypoint,omitempty" json:"entrypoint"` // NOTE: we can NOT omitempty for JSON! see ShellCommand type for details.

	Environment     MappingWithEquals                `yaml:"environment,omitempty" json:"environment,omitempty"`
	EnvFiles        []EnvFile                        `yaml:"env_file,omitempty" json:"env_file,omitempty"`
	Expose          StringOrNumberList               `yaml:"expose,omitempty" json:"expose,omitempty"`
	Extends         *ExtendsConfig                   `yaml:"extends,omitempty" json:"extends,omitempty"`
	ExternalLinks   []string                         `yaml:"external_links,omitempty" json:"external_links,omitempty"`
	ExtraHosts      HostsList                        `yaml:"extra_hosts,omitempty" json:"extra_hosts,omitempty"`
	GroupAdd        []string                         `yaml:"group_add,omitempty" json:"group_add,omitempty"`
	Hostname        string                           `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	HealthCheck     *HealthCheckConfig               `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty"`
	Image           string                           `yaml:"image,omitempty" json:"image,omitempty"`
	Init            *bool                            `yaml:"init,omitempty" json:"init,omitempty"`
	Ipc             string                           `yaml:"ipc,omitempty" json:"ipc,omitempty"`
	Isolation       string                           `yaml:"isolation,omitempty" json:"isolation,omitempty"`
	Labels          Labels                           `yaml:"labels,omitempty" json:"labels,omitempty"`
	CustomLabels    Labels                           `yaml:"-" json:"-"`
	Links           []string                         `yaml:"links,omitempty" json:"links,omitempty"`
	Logging         *LoggingConfig                   `yaml:"logging,omitempty" json:"logging,omitempty"`
	LogDriver       string                           `yaml:"log_driver,omitempty" json:"log_driver,omitempty"`
	LogOpt          map[string]string                `yaml:"log_opt,omitempty" json:"log_opt,omitempty"`
	MemLimit        UnitBytes                        `yaml:"mem_limit,omitempty" json:"mem_limit,omitempty"`
	MemReservation  UnitBytes                        `yaml:"mem_reservation,omitempty" json:"mem_reservation,omitempty"`
	MemSwapLimit    UnitBytes                        `yaml:"memswap_limit,omitempty" json:"memswap_limit,omitempty"`
	MemSwappiness   UnitBytes                        `yaml:"mem_swappiness,omitempty" json:"mem_swappiness,omitempty"`
	MacAddress      string                           `yaml:"mac_address,omitempty" json:"mac_address,omitempty"`
	Net             string                           `yaml:"net,omitempty" json:"net,omitempty"`
	NetworkMode     string                           `yaml:"network_mode,omitempty" json:"network_mode,omitempty"`
	Networks        map[string]*ServiceNetworkConfig `yaml:"networks,omitempty" json:"networks,omitempty"`
	OomKillDisable  bool                             `yaml:"oom_kill_disable,omitempty" json:"oom_kill_disable,omitempty"`
	OomScoreAdj     int64                            `yaml:"oom_score_adj,omitempty" json:"oom_score_adj,omitempty"`
	Pid             string                           `yaml:"pid,omitempty" json:"pid,omitempty"`
	PidsLimit       int64                            `yaml:"pids_limit,omitempty" json:"pids_limit,omitempty"`
	Platform        string                           `yaml:"platform,omitempty" json:"platform,omitempty"`
	Ports           []ServicePortConfig              `yaml:"ports,omitempty" json:"ports,omitempty"`
	Privileged      bool                             `yaml:"privileged,omitempty" json:"privileged,omitempty"`
	PullPolicy      string                           `yaml:"pull_policy,omitempty" json:"pull_policy,omitempty"`
	ReadOnly        bool                             `yaml:"read_only,omitempty" json:"read_only,omitempty"`
	Restart         string                           `yaml:"restart,omitempty" json:"restart,omitempty"`
	Runtime         string                           `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Scale           *int                             `yaml:"scale,omitempty" json:"scale,omitempty"`
	Secrets         []ServiceSecretConfig            `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	SecurityOpt     []string                         `yaml:"security_opt,omitempty" json:"security_opt,omitempty"`
	ShmSize         UnitBytes                        `yaml:"shm_size,omitempty" json:"shm_size,omitempty"`
	StdinOpen       bool                             `yaml:"stdin_open,omitempty" json:"stdin_open,omitempty"`
	StopGracePeriod *Duration                        `yaml:"stop_grace_period,omitempty" json:"stop_grace_period,omitempty"`
	StopSignal      string                           `yaml:"stop_signal,omitempty" json:"stop_signal,omitempty"`
	StorageOpt      map[string]string                `yaml:"storage_opt,omitempty" json:"storage_opt,omitempty"`
	Sysctls         Mapping                          `yaml:"sysctls,omitempty" json:"sysctls,omitempty"`
	Tmpfs           StringList                       `yaml:"tmpfs,omitempty" json:"tmpfs,omitempty"`
	Tty             bool                             `yaml:"tty,omitempty" json:"tty,omitempty"`
	Ulimits         map[string]*UlimitsConfig        `yaml:"ulimits,omitempty" json:"ulimits,omitempty"`
	User            string                           `yaml:"user,omitempty" json:"user,omitempty"`
	UserNSMode      string                           `yaml:"userns_mode,omitempty" json:"userns_mode,omitempty"`
	Uts             string                           `yaml:"uts,omitempty" json:"uts,omitempty"`
	VolumeDriver    string                           `yaml:"volume_driver,omitempty" json:"volume_driver,omitempty"`
	Volumes         []ServiceVolumeConfig            `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	VolumesFrom     []string                         `yaml:"volumes_from,omitempty" json:"volumes_from,omitempty"`
	WorkingDir      string                           `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// MarshalYAML makes ServiceConfig implement yaml.Marshaller
func (s ServiceConfig) MarshalYAML() (interface{}, error) {
	type t ServiceConfig
	value := t(s)
	value.Name = "" // set during map to slice conversion, not part of the yaml representation
	return value, nil
}

// NetworksByPriority return the service networks IDs sorted according to Priority
func (s *ServiceConfig) NetworksByPriority() []string {
	type key struct {
		name     string
		priority int
	}
	var keys []key
	for k, v := range s.Networks {
		priority := 0
		if v != nil {
			priority = v.Priority
		}
		keys = append(keys, key{
			name:     k,
			priority: priority,
		})
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].priority == keys[j].priority {
			return keys[i].name < keys[j].name
		}
		return keys[i].priority > keys[j].priority
	})
	var sorted []string
	for _, k := range keys {
		sorted = append(sorted, k.name)
	}
	return sorted
}

func (s *ServiceConfig) GetScale() int {
	if s.Scale != nil {
		return *s.Scale
	}
	if s.Deploy != nil && s.Deploy.Replicas != nil {
		// this should not be required as compose-go enforce consistency between scale anr replicas
		return *s.Deploy.Replicas
	}
	return 1
}

func (s *ServiceConfig) SetScale(scale int) {
	s.Scale = &scale
	if s.Deploy != nil {
		s.Deploy.Replicas = &scale
	}
}

func (s *ServiceConfig) deepCopy() *ServiceConfig {
	if s == nil {
		return nil
	}
	n := &ServiceConfig{}
	deriveDeepCopyService(n, s)
	return n
}

const (
	// PullPolicyAlways always pull images
	PullPolicyAlways = "always"
	// PullPolicyNever never pull images
	PullPolicyNever = "never"
	// PullPolicyIfNotPresent pull missing images
	PullPolicyIfNotPresent = "if_not_present"
	// PullPolicyMissing pull missing images
	PullPolicyMissing = "missing"
	// PullPolicyBuild force building images
	PullPolicyBuild = "build"
)

const (
	// RestartPolicyAlways always restart the container if it stops
	RestartPolicyAlways = "always"
	// RestartPolicyOnFailure restart the container if it exits due to an error
	RestartPolicyOnFailure = "on-failure"
	// RestartPolicyNo do not automatically restart the container
	RestartPolicyNo = "no"
	// RestartPolicyUnlessStopped always restart the container unless the container is stopped (manually or otherwise)
	RestartPolicyUnlessStopped = "unless-stopped"
)

const (
	// ServicePrefix is the prefix for references pointing to a service
	ServicePrefix = "service:"
	// ContainerPrefix is the prefix for references pointing to a container
	ContainerPrefix = "container:"

	// NetworkModeServicePrefix is the prefix for network_mode pointing to a service
	// Deprecated prefer ServicePrefix
	NetworkModeServicePrefix = ServicePrefix
	// NetworkModeContainerPrefix is the prefix for network_mode pointing to a container
	// Deprecated prefer ContainerPrefix
	NetworkModeContainerPrefix = ContainerPrefix
)

const (
	SecretConfigXValue = "x-#value"
)

// GetDependencies retrieves all services this service depends on
func (s ServiceConfig) GetDependencies() []string {
	var dependencies []string
	for service := range s.DependsOn {
		dependencies = append(dependencies, service)
	}
	return dependencies
}

// GetDependents retrieves all services which depend on this service
func (s ServiceConfig) GetDependents(p *Project) []string {
	var dependent []string
	for _, service := range p.Services {
		for name := range service.DependsOn {
			if name == s.Name {
				dependent = append(dependent, service.Name)
			}
		}
	}
	return dependent
}

// BuildConfig is a type for build
type BuildConfig struct {
	Context            string                    `yaml:"context,omitempty" json:"context,omitempty"`
	Dockerfile         string                    `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	DockerfileInline   string                    `yaml:"dockerfile_inline,omitempty" json:"dockerfile_inline,omitempty"`
	Entitlements       []string                  `yaml:"entitlements,omitempty" json:"entitlements,omitempty"`
	Args               MappingWithEquals         `yaml:"args,omitempty" json:"args,omitempty"`
	SSH                SSHConfig                 `yaml:"ssh,omitempty" json:"ssh,omitempty"`
	Labels             Labels                    `yaml:"labels,omitempty" json:"labels,omitempty"`
	CacheFrom          StringList                `yaml:"cache_from,omitempty" json:"cache_from,omitempty"`
	CacheTo            StringList                `yaml:"cache_to,omitempty" json:"cache_to,omitempty"`
	NoCache            bool                      `yaml:"no_cache,omitempty" json:"no_cache,omitempty"`
	AdditionalContexts Mapping                   `yaml:"additional_contexts,omitempty" json:"additional_contexts,omitempty"`
	Pull               bool                      `yaml:"pull,omitempty" json:"pull,omitempty"`
	ExtraHosts         HostsList                 `yaml:"extra_hosts,omitempty" json:"extra_hosts,omitempty"`
	Isolation          string                    `yaml:"isolation,omitempty" json:"isolation,omitempty"`
	Network            string                    `yaml:"network,omitempty" json:"network,omitempty"`
	Target             string                    `yaml:"target,omitempty" json:"target,omitempty"`
	Secrets            []ServiceSecretConfig     `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	ShmSize            UnitBytes                 `yaml:"shm_size,omitempty" json:"shm_size,omitempty"`
	Tags               StringList                `yaml:"tags,omitempty" json:"tags,omitempty"`
	Ulimits            map[string]*UlimitsConfig `yaml:"ulimits,omitempty" json:"ulimits,omitempty"`
	Platforms          StringList                `yaml:"platforms,omitempty" json:"platforms,omitempty"`
	Privileged         bool                      `yaml:"privileged,omitempty" json:"privileged,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// BlkioConfig define blkio config
type BlkioConfig struct {
	Weight          uint16           `yaml:"weight,omitempty" json:"weight,omitempty"`
	WeightDevice    []WeightDevice   `yaml:"weight_device,omitempty" json:"weight_device,omitempty"`
	DeviceReadBps   []ThrottleDevice `yaml:"device_read_bps,omitempty" json:"device_read_bps,omitempty"`
	DeviceReadIOps  []ThrottleDevice `yaml:"device_read_iops,omitempty" json:"device_read_iops,omitempty"`
	DeviceWriteBps  []ThrottleDevice `yaml:"device_write_bps,omitempty" json:"device_write_bps,omitempty"`
	DeviceWriteIOps []ThrottleDevice `yaml:"device_write_iops,omitempty" json:"device_write_iops,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

type DeviceMapping struct {
	Source      string `yaml:"source,omitempty" json:"source,omitempty"`
	Target      string `yaml:"target,omitempty" json:"target,omitempty"`
	Permissions string `yaml:"permissions,omitempty" json:"permissions,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// WeightDevice is a structure that holds device:weight pair
type WeightDevice struct {
	Path   string
	Weight uint16

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ThrottleDevice is a structure that holds device:rate_per_second pair
type ThrottleDevice struct {
	Path string
	Rate UnitBytes

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// MappingWithColon is a mapping type that can be converted from a list of
// 'key: value' strings
type MappingWithColon map[string]string

// LoggingConfig the logging configuration for a service
type LoggingConfig struct {
	Driver  string  `yaml:"driver,omitempty" json:"driver,omitempty"`
	Options Options `yaml:"options,omitempty" json:"options,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// DeployConfig the deployment configuration for a service
type DeployConfig struct {
	Mode           string         `yaml:"mode,omitempty" json:"mode,omitempty"`
	Replicas       *int           `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Labels         Labels         `yaml:"labels,omitempty" json:"labels,omitempty"`
	UpdateConfig   *UpdateConfig  `yaml:"update_config,omitempty" json:"update_config,omitempty"`
	RollbackConfig *UpdateConfig  `yaml:"rollback_config,omitempty" json:"rollback_config,omitempty"`
	Resources      Resources      `yaml:"resources,omitempty" json:"resources,omitempty"`
	RestartPolicy  *RestartPolicy `yaml:"restart_policy,omitempty" json:"restart_policy,omitempty"`
	Placement      Placement      `yaml:"placement,omitempty" json:"placement,omitempty"`
	EndpointMode   string         `yaml:"endpoint_mode,omitempty" json:"endpoint_mode,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// UpdateConfig the service update configuration
type UpdateConfig struct {
	Parallelism     *uint64  `yaml:"parallelism,omitempty" json:"parallelism,omitempty"`
	Delay           Duration `yaml:"delay,omitempty" json:"delay,omitempty"`
	FailureAction   string   `yaml:"failure_action,omitempty" json:"failure_action,omitempty"`
	Monitor         Duration `yaml:"monitor,omitempty" json:"monitor,omitempty"`
	MaxFailureRatio float32  `yaml:"max_failure_ratio,omitempty" json:"max_failure_ratio,omitempty"`
	Order           string   `yaml:"order,omitempty" json:"order,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// Resources the resource limits and reservations
type Resources struct {
	Limits       *Resource `yaml:"limits,omitempty" json:"limits,omitempty"`
	Reservations *Resource `yaml:"reservations,omitempty" json:"reservations,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// Resource is a resource to be limited or reserved
type Resource struct {
	// TODO: types to convert from units and ratios
	NanoCPUs         NanoCPUs          `yaml:"cpus,omitempty" json:"cpus,omitempty"`
	MemoryBytes      UnitBytes         `yaml:"memory,omitempty" json:"memory,omitempty"`
	Pids             int64             `yaml:"pids,omitempty" json:"pids,omitempty"`
	Devices          []DeviceRequest   `yaml:"devices,omitempty" json:"devices,omitempty"`
	GenericResources []GenericResource `yaml:"generic_resources,omitempty" json:"generic_resources,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

type NanoCPUs float32

func (n *NanoCPUs) DecodeMapstructure(a any) error {
	switch v := a.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		*n = NanoCPUs(f)
	case float32:
		*n = NanoCPUs(v)
	case float64:
		*n = NanoCPUs(v)
	default:
		return fmt.Errorf("unexpected value type %T for cpus", v)
	}
	return nil
}

func (n *NanoCPUs) Value() float32 {
	return float32(*n)
}

// GenericResource represents a "user defined" resource which can
// only be an integer (e.g: SSD=3) for a service
type GenericResource struct {
	DiscreteResourceSpec *DiscreteGenericResource `yaml:"discrete_resource_spec,omitempty" json:"discrete_resource_spec,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// DiscreteGenericResource represents a "user defined" resource which is defined
// as an integer
// "Kind" is used to describe the Kind of a resource (e.g: "GPU", "FPGA", "SSD", ...)
// Value is used to count the resource (SSD=5, HDD=3, ...)
type DiscreteGenericResource struct {
	Kind  string `json:"kind"`
	Value int64  `json:"value"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// RestartPolicy the service restart policy
type RestartPolicy struct {
	Condition   string    `yaml:"condition,omitempty" json:"condition,omitempty"`
	Delay       *Duration `yaml:"delay,omitempty" json:"delay,omitempty"`
	MaxAttempts *uint64   `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	Window      *Duration `yaml:"window,omitempty" json:"window,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// Placement constraints for the service
type Placement struct {
	Constraints []string               `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	Preferences []PlacementPreferences `yaml:"preferences,omitempty" json:"preferences,omitempty"`
	MaxReplicas uint64                 `yaml:"max_replicas_per_node,omitempty" json:"max_replicas_per_node,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// PlacementPreferences is the preferences for a service placement
type PlacementPreferences struct {
	Spread string `yaml:"spread,omitempty" json:"spread,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ServiceNetworkConfig is the network configuration for a service
type ServiceNetworkConfig struct {
	Priority     int      `yaml:"priority,omitempty" json:"priority,omitempty"`
	Aliases      []string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Ipv4Address  string   `yaml:"ipv4_address,omitempty" json:"ipv4_address,omitempty"`
	Ipv6Address  string   `yaml:"ipv6_address,omitempty" json:"ipv6_address,omitempty"`
	LinkLocalIPs []string `yaml:"link_local_ips,omitempty" json:"link_local_ips,omitempty"`
	MacAddress   string   `yaml:"mac_address,omitempty" json:"mac_address,omitempty"`
	DriverOpts   Options  `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ServicePortConfig is the port configuration for a service
type ServicePortConfig struct {
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`
	Mode        string `yaml:"mode,omitempty" json:"mode,omitempty"`
	HostIP      string `yaml:"host_ip,omitempty" json:"host_ip,omitempty"`
	Target      uint32 `yaml:"target,omitempty" json:"target,omitempty"`
	Published   string `yaml:"published,omitempty" json:"published,omitempty"`
	Protocol    string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	AppProtocol string `yaml:"app_protocol,omitempty" json:"app_protocol,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ParsePortConfig parse short syntax for service port configuration
func ParsePortConfig(value string) ([]ServicePortConfig, error) {
	var portConfigs []ServicePortConfig
	ports, portBindings, err := nat.ParsePortSpecs([]string{value})
	if err != nil {
		return nil, err
	}
	// We need to sort the key of the ports to make sure it is consistent
	keys := []string{}
	for port := range ports {
		keys = append(keys, string(port))
	}
	sort.Strings(keys)

	for _, key := range keys {
		port := nat.Port(key)
		converted, err := convertPortToPortConfig(port, portBindings)
		if err != nil {
			return nil, err
		}
		portConfigs = append(portConfigs, converted...)
	}
	return portConfigs, nil
}

func convertPortToPortConfig(port nat.Port, portBindings map[nat.Port][]nat.PortBinding) ([]ServicePortConfig, error) {
	var portConfigs []ServicePortConfig
	for _, binding := range portBindings[port] {
		portConfigs = append(portConfigs, ServicePortConfig{
			HostIP:    binding.HostIP,
			Protocol:  strings.ToLower(port.Proto()),
			Target:    uint32(port.Int()),
			Published: binding.HostPort,
			Mode:      "ingress",
		})
	}
	return portConfigs, nil
}

// ServiceVolumeConfig are references to a volume used by a service
type ServiceVolumeConfig struct {
	Type        string               `yaml:"type,omitempty" json:"type,omitempty"`
	Source      string               `yaml:"source,omitempty" json:"source,omitempty"`
	Target      string               `yaml:"target,omitempty" json:"target,omitempty"`
	ReadOnly    bool                 `yaml:"read_only,omitempty" json:"read_only,omitempty"`
	Consistency string               `yaml:"consistency,omitempty" json:"consistency,omitempty"`
	Bind        *ServiceVolumeBind   `yaml:"bind,omitempty" json:"bind,omitempty"`
	Volume      *ServiceVolumeVolume `yaml:"volume,omitempty" json:"volume,omitempty"`
	Tmpfs       *ServiceVolumeTmpfs  `yaml:"tmpfs,omitempty" json:"tmpfs,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// String render ServiceVolumeConfig as a volume string, one can parse back using loader.ParseVolume
func (s ServiceVolumeConfig) String() string {
	access := "rw"
	if s.ReadOnly {
		access = "ro"
	}
	options := []string{access}
	if s.Bind != nil && s.Bind.SELinux != "" {
		options = append(options, s.Bind.SELinux)
	}
	if s.Bind != nil && s.Bind.Propagation != "" {
		options = append(options, s.Bind.Propagation)
	}
	if s.Volume != nil && s.Volume.NoCopy {
		options = append(options, "nocopy")
	}
	if s.Volume != nil && s.Volume.Subpath != "" {
		options = append(options, s.Volume.Subpath)
	}
	return fmt.Sprintf("%s:%s:%s", s.Source, s.Target, strings.Join(options, ","))
}

const (
	// VolumeTypeBind is the type for mounting host dir
	VolumeTypeBind = "bind"
	// VolumeTypeVolume is the type for remote storage volumes
	VolumeTypeVolume = "volume"
	// VolumeTypeTmpfs is the type for mounting tmpfs
	VolumeTypeTmpfs = "tmpfs"
	// VolumeTypeNamedPipe is the type for mounting Windows named pipes
	VolumeTypeNamedPipe = "npipe"
	// VolumeTypeCluster is the type for mounting container storage interface (CSI) volumes
	VolumeTypeCluster = "cluster"

	// SElinuxShared share the volume content
	SElinuxShared = "z"
	// SElinuxUnshared label content as private unshared
	SElinuxUnshared = "Z"
)

// ServiceVolumeBind are options for a service volume of type bind
type ServiceVolumeBind struct {
	SELinux        string `yaml:"selinux,omitempty" json:"selinux,omitempty"`
	Propagation    string `yaml:"propagation,omitempty" json:"propagation,omitempty"`
	CreateHostPath bool   `yaml:"create_host_path,omitempty" json:"create_host_path,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// SELinux represents the SELinux re-labeling options.
const (
	// SELinuxShared option indicates that the bind mount content is shared among multiple containers
	SELinuxShared string = "z"
	// SELinuxPrivate option indicates that the bind mount content is private and unshared
	SELinuxPrivate string = "Z"
)

// Propagation represents the propagation of a mount.
const (
	// PropagationRPrivate RPRIVATE
	PropagationRPrivate string = "rprivate"
	// PropagationPrivate PRIVATE
	PropagationPrivate string = "private"
	// PropagationRShared RSHARED
	PropagationRShared string = "rshared"
	// PropagationShared SHARED
	PropagationShared string = "shared"
	// PropagationRSlave RSLAVE
	PropagationRSlave string = "rslave"
	// PropagationSlave SLAVE
	PropagationSlave string = "slave"
)

// ServiceVolumeVolume are options for a service volume of type volume
type ServiceVolumeVolume struct {
	NoCopy  bool   `yaml:"nocopy,omitempty" json:"nocopy,omitempty"`
	Subpath string `yaml:"subpath,omitempty" json:"subpath,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ServiceVolumeTmpfs are options for a service volume of type tmpfs
type ServiceVolumeTmpfs struct {
	Size UnitBytes `yaml:"size,omitempty" json:"size,omitempty"`

	Mode uint32 `yaml:"mode,omitempty" json:"mode,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// FileReferenceConfig for a reference to a swarm file object
type FileReferenceConfig struct {
	Source string  `yaml:"source,omitempty" json:"source,omitempty"`
	Target string  `yaml:"target,omitempty" json:"target,omitempty"`
	UID    string  `yaml:"uid,omitempty" json:"uid,omitempty"`
	GID    string  `yaml:"gid,omitempty" json:"gid,omitempty"`
	Mode   *uint32 `yaml:"mode,omitempty" json:"mode,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ServiceConfigObjConfig is the config obj configuration for a service
type ServiceConfigObjConfig FileReferenceConfig

// ServiceSecretConfig is the secret configuration for a service
type ServiceSecretConfig FileReferenceConfig

// UlimitsConfig the ulimit configuration
type UlimitsConfig struct {
	Single int `yaml:"single,omitempty" json:"single,omitempty"`
	Soft   int `yaml:"soft,omitempty" json:"soft,omitempty"`
	Hard   int `yaml:"hard,omitempty" json:"hard,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

func (u *UlimitsConfig) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case *UlimitsConfig:
		// this call to DecodeMapstructure is triggered after initial value conversion as we use a map[string]*UlimitsConfig
		return nil
	case int:
		u.Single = v
		u.Soft = 0
		u.Hard = 0
	case map[string]any:
		u.Single = 0
		soft, ok := v["soft"]
		if ok {
			u.Soft = soft.(int)
		}
		hard, ok := v["hard"]
		if ok {
			u.Hard = hard.(int)
		}
	default:
		return fmt.Errorf("unexpected value type %T for ulimit", value)
	}
	return nil
}

// MarshalYAML makes UlimitsConfig implement yaml.Marshaller
func (u *UlimitsConfig) MarshalYAML() (interface{}, error) {
	if u.Single != 0 {
		return u.Single, nil
	}
	return struct {
		Soft int
		Hard int
	}{
		Soft: u.Soft,
		Hard: u.Hard,
	}, nil
}

// MarshalJSON makes UlimitsConfig implement json.Marshaller
func (u *UlimitsConfig) MarshalJSON() ([]byte, error) {
	if u.Single != 0 {
		return json.Marshal(u.Single)
	}
	// Pass as a value to avoid re-entering this method and use the default implementation
	return json.Marshal(*u)
}

// NetworkConfig for a network
type NetworkConfig struct {
	Name       string     `yaml:"name,omitempty" json:"name,omitempty"`
	Driver     string     `yaml:"driver,omitempty" json:"driver,omitempty"`
	DriverOpts Options    `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"`
	Ipam       IPAMConfig `yaml:"ipam,omitempty" json:"ipam,omitempty"`
	External   External   `yaml:"external,omitempty" json:"external,omitempty"`
	Internal   bool       `yaml:"internal,omitempty" json:"internal,omitempty"`
	Attachable bool       `yaml:"attachable,omitempty" json:"attachable,omitempty"`
	Labels     Labels     `yaml:"labels,omitempty" json:"labels,omitempty"`
	EnableIPv6 *bool      `yaml:"enable_ipv6,omitempty" json:"enable_ipv6,omitempty"`
	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// IPAMConfig for a network
type IPAMConfig struct {
	Driver     string      `yaml:"driver,omitempty" json:"driver,omitempty"`
	Config     []*IPAMPool `yaml:"config,omitempty" json:"config,omitempty"`
	Extensions Extensions  `yaml:"#extensions,inline,omitempty" json:"-"`
}

// IPAMPool for a network
type IPAMPool struct {
	Subnet             string     `yaml:"subnet,omitempty" json:"subnet,omitempty"`
	Gateway            string     `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	IPRange            string     `yaml:"ip_range,omitempty" json:"ip_range,omitempty"`
	AuxiliaryAddresses Mapping    `yaml:"aux_addresses,omitempty" json:"aux_addresses,omitempty"`
	Extensions         Extensions `yaml:",inline" json:"-"`
}

// VolumeConfig for a volume
type VolumeConfig struct {
	Name       string     `yaml:"name,omitempty" json:"name,omitempty"`
	Driver     string     `yaml:"driver,omitempty" json:"driver,omitempty"`
	DriverOpts Options    `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"`
	External   External   `yaml:"external,omitempty" json:"external,omitempty"`
	Labels     Labels     `yaml:"labels,omitempty" json:"labels,omitempty"`
	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// External identifies a Volume or Network as a reference to a resource that is
// not managed, and should already exist.
type External bool

// CredentialSpecConfig for credential spec on Windows
type CredentialSpecConfig struct {
	Config     string     `yaml:"config,omitempty" json:"config,omitempty"` // Config was added in API v1.40
	File       string     `yaml:"file,omitempty" json:"file,omitempty"`
	Registry   string     `yaml:"registry,omitempty" json:"registry,omitempty"`
	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// FileObjectConfig is a config type for a file used by a service
type FileObjectConfig struct {
	Name           string            `yaml:"name,omitempty" json:"name,omitempty"`
	File           string            `yaml:"file,omitempty" json:"file,omitempty"`
	Environment    string            `yaml:"environment,omitempty" json:"environment,omitempty"`
	Content        string            `yaml:"content,omitempty" json:"content,omitempty"`
	External       External          `yaml:"external,omitempty" json:"external,omitempty"`
	Labels         Labels            `yaml:"labels,omitempty" json:"labels,omitempty"`
	Driver         string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	DriverOpts     map[string]string `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"`
	TemplateDriver string            `yaml:"template_driver,omitempty" json:"template_driver,omitempty"`
	Extensions     Extensions        `yaml:"#extensions,inline,omitempty" json:"-"`
}

const (
	// ServiceConditionCompletedSuccessfully is the type for waiting until a service has completed successfully (exit code 0).
	ServiceConditionCompletedSuccessfully = "service_completed_successfully"

	// ServiceConditionHealthy is the type for waiting until a service is healthy.
	ServiceConditionHealthy = "service_healthy"

	// ServiceConditionStarted is the type for waiting until a service has started (default).
	ServiceConditionStarted = "service_started"
)

type DependsOnConfig map[string]ServiceDependency

type ServiceDependency struct {
	Condition  string     `yaml:"condition,omitempty" json:"condition,omitempty"`
	Restart    bool       `yaml:"restart,omitempty" json:"restart,omitempty"`
	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
	Required   bool       `yaml:"required" json:"required"`
}

type ExtendsConfig struct {
	File    string `yaml:"file,omitempty" json:"file,omitempty"`
	Service string `yaml:"service,omitempty" json:"service,omitempty"`
}

// SecretConfig for a secret
type SecretConfig FileObjectConfig

// MarshalYAML makes SecretConfig implement yaml.Marshaller
func (s SecretConfig) MarshalYAML() (interface{}, error) {
	// secret content is set while loading model. Never marshall it
	s.Content = ""
	return FileObjectConfig(s), nil
}

// MarshalJSON makes SecretConfig implement json.Marshaller
func (s SecretConfig) MarshalJSON() ([]byte, error) {
	// secret content is set while loading model. Never marshall it
	s.Content = ""
	return json.Marshal(FileObjectConfig(s))
}

// ConfigObjConfig is the config for the swarm "Config" object
type ConfigObjConfig FileObjectConfig

// MarshalYAML makes ConfigObjConfig implement yaml.Marshaller
func (s ConfigObjConfig) MarshalYAML() (interface{}, error) {
	// config content may have been set from environment while loading model. Marshall actual source
	if s.Environment != "" {
		s.Content = ""
	}
	return FileObjectConfig(s), nil
}

// MarshalJSON makes ConfigObjConfig implement json.Marshaller
func (s ConfigObjConfig) MarshalJSON() ([]byte, error) {
	// config content may have been set from environment while loading model. Marshall actual source
	if s.Environment != "" {
		s.Content = ""
	}
	return json.Marshal(FileObjectConfig(s))
}

type IncludeConfig struct {
	Path             StringList `yaml:"path,omitempty" json:"path,omitempty"`
	ProjectDirectory string     `yaml:"project_directory,omitempty" json:"project_directory,omitempty"`
	EnvFile          StringList `yaml:"env_file,omitempty" json:"env_file,omitempty"`
}
