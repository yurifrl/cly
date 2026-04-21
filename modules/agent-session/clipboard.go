package agentsession

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// copyToClipboard copies s to the system clipboard using the platform's
// native tool (pbcopy on macOS, xclip/xsel/wl-copy on Linux).
func copyToClipboard(s string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		for _, bin := range []string{"wl-copy", "xclip", "xsel"} {
			if _, err := exec.LookPath(bin); err == nil {
				switch bin {
				case "xclip":
					cmd = exec.Command("xclip", "-selection", "clipboard")
				case "xsel":
					cmd = exec.Command("xsel", "--clipboard", "--input")
				default:
					cmd = exec.Command(bin)
				}
				break
			}
		}
		if cmd == nil {
			return fmt.Errorf("no clipboard tool found (install wl-copy, xclip, or xsel)")
		}
	default:
		return fmt.Errorf("clipboard unsupported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}
