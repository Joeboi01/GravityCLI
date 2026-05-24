//go:build !windows

package browser

import (
	"os/exec"
	"runtime"
)

// Open opens the specified URL in the user's default browser on non-Windows platforms.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return nil
	}
	return cmd.Start()
}
