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
	"reflect"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/tree"
	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

type ResetProcessor struct {
	target interface{}
	paths  []tree.Path
}

// UnmarshalYAML implement yaml.Unmarshaler
func (p *ResetProcessor) UnmarshalYAML(value *yaml.Node) error {
	resolved, err := p.resolveReset(value, tree.NewPath())
	if err != nil {
		return err
	}
	return resolved.Decode(p.target)
}

// resolveReset detects `!reset` tag being set on yaml nodes and record position in the yaml tree
func (p *ResetProcessor) resolveReset(node *yaml.Node, path tree.Path) (*yaml.Node, error) {
	if node.Tag == "!reset" {
		p.paths = append(p.paths, path)
	}
	switch node.Kind {
	case yaml.SequenceNode:
		var err error
		for idx, v := range node.Content {
			next := path.Next(strconv.Itoa(idx))
			node.Content[idx], err = p.resolveReset(v, next)
			if err != nil {
				return nil, err
			}
		}
	case yaml.MappingNode:
		var err error
		var key string
		for idx, v := range node.Content {
			if idx%2 == 0 {
				key = v.Value
			} else {
				node.Content[idx], err = p.resolveReset(v, path.Next(key))
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return node, nil
}

// Apply finds the go attributes matching recorded paths and reset them to zero value
func (p *ResetProcessor) Apply(target *types.Config) error {
	return p.applyNullOverrides(reflect.ValueOf(target), tree.NewPath())
}

// applyNullOverrides set val to Zero if it matches any of the recorded paths
func (p *ResetProcessor) applyNullOverrides(val reflect.Value, path tree.Path) error {
	val = reflect.Indirect(val)
	if !val.IsValid() {
		return nil
	}
	typ := val.Type()
	switch {
	case path == "services":
		// Project.Services is a slice in compose-go, but a mapping in yaml
		for i := 0; i < val.Len(); i++ {
			service := val.Index(i)
			name := service.FieldByName("Name")
			next := path.Next(name.String())
			err := p.applyNullOverrides(service, next)
			if err != nil {
				return err
			}
		}
	case typ.Kind() == reflect.Map:
		iter := val.MapRange()
	KEYS:
		for iter.Next() {
			k := iter.Key()
			next := path.Next(k.String())
			for _, pattern := range p.paths {
				if next.Matches(pattern) {
					val.SetMapIndex(k, reflect.Value{})
					continue KEYS
				}
			}
			return p.applyNullOverrides(iter.Value(), next)
		}
	case typ.Kind() == reflect.Slice:
	ITER:
		for i := 0; i < val.Len(); i++ {
			next := path.Next(fmt.Sprintf("[%d]", i))
			for _, pattern := range p.paths {
				if next.Matches(pattern) {

					continue ITER
				}
			}
			// TODO(ndeloof) support removal from sequence
			return p.applyNullOverrides(val.Index(i), next)
		}

	case typ.Kind() == reflect.Struct:
	FIELDS:
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name := field.Name
			attr := strings.ToLower(name)
			tag := field.Tag.Get("yaml")
			tag = strings.Split(tag, ",")[0]
			if tag != "" && tag != "-" {
				attr = tag
			}
			next := path.Next(attr)
			f := val.Field(i)
			for _, pattern := range p.paths {
				if next.Matches(pattern) {
					f := f
					if !f.CanSet() {
						return fmt.Errorf("can't override attribute %s", name)
					}
					// f.SetZero() requires go 1.20
					f.Set(reflect.Zero(f.Type()))
					continue FIELDS
				}
			}
			err := p.applyNullOverrides(f, next)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
