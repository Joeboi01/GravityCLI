//go:build windows

package browser

import (
	"golang.org/x/sys/windows"
)

// Open opens the specified URL in the user's default browser on Windows.
// It uses ShellExecute to bypass cmd.exe entirely, preventing shell injection
// or URL truncating issues caused by standard shell characters like '&'.
func Open(url string) error {
	verbPtr := windows.StringToUTF16Ptr("open")
	urlPtr := windows.StringToUTF16Ptr(url)
	return windows.ShellExecute(0, verbPtr, urlPtr, nil, nil, windows.SW_SHOWNORMAL)
}
