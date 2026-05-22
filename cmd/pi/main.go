// Standalone `pi` binary: cly's wrapper around the upstream pi binary.
// Adds --name / -n to label the session and rename the current cmux
// tab, then forwards all remaining args to pi.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/yurifrl/cly/modules/piwrap"
)

func main() {
	err := piwrap.Run(os.Args[1:])
	if err == nil {
		return
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
