package terminal

import (
	"io"

	dockerterm "github.com/moby/term"
	"k8s.io/kubectl/pkg/util/term"
)

// SetupTTY creates a term.TTY (docker)
func SetupTTY(stdin io.Reader, stdout io.Writer) (bool, term.TTY) {
	t := term.TTY{
		Out: stdout,
		In:  stdin,
	}

	if !t.IsTerminalIn() {
		return false, t
	}

	// if we get to here, the user wants to attach stdin, wants a TTY, and In is a terminal, so we
	// can safely set t.Raw to true
	t.Raw = true

	newStdin, newStdout, _ := dockerterm.StdStreams()
	t.In = newStdin
	if stdout != nil {
		t.Out = newStdout
	}

	return true, t
}
