package diff2

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser launches a browser pointing at url.
// Prefers `cmux browser open <url>` when cmux is on PATH, so the UI
// appears inside the current cmux workspace instead of the user's
// default OS browser. Falls back to native openers.
func OpenBrowser(url string) error {
	if _, err := exec.LookPath("cmux"); err == nil {
		cmd := exec.Command("cmux", "browser", "open", url)
		if err := cmd.Start(); err == nil {
			return nil
		}
		// fall through to native opener on cmux failure
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("diff2: no browser opener for %s", runtime.GOOS)
	}
	return cmd.Start()
}
