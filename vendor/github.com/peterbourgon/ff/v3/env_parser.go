package ff

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// EnvParser is a parser for .env files. Each line is tokenized on the first `=`
// character. The first token is interpreted as the flag name, and the second
// token is interpreted as the value. Both tokens are trimmed of leading and
// trailing whitespace. If the value is "double quoted", control characters like
// `\n` are expanded. Lines beginning with `#` are interpreted as comments.
//
// EnvParser respects WithEnvVarPrefix, e.g. an .env file containing `A_B=c`
// will set a flag named "b" if Parse is called with WithEnvVarPrefix("A").
func EnvParser(r io.Reader, set func(name, value string) error) error {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue // skip empties
		}

		if line[0] == '#' {
			continue // skip comments
		}

		index := strings.IndexRune(line, '=')
		if index < 0 {
			return fmt.Errorf("invalid line: %s", line)
		}

		var (
			name  = strings.TrimSpace(line[:index])
			value = strings.TrimSpace(line[index+1:])
		)

		if len(name) <= 0 {
			return fmt.Errorf("invalid line: %s", line)
		}

		if len(value) <= 0 {
			return fmt.Errorf("invalid line: %s", line)
		}

		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}

		if err := set(name, value); err != nil {
			return err
		}
	}
	return nil
}
