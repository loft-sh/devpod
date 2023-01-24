package ansi

import (
	"fmt"
)

var (
	ansiStdout = NewAnsiStdout()
)

// Print prints given arguments with escape sequence conversion for windows.
func Print(a ...interface{}) (n int, err error) {
	return fmt.Fprint(ansiStdout, a...)
}

// Printf prints a given format with escape sequence conversion for windows.
func Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(ansiStdout, format, a...)
}

// Println prints given arguments with newline and escape sequence conversion
// for windows.
func Println(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(ansiStdout, a...)
}
