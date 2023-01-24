package terminal

import (
	"bytes"
	"os"
	"testing"

	"gotest.tools/assert"
)

func TestTTY(t *testing.T) {
	buf := make([]byte, 1000)
	writer := bytes.NewBuffer(buf)

	_, tty := SetupTTY(os.Stdin, writer)
	assert.Equal(t, false, tty.Raw, "Raw terminal that doesn't got a terminal stdin")
}
