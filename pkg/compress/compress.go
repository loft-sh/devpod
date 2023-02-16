package compress

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
)

// Compress gzips a string and base64 encodes it
func Compress(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	_, err := gz.Write([]byte(s))
	if err != nil {
		return "", err
	}

	err = gz.Flush()
	if err != nil {
		return "", err
	}

	err = gz.Close()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

// Decompress decompresses a string
func Decompress(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}

	rdata := bytes.NewReader(decoded)
	r, err := gzip.NewReader(rdata)
	if err != nil {
		return "", err
	}

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return string(decompressed), nil
}
