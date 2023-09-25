//go:build !linux

package inject

func isNoExec(path string) (bool, error) {
	return false, nil
}
