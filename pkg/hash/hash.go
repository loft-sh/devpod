package hash

import "hash/fnv"

// StringToNumber hashes a given string to a number
func StringToNumber(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
