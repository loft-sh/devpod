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
	"strconv"
	"strings"

	interp "github.com/compose-spec/compose-go/v2/interpolation"
	"github.com/compose-spec/compose-go/v2/tree"
	"github.com/sirupsen/logrus"
)

var interpolateTypeCastMapping = map[tree.Path]interp.Cast{
	servicePath("configs", tree.PathMatchList, "mode"):             toInt,
	servicePath("cpu_count"):                                       toInt64,
	servicePath("cpu_percent"):                                     toFloat,
	servicePath("cpu_period"):                                      toInt64,
	servicePath("cpu_quota"):                                       toInt64,
	servicePath("cpu_rt_period"):                                   toInt64,
	servicePath("cpu_rt_runtime"):                                  toInt64,
	servicePath("cpus"):                                            toFloat32,
	servicePath("cpu_shares"):                                      toInt64,
	servicePath("init"):                                            toBoolean,
	servicePath("deploy", "replicas"):                              toInt,
	servicePath("deploy", "update_config", "parallelism"):          toInt,
	servicePath("deploy", "update_config", "max_failure_ratio"):    toFloat,
	servicePath("deploy", "rollback_config", "parallelism"):        toInt,
	servicePath("deploy", "rollback_config", "max_failure_ratio"):  toFloat,
	servicePath("deploy", "restart_policy", "max_attempts"):        toInt,
	servicePath("deploy", "placement", "max_replicas_per_node"):    toInt,
	servicePath("healthcheck", "retries"):                          toInt,
	servicePath("healthcheck", "disable"):                          toBoolean,
	servicePath("oom_kill_disable"):                                toBoolean,
	servicePath("oom_score_adj"):                                   toInt64,
	servicePath("pids_limit"):                                      toInt64,
	servicePath("ports", tree.PathMatchList, "target"):             toInt,
	servicePath("privileged"):                                      toBoolean,
	servicePath("read_only"):                                       toBoolean,
	servicePath("scale"):                                           toInt,
	servicePath("secrets", tree.PathMatchList, "mode"):             toInt,
	servicePath("stdin_open"):                                      toBoolean,
	servicePath("tty"):                                             toBoolean,
	servicePath("ulimits", tree.PathMatchAll):                      toInt,
	servicePath("ulimits", tree.PathMatchAll, "hard"):              toInt,
	servicePath("ulimits", tree.PathMatchAll, "soft"):              toInt,
	servicePath("volumes", tree.PathMatchList, "read_only"):        toBoolean,
	servicePath("volumes", tree.PathMatchList, "volume", "nocopy"): toBoolean,
	iPath("networks", tree.PathMatchAll, "external"):               toBoolean,
	iPath("networks", tree.PathMatchAll, "internal"):               toBoolean,
	iPath("networks", tree.PathMatchAll, "attachable"):             toBoolean,
	iPath("networks", tree.PathMatchAll, "enable_ipv6"):            toBoolean,
	iPath("volumes", tree.PathMatchAll, "external"):                toBoolean,
	iPath("secrets", tree.PathMatchAll, "external"):                toBoolean,
	iPath("configs", tree.PathMatchAll, "external"):                toBoolean,
}

func iPath(parts ...string) tree.Path {
	return tree.NewPath(parts...)
}

func servicePath(parts ...string) tree.Path {
	return iPath(append([]string{"services", tree.PathMatchAll}, parts...)...)
}

func toInt(value string) (interface{}, error) {
	return strconv.Atoi(value)
}

func toInt64(value string) (interface{}, error) {
	return strconv.ParseInt(value, 10, 64)
}

func toFloat(value string) (interface{}, error) {
	return strconv.ParseFloat(value, 64)
}

func toFloat32(value string) (interface{}, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return nil, err
	}
	return float32(f), nil
}

// should match http://yaml.org/type/bool.html
func toBoolean(value string) (interface{}, error) {
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "y", "yes", "on":
		logrus.Warnf("%q for boolean is not supported by YAML 1.2, please use `true`", value)
		return true, nil
	case "n", "no", "off":
		logrus.Warnf("%q for boolean is not supported by YAML 1.2, please use `false`", value)
		return false, nil
	default:
		return nil, fmt.Errorf("invalid boolean: %s", value)
	}
}
