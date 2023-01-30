package hash

import (
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"io"
)

// Sha256 hashes a given string to a sha256 string
func Sha256(s string) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, s)

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// StringToNumber hashes a given string to a number
func StringToNumber(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
