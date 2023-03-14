//go:build !oniguruma
// +build !oniguruma

package regex

import (
	"regexp"
)

const Name = RE2

type EnryRegexp = *regexp.Regexp

func MustCompile(str string) EnryRegexp {
	return regexp.MustCompile(str)
}

// MustCompileMultiline mimics Ruby defaults for regexp, where ^$ matches begin/end of line.
// I.e. it converts Ruby regexp syntaxt to RE2 equivalent
func MustCompileMultiline(s string) EnryRegexp {
	const multilineModeFlag = "(?m)"
	return regexp.MustCompile(multilineModeFlag + s)
}

// MustCompileRuby used for expressions with syntax not supported by RE2.
// Now it's confusing as we use the result as [data/rule.Matcher] and
//
//	(*Matcher)(nil) != nil
//
// What is a better way for an expression to indicate unsupported syntax?
// e.g. add .IsValidSyntax() to both, Matcher interface and EnryRegexp implementations?
func MustCompileRuby(s string) EnryRegexp {
	return nil
}

func QuoteMeta(s string) string {
	return regexp.QuoteMeta(s)
}
