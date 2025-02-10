package types

import "io"

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}
