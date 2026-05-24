//go:build windows

// Package console enables ANSI/VT100 escape code support on Windows.
// This is required for Bubble Tea and Lip Gloss color output to render
// correctly in CMD.exe, PowerShell 5, and Windows Terminal.
package console

import (
	"golang.org/x/sys/windows"
)

// EnableVirtualTerminalProcessing enables ANSI color output in the Windows
// console and configures standard input/output to use UTF-8 encoding (CP 65001).
// This guarantees that Lip Gloss colors, borders, and emojis render perfectly
// in CMD.exe and PowerShell without encoding artifacts.
func EnableVirtualTerminalProcessing() {
	// Set console code page and output code page to UTF-8 (65001)
	_ = windows.SetConsoleCP(65001)
	_ = windows.SetConsoleOutputCP(65001)

	for _, name := range []windows.Handle{
		windows.Stdout,
		windows.Stderr,
	} {
		var mode uint32
		if err := windows.GetConsoleMode(name, &mode); err != nil {
			continue
		}
		mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
		_ = windows.SetConsoleMode(name, mode)
	}
}
