//go:build windows

package output

import (
	"os"

	"golang.org/x/sys/windows"
)

const enableVirtualTerminalProcessing = 0x0004

// enableVT tries to turn on ANSI escape sequence processing for the console.
// Returns false if the console doesn't support it (legacy PowerShell, cmd.exe
// on older Windows builds, or non-TTY output).
func enableVT() bool {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return false
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}
	return windows.SetConsoleMode(h, mode|enableVirtualTerminalProcessing) == nil
}
