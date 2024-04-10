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

package dotenv

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func GetEnvFromFile(currentEnv map[string]string, workingDir string, filenames []string) (map[string]string, error) {
	envMap := make(map[string]string)

	dotEnvFiles := filenames
	if len(dotEnvFiles) == 0 {
		dotEnvFiles = append(dotEnvFiles, filepath.Join(workingDir, ".env"))
	}
	for _, dotEnvFile := range dotEnvFiles {
		abs, err := filepath.Abs(dotEnvFile)
		if err != nil {
			return envMap, err
		}
		dotEnvFile = abs

		s, err := os.Stat(dotEnvFile)
		if os.IsNotExist(err) {
			if len(filenames) == 0 {
				return envMap, nil
			}
			return envMap, errors.Errorf("Couldn't find env file: %s", dotEnvFile)
		}
		if err != nil {
			return envMap, err
		}

		if s.IsDir() {
			if len(filenames) == 0 {
				return envMap, nil
			}
			return envMap, errors.Errorf("%s is a directory", dotEnvFile)
		}

		b, err := os.ReadFile(dotEnvFile)
		if os.IsNotExist(err) {
			return nil, errors.Errorf("Couldn't read env file: %s", dotEnvFile)
		}
		if err != nil {
			return envMap, err
		}

		env, err := ParseWithLookup(bytes.NewReader(b), func(k string) (string, bool) {
			v, ok := currentEnv[k]
			if ok {
				return v, true
			}
			v, ok = envMap[k]
			return v, ok
		})
		if err != nil {
			return envMap, errors.Wrapf(err, "failed to read %s", dotEnvFile)
		}
		for k, v := range env {
			envMap[k] = v
		}
	}

	return envMap, nil
}
