//go:build windows

package command

func isRunning(pid string) (bool, error) {
	panic("unsupported")
}
