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
	"reflect"
	"strconv"
)

// comparable to yaml.Unmarshaler, decoder allow a type to define it's own custom logic to convert value
// see https://github.com/mitchellh/mapstructure/pull/294
type decoder interface {
	DecodeMapstructure(interface{}) error
}

// see https://github.com/mitchellh/mapstructure/issues/115#issuecomment-735287466
// adapted to support types derived from built-in types, as DecodeMapstructure would not be able to mutate internal
// value, so need to invoke DecodeMapstructure defined by pointer to type
func decoderHook(from reflect.Value, to reflect.Value) (interface{}, error) {
	// If the destination implements the decoder interface
	u, ok := to.Interface().(decoder)
	if !ok {
		// for non-struct types we need to invoke func (*type) DecodeMapstructure()
		if to.CanAddr() {
			pto := to.Addr()
			u, ok = pto.Interface().(decoder)
		}
		if !ok {
			return from.Interface(), nil
		}
	}
	// If it is nil and a pointer, create and assign the target value first
	if to.Type().Kind() == reflect.Ptr && to.IsNil() {
		to.Set(reflect.New(to.Type().Elem()))
		u = to.Interface().(decoder)
	}
	// Call the custom DecodeMapstructure method
	if err := u.DecodeMapstructure(from.Interface()); err != nil {
		return to.Interface(), err
	}
	return to.Interface(), nil
}

func cast(from reflect.Value, to reflect.Value) (interface{}, error) {
	switch from.Type().Kind() {
	case reflect.String:
		switch to.Kind() {
		case reflect.Bool:
			return toBoolean(from.String())
		case reflect.Int:
			return toInt(from.String())
		case reflect.Int64:
			return toInt64(from.String())
		case reflect.Float32:
			return toFloat32(from.String())
		case reflect.Float64:
			return toFloat(from.String())
		}
	case reflect.Int:
		if to.Kind() == reflect.String {
			return strconv.FormatInt(from.Int(), 10), nil
		}
	}
	return from.Interface(), nil
}
