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
	"strings"
)

// HostsList is a list of colon-separated host-ip mappings
type HostsList map[string][]string

// NewHostsList creates a HostsList from a list of `host=ip` strings
func NewHostsList(hosts []string) (HostsList, error) {
	list := HostsList{}
	for _, s := range hosts {
		var found bool
		for _, sep := range hostListSerapators {
			host, ip, ok := strings.Cut(s, sep)
			if ok {
				// Mapping found with this separator, stop here.
				if ips, ok := list[host]; ok {
					list[host] = append(ips, strings.Split(ip, ",")...)
				} else {
					list[host] = strings.Split(ip, ",")
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("invalid additional host, missing IP: %s", s)
		}
	}
	err := list.cleanup()
	return list, err
}

// AsList returns host-ip mappings as a list of strings, using the given
// separator. The Docker Engine API expects ':' separators, the original format
// for '--add-hosts'. But an '=' separator is used in YAML/JSON renderings to
// make IPv6 addresses more readable (for example "my-host=::1" instead of
// "my-host:::1").
func (h HostsList) AsList(sep string) []string {
	l := make([]string, 0, len(h))
	for k, v := range h {
		for _, ip := range v {
			l = append(l, fmt.Sprintf("%s%s%s", k, sep, ip))
		}
	}
	return l
}

func (h HostsList) MarshalYAML() (interface{}, error) {
	list := h.AsList("=")
	sort.Strings(list)
	return list, nil
}

func (h HostsList) MarshalJSON() ([]byte, error) {
	list := h.AsList("=")
	sort.Strings(list)
	return json.Marshal(list)
}

var hostListSerapators = []string{"=", ":"}

func (h *HostsList) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		list := make(HostsList, len(v))
		for i, e := range v {
			if e == nil {
				e = ""
			}
			switch t := e.(type) {
			case string:
				list[i] = []string{t}
			case []any:
				hosts := make([]string, len(t))
				for j, h := range t {
					hosts[j] = fmt.Sprint(h)
				}
				list[i] = hosts
			default:
				return fmt.Errorf("unexpected value type %T for extra_hosts entry", value)
			}
		}
		err := list.cleanup()
		if err != nil {
			return err
		}
		*h = list
		return nil
	case []interface{}:
		s := make([]string, len(v))
		for i, e := range v {
			s[i] = fmt.Sprint(e)
		}
		list, err := NewHostsList(s)
		if err != nil {
			return err
		}
		*h = list
		return nil
	default:
		return fmt.Errorf("unexpected value type %T for extra_hosts", value)
	}
}

func (h HostsList) cleanup() error {
	for host, ips := range h {
		// Check that there is a hostname and that it doesn't contain either
		// of the allowed separators, to generate a clearer error than the
		// engine would do if it splits the string differently.
		if host == "" || strings.ContainsAny(host, ":=") {
			return fmt.Errorf("bad host name '%s'", host)
		}
		for i, ip := range ips {
			// Remove brackets from IP addresses (for example "[::1]" -> "::1").
			if len(ip) > 2 && ip[0] == '[' && ip[len(ip)-1] == ']' {
				ips[i] = ip[1 : len(ip)-1]
			}
		}
		h[host] = ips
	}
	return nil
}
