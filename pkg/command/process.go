package command

func IsRunning(pid string) (bool, error) {
	return isRunning(pid)
}

func Kill(pid string) error {
	return kill(pid)
}
