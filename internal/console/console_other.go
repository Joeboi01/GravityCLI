//go:build !windows

// Package console provides a no-op stub for non-Windows platforms.
package console

// EnableVirtualTerminalProcessing is a no-op on non-Windows platforms
// since ANSI escape codes work out of the box on macOS, Linux, and WSL.
func EnableVirtualTerminalProcessing() {}
