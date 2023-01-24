// +build !windows

package ansi

import (
	"fmt"
)

// Move the cursor n cells to up.
func CursorUp(n int) {
	fmt.Printf("\x1b[%dA", n)
}

// Move the cursor n cells to down.
func CursorDown(n int) {
	fmt.Printf("\x1b[%dB", n)
}

// Move the cursor n cells to right.
func CursorForward(n int) {
	fmt.Printf("\x1b[%dC", n)
}

// Move the cursor n cells to left.
func CursorBack(n int) {
	fmt.Printf("\x1b[%dD", n)
}

// Move cursor to beginning of the line n lines down.
func CursorNextLine(n int) {
	fmt.Printf("\x1b[%dE", n)
}

// Move cursor to beginning of the line n lines up.
func CursorPreviousLine(n int) {
	fmt.Printf("\x1b[%dF", n)
}

// Move cursor horizontally to x.
func CursorHorizontalAbsolute(x int) {
	fmt.Printf("\x1b[%dG", x)
}

// Show the cursor.
func CursorShow() {
	fmt.Print("\x1b[?25h")
}

// Hide the cursor.
func CursorHide() {
	fmt.Print("\x1b[?25l")
}
