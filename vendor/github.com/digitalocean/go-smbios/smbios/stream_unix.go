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

//+build dragonfly freebsd netbsd openbsd solaris

// Linux intentionally omitted because it has an alternative method that
// is used before attempting /dev/mem access.  See stream_linux.go.

package smbios

import (
	"io"
)

// stream opens the SMBIOS entry point and an SMBIOS structure stream.
func stream() (io.ReadCloser, EntryPoint, error) {
	// Use the standard UNIX-like system method.
	return devMemStream()
}
