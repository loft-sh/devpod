package random

import "math/rand"

func InRange(min, max int) int {
	return rand.Intn(max-min) + min
}
