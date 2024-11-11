package huh

func clamp(n, low, high int) int {
	if low > high {
		low, high = high, low
	}
	return min(high, max(low, n))
}
