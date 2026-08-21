package envs

import (
	"fmt"
	"os/exec"
)

// setLaunchctlEnvs calls `launchctl setenv` for each field so all GUI-spawned
// processes (terminal emulators, editors, etc.) inherit the variables without
// needing a shell to source them first.
func setLaunchctlEnvs(fields []Field) error {
	for _, field := range fields {
		cmd := exec.Command("launchctl", "setenv", field.Label, field.Value)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("launchctl setenv %s: %w: %s", field.Label, err, output)
		}
	}
	return nil
}
