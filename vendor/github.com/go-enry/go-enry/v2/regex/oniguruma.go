//go:build oniguruma
// +build oniguruma

package regex

import (
	rubex "github.com/go-enry/go-oniguruma"
)

const Name = Oniguruma

type EnryRegexp = *rubex.Regexp

func MustCompile(s string) EnryRegexp {
	return rubex.MustCompileASCII(s)
}

// MustCompileMultiline matches in multi-line mode by default with Oniguruma.
func MustCompileMultiline(s string) EnryRegexp {
	return MustCompile(s)
}

func MustCompileRuby(s string) EnryRegexp {
	return MustCompile(s)
}

func QuoteMeta(s string) string {
	return rubex.QuoteMeta(s)
}
