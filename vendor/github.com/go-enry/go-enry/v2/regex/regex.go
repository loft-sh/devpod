package regex

// Package regex abstracts regular expression engine
// that can be chosen at compile-time by a build tag.

const (
	RE2       = "RE2"
	Oniguruma = "Oniguruma"
)
