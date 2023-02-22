//go:build windows

package command

func isRunning(pid string) (bool, error) {
	panic("unsupported")
}

func kill(pid string) error {
	panic("unsupported")
}
