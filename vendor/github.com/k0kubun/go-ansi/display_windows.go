package ansi

import (
	"os"
	"syscall"
	"unsafe"
)

func EraseInLine(mode int) {
	handle := syscall.Handle(os.Stdout.Fd())

	var csbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))

	var w uint32
	var x short
	cursor := csbi.cursorPosition
	switch mode {
	case 1:
		x = csbi.size.x
	case 2:
		x = 0
	case 3:
		cursor.x = 0
		x = csbi.size.x
	}
	procFillConsoleOutputCharacter.Call(uintptr(handle), uintptr(' '), uintptr(x), uintptr(*(*int32)(unsafe.Pointer(&cursor))), uintptr(unsafe.Pointer(&w)))
}
