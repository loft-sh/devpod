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

import "strings"

func (r *relativePathsResolver) absContextPath(value any) (any, error) {
	v := value.(string)
	if strings.Contains(v, "://") { // `docker-image://` or any builder specific context type
		return v, nil
	}
	if isRemoteContext(v) {
		return v, nil
	}
	return r.absPath(v)
}

// isRemoteContext returns true if the value is a Git reference or HTTP(S) URL.
//
// Any other value is assumed to be a local filesystem path and returns false.
//
// See: https://github.com/moby/buildkit/blob/18fc875d9bfd6e065cd8211abc639434ba65aa56/frontend/dockerui/context.go#L76-L79
func isRemoteContext(maybeURL string) bool {
	for _, prefix := range []string{"https://", "http://", "git://", "ssh://", "github.com/", "git@"} {
		if strings.HasPrefix(maybeURL, prefix) {
			return true
		}
	}
	return false
}
