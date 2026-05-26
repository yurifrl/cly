package memwatch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Restart finds any running `cly memwatch run` process and sends it SIGTERM.
// It relies on the supervisor (process-compose, see
// ~/.config/process-compose/process-compose.yaml) to respawn it with the
// freshly installed binary.
//
// Returns nil when there's nothing to kill — this is a non-error state so
// post-install hooks can always call it safely.
func Restart(ctx context.Context) error {
	pids, err := findRunPIDs(ctx)
	if err != nil {
		return fmt.Errorf("find memwatch processes: %w", err)
	}

	self := os.Getpid()
	ppid := os.Getppid()
	killed := 0
	for _, pid := range pids {
		if pid == self || pid == ppid {
			continue // never signal ourselves or our parent
		}
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			fmt.Fprintf(os.Stderr, "memwatch: failed to SIGTERM pid %d: %v\n", pid, err)
			continue
		}
		killed++
	}

	if killed == 0 {
		fmt.Fprintln(os.Stderr, "memwatch: no running `cly memwatch run` process found (process-compose will pick up the new binary on next launch)")
		return nil
	}

	fmt.Fprintf(os.Stderr, "memwatch: signalled %d process(es) — process-compose will relaunch with the new binary\n", killed)
	// Give the supervisor a moment to respawn; best-effort, non-blocking if unavailable.
	time.Sleep(300 * time.Millisecond)
	return nil
}

// findRunPIDs returns PIDs of processes whose command line matches `cly memwatch run`.
// It uses `pgrep -af` (BSD/macOS + GNU compatible) so we don't need a ps parser.
func findRunPIDs(ctx context.Context) ([]int, error) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	out, err := exec.CommandContext(cctx, "pgrep", "-af", "cly memwatch run").Output()
	if err != nil {
		// pgrep exits 1 when nothing matches — treat as "no pids".
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}

	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}
