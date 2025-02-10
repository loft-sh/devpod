package ansi

import (
	"bytes"

	"github.com/charmbracelet/x/ansi/parser"
	"github.com/rivo/uniseg"
)

// Truncate truncates a string to a given length, adding a tail to the
// end if the string is longer than the given length.
// This function is aware of ANSI escape codes and will not break them, and
// accounts for wide-characters (such as East Asians and emojis).
func Truncate(s string, length int, tail string) string {
	if sw := StringWidth(s); sw <= length {
		return s
	}

	tw := StringWidth(tail)
	length -= tw
	if length < 0 {
		return ""
	}

	var cluster []byte
	var buf bytes.Buffer
	curWidth := 0
	ignoring := false
	pstate := parser.GroundState // initial state
	b := []byte(s)
	i := 0

	// Here we iterate over the bytes of the string and collect printable
	// characters and runes. We also keep track of the width of the string
	// in cells.
	// Once we reach the given length, we start ignoring characters and only
	// collect ANSI escape codes until we reach the end of string.
	for i < len(b) {
		state, action := parser.Table.Transition(pstate, b[i])
		if state == parser.Utf8State {
			// This action happens when we transition to the Utf8State.
			var width int
			cluster, _, width, _ = uniseg.FirstGraphemeCluster(b[i:], -1)

			// increment the index by the length of the cluster
			i += len(cluster)

			// Are we ignoring? Skip to the next byte
			if ignoring {
				continue
			}

			// Is this gonna be too wide?
			// If so write the tail and stop collecting.
			if curWidth+width > length && !ignoring {
				ignoring = true
				buf.WriteString(tail)
			}

			if curWidth+width > length {
				continue
			}

			curWidth += width
			buf.Write(cluster)

			// Done collecting, now we're back in the ground state.
			pstate = parser.GroundState
			continue
		}

		switch action {
		case parser.PrintAction:
			// Is this gonna be too wide?
			// If so write the tail and stop collecting.
			if curWidth >= length && !ignoring {
				ignoring = true
				buf.WriteString(tail)
			}

			// Skip to the next byte if we're ignoring
			if ignoring {
				i++
				continue
			}

			// collects printable ASCII
			curWidth++
			fallthrough
		default:
			buf.WriteByte(b[i])
			i++
		}

		// Transition to the next state.
		pstate = state

		// Once we reach the given length, we start ignoring runes and write
		// the tail to the buffer.
		if curWidth > length && !ignoring {
			ignoring = true
			buf.WriteString(tail)
		}
	}

	return buf.String()
}
