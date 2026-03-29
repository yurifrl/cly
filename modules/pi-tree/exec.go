package pitree

import (
	"os/exec"
)

// runCommand executes a command and returns combined stdout.
func runCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
