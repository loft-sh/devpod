package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"os"
)

// String hashes a given string to a sha256 string
func String(s string) string {
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

// File hashes a given file to a sha256 string
func File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
