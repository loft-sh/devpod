// +build !windows

package ansi

import (
	"fmt"
)

func EraseInLine(mode int) {
	fmt.Printf("\x1b[%dK", mode)
}
