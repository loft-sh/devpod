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

package paths

import (
	"path"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/utils"
)

func (r *relativePathsResolver) maybeUnixPath(a any) (any, error) {
	p := a.(string)
	p = ExpandUser(p)
	// Check if source is an absolute path (either Unix or Windows), to
	// handle a Windows client with a Unix daemon or vice-versa.
	//
	// Note that this is not required for Docker for Windows when specifying
	// a local Windows path, because Docker for Windows translates the Windows
	// path into a valid path within the VM.
	if !path.IsAbs(p) && !isWindowsAbs(p) {
		if filepath.IsAbs(p) {
			return p, nil
		}
		return filepath.Join(r.workingDir, p), nil
	}
	return p, nil
}

func (r *relativePathsResolver) absSymbolicLink(value any) (any, error) {
	abs, err := r.absPath(value)
	if err != nil {
		return nil, err
	}
	str, ok := abs.(string)
	if !ok {
		return abs, nil
	}
	return utils.ResolveSymbolicLink(str)
}
