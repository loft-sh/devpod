package random

import "math/rand"

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

// String creates a new random string with the given length
func String(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func InRange(min, max int) int {
	return rand.Intn(max-min) + min
}
