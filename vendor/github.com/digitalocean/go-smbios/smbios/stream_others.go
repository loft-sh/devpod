// Copyright 2017-2018 DigitalOcean.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//+build !dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris,!windows

package smbios

import (
	"fmt"
	"io"
	"runtime"
)

// stream is not implemented for unsupported platforms.
func stream() (io.ReadCloser, EntryPoint, error) {
	return nil, nil, fmt.Errorf("opening SMBIOS stream not implemented on %q", runtime.GOOS)
}
