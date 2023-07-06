package up

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
)

func findMessage(reader io.Reader, message string) error {
	scan := scanner.NewScanner(reader)
	for scan.Scan() {
		line := scan.Bytes()
		if len(line) == 0 {
			continue
		}

		lineObject := &log.Line{}
		err := json.Unmarshal(line, lineObject)
		if err == nil && strings.Contains(lineObject.Message, message) {
			return nil
		}
	}

	return fmt.Errorf("couldn't find message '%s' in log", message)
}

func verifyLogStream(reader io.Reader) error {
	scan := scanner.NewScanner(reader)
	for scan.Scan() {
		line := scan.Bytes()
		if len(line) == 0 {
			continue
		}

		lineObject := &log.Line{}
		err := json.Unmarshal(line, lineObject)
		if err != nil {
			return fmt.Errorf("error reading line %s: %w", string(line), err)
		}
	}

	return nil
}
