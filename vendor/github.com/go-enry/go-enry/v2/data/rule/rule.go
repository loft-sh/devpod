// Package rule contains rule-based heuristic implementations.
// It is used in the generated code in content.go for disambiguation of languages
// with colliding extensions, based on regexps from Linguist data.
package rule

import "github.com/go-enry/go-enry/v2/regex"

// Matcher checks if the data matches (number of) pattern(s).
// Every heuristic rule below implements this interface.
// A regexp.Regexp satisfies this interface and can be used instead.
type Matcher interface {
	Match(data []byte) bool
}

// Heuristic consist of (a number of) rules where each, if matches,
// identifies content as belonging to a programming language(s).
type Heuristic interface {
	Matcher
	Languages() []string
}

// languages base struct with all the languages that a Matcher identifies.
type languages struct {
	langs []string
}

// Languages returns all languages, identified by this Matcher.
func (l languages) Languages() []string {
	return l.langs
}

// MatchingLanguages is a helper to create new languages.
func MatchingLanguages(langs ...string) languages {
	return languages{langs}
}

func noLanguages() languages {
	return MatchingLanguages([]string{}...)
}

// Implements a Heuristic.
type or struct {
	languages
	pattern Matcher
}

// Or rule matches, if a single matching pattern exists.
// It receives only one pattern as it relies on optimization that
// represtes union with | inside a single regexp during code generation.
func Or(l languages, p Matcher) Heuristic {
	//FIXME(bzz): this will not be the case as only some of the patterns may
	// be non-RE2 => we shouldn't collate them not to loose the (accuracty of) whole rule
	return or{l, p}
}

// Match implements rule.Matcher.
func (r or) Match(data []byte) bool {
	if runOnRE2AndRegexNotAccepted(r.pattern) {
		return false
	}
	return r.pattern.Match(data)
}

// Implements a Heuristic.
type and struct {
	languages
	patterns []Matcher
}

// And rule matches, if each of the patterns does match.
func And(l languages, m ...Matcher) Heuristic {
	return and{l, m}
}

// Match implements data.Matcher.
func (r and) Match(data []byte) bool {
	for _, p := range r.patterns {
		if runOnRE2AndRegexNotAccepted(p) {
			continue
		}
		if !p.Match(data) {
			return false
		}
	}
	return true
}

// Implements a Heuristic.
type not struct {
	languages
	Patterns []Matcher
}

// Not rule matches if none of the patterns match.
func Not(l languages, r ...Matcher) Heuristic {
	return not{l, r}
}

// Match implements data.Matcher.
func (r not) Match(data []byte) bool {
	for _, p := range r.Patterns {
		if runOnRE2AndRegexNotAccepted(p) {
			continue
		}
		if p.Match(data) {
			return false
		}
	}
	return true
}

// Implements a Heuristic.
type always struct {
	languages
}

// Always rule always matches. Often is used as a default fallback.
func Always(l languages) Heuristic {
	return always{l}
}

// Match implements Matcher.
func (r always) Match(data []byte) bool {
	return true
}

// Checks if a regex syntax isn't accepted by RE2 engine.
// It's nil by construction from regex.MustCompileRuby but
// is used here as a Matcher interface wich itself is non-nil.
func runOnRE2AndRegexNotAccepted(re Matcher) bool {
	v, ok := re.(regex.EnryRegexp)
	return ok && v == nil
}
