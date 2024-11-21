package metrics

func Short(str string) string {
	if len(str) <= 80 {
		return str
	}
	return str[:80] + "..."
}
