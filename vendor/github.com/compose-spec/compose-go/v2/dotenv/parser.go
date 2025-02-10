package dotenv

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	charComment       = '#'
	prefixSingleQuote = '\''
	prefixDoubleQuote = '"'
)

var (
	escapeSeqRegex = regexp.MustCompile(`(\\(?:[abcfnrtv$"\\]|0\d{0,3}))`)
	exportRegex    = regexp.MustCompile(`^export\s+`)
)

type parser struct {
	line int
}

func newParser() *parser {
	return &parser{
		line: 1,
	}
}

func (p *parser) parse(src string, out map[string]string, lookupFn LookupFn) error {
	cutset := src
	if lookupFn == nil {
		lookupFn = noLookupFn
	}
	for {
		cutset = p.getStatementStart(cutset)
		if cutset == "" {
			// reached end of file
			break
		}

		key, left, inherited, err := p.locateKeyName(cutset)
		if err != nil {
			return err
		}
		if strings.Contains(key, " ") {
			return fmt.Errorf("line %d: key cannot contain a space", p.line)
		}

		if inherited {
			value, ok := lookupFn(key)
			if ok {
				out[key] = value
			}
			cutset = left
			continue
		}

		value, left, err := p.extractVarValue(left, out, lookupFn)
		if err != nil {
			return err
		}

		out[key] = value
		cutset = left
	}

	return nil
}

// getStatementPosition returns position of statement begin.
//
// It skips any comment line or non-whitespace character.
func (p *parser) getStatementStart(src string) string {
	pos := p.indexOfNonSpaceChar(src)
	if pos == -1 {
		return ""
	}

	src = src[pos:]
	if src[0] != charComment {
		return src
	}

	// skip comment section
	pos = strings.IndexFunc(src, isCharFunc('\n'))
	if pos == -1 {
		return ""
	}
	return p.getStatementStart(src[pos:])
}

// locateKeyName locates and parses key name and returns rest of slice
func (p *parser) locateKeyName(src string) (string, string, bool, error) {
	var key string
	var inherited bool
	// trim "export" and space at beginning
	if exportRegex.MatchString(src) {
		// we use a `strings.trim` to preserve the pointer to the same underlying memory.
		// a regexp replace would copy the string.
		src = strings.TrimLeftFunc(strings.TrimPrefix(src, "export"), isSpace)
	}

	// locate key name end and validate it in single loop
	offset := 0
loop:
	for i, rune := range src {
		if isSpace(rune) {
			continue
		}

		switch rune {
		case '=', ':', '\n':
			// library also supports yaml-style value declaration
			key = string(src[0:i])
			offset = i + 1
			inherited = rune == '\n'
			break loop
		case '_', '.', '[', ']':
		default:
			// variable name should match [A-Za-z0-9_.-]
			if unicode.IsLetter(rune) || unicode.IsNumber(rune) {
				continue
			}

			return "", "", inherited, fmt.Errorf(
				`line %d: unexpected character %q in variable name %q`,
				p.line, string(rune), strings.Split(src, "\n")[0])
		}
	}

	if src == "" {
		return "", "", inherited, errors.New("zero length string")
	}

	// trim whitespace
	key = strings.TrimRightFunc(key, unicode.IsSpace)
	cutset := strings.TrimLeftFunc(src[offset:], isSpace)
	return key, cutset, inherited, nil
}

// extractVarValue extracts variable value and returns rest of slice
func (p *parser) extractVarValue(src string, envMap map[string]string, lookupFn LookupFn) (string, string, error) {
	quote, isQuoted := hasQuotePrefix(src)
	if !isQuoted {
		// unquoted value - read until new line
		value, rest, _ := strings.Cut(src, "\n")
		p.line++

		// Remove inline comments on unquoted lines
		value, _, _ = strings.Cut(value, " #")
		value = strings.TrimRightFunc(value, unicode.IsSpace)
		retVal, err := expandVariables(string(value), envMap, lookupFn)
		return retVal, rest, err
	}

	previousCharIsEscape := false
	// lookup quoted string terminator
	var chars []byte
	for i := 1; i < len(src); i++ {
		char := src[i]
		if char == '\n' {
			p.line++
		}
		if char != quote {
			if !previousCharIsEscape && char == '\\' {
				previousCharIsEscape = true
				continue
			}
			if previousCharIsEscape {
				previousCharIsEscape = false
				chars = append(chars, '\\')
			}
			chars = append(chars, char)
			continue
		}

		// skip escaped quote symbol (\" or \', depends on quote)
		if previousCharIsEscape {
			previousCharIsEscape = false
			chars = append(chars, char)
			continue
		}

		// trim quotes
		value := string(chars)
		if quote == prefixDoubleQuote {
			// expand standard shell escape sequences & then interpolate
			// variables on the result
			retVal, err := expandVariables(expandEscapes(value), envMap, lookupFn)
			if err != nil {
				return "", "", err
			}
			value = retVal
		}

		return value, src[i+1:], nil
	}

	// return formatted error if quoted string is not terminated
	valEndIndex := strings.IndexFunc(src, isCharFunc('\n'))
	if valEndIndex == -1 {
		valEndIndex = len(src)
	}

	return "", "", fmt.Errorf("line %d: unterminated quoted value %s", p.line, src[:valEndIndex])
}

func expandEscapes(str string) string {
	out := escapeSeqRegex.ReplaceAllStringFunc(str, func(match string) string {
		if match == `\$` {
			// `\$` is not a Go escape sequence, the expansion parser uses
			// the special `$$` syntax
			// both `FOO=\$bar` and `FOO=$$bar` are valid in an env file and
			// will result in FOO w/ literal value of "$bar" (no interpolation)
			return "$$"
		}

		if strings.HasPrefix(match, `\0`) {
			// octal escape sequences in Go are not prefixed with `\0`, so
			// rewrite the prefix, e.g. `\0123` -> `\123` -> literal value "S"
			match = strings.Replace(match, `\0`, `\`, 1)
		}

		// use Go to unquote (unescape) the literal
		// see https://go.dev/ref/spec#Rune_literals
		//
		// NOTE: Go supports ADDITIONAL escapes like `\x` & `\u` & `\U`!
		// These are NOT supported, which is why we use a regex to find
		// only matches we support and then use `UnquoteChar` instead of a
		// `Unquote` on the entire value
		v, _, _, err := strconv.UnquoteChar(match, '"')
		if err != nil {
			return match
		}
		return string(v)
	})
	return out
}

func (p *parser) indexOfNonSpaceChar(src string) int {
	return strings.IndexFunc(src, func(r rune) bool {
		if r == '\n' {
			p.line++
		}
		return !unicode.IsSpace(r)
	})
}

// hasQuotePrefix reports whether charset starts with single or double quote and returns quote character
func hasQuotePrefix(src string) (byte, bool) {
	if src == "" {
		return 0, false
	}

	switch quote := src[0]; quote {
	case prefixDoubleQuote, prefixSingleQuote:
		return quote, true // isQuoted
	default:
		return 0, false
	}
}

func isCharFunc(char rune) func(rune) bool {
	return func(v rune) bool {
		return v == char
	}
}

// isSpace reports whether the rune is a space character but not line break character
//
// this differs from unicode.IsSpace, which also applies line break as space
func isSpace(r rune) bool {
	switch r {
	case '\t', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	}
	return false
}
