package command

func IsRunning(pid string) (bool, error) {
	return isRunning(pid)
}
