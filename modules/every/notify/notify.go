// Package notify is the cly-every desktop notification subsystem. It is
// self-contained: nothing here imports pkg/cmux or any other cly module.
package notify

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Level enumerates the three transitions cly-every emits.
type Level int

const (
	LevelFailing Level = iota
	LevelRecovered
	LevelGaveUp
)

// String returns the canonical key used in config files.
func (l Level) String() string {
	switch l {
	case LevelFailing:
		return "failing"
	case LevelRecovered:
		return "recovered"
	case LevelGaveUp:
		return "gaveup"
	}
	return "unknown"
}

// Notification is the input to Send.
type Notification struct {
	TaskName string
	Level    Level
	Title    string
	Body     string
}

var (
	defaultSounds = map[Level]string{
		LevelFailing:   "Basso",
		LevelRecovered: "Glass",
		LevelGaveUp:    "Sosumi",
	}

	missingWarnOnce sync.Once

	// Runner is the os/exec entrypoint. Tests can swap it.
	Runner = func(ctx context.Context, name string, args ...string) error {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// LookPath is swappable for tests.
	LookPath = exec.LookPath

	// ConfigDir resolves ~/.config/cly/every. Override in tests.
	ConfigDir = defaultConfigDir
)

func defaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cly", "every"), nil
}

// Send shells out to terminal-notifier. When terminal-notifier is missing or
// returns an error this never crashes the caller.
func Send(ctx context.Context, n Notification) error {
	if _, err := LookPath("terminal-notifier"); err != nil {
		missingWarnOnce.Do(func() {
			fmt.Fprintln(os.Stderr, "cly every notify: terminal-notifier not on PATH; desktop notifications disabled")
		})
		return nil
	}

	icon, ierr := iconPath(n.Level)
	if ierr != nil {
		fmt.Fprintf(os.Stderr, "cly every notify: icon resolve: %v\n", ierr)
	}

	sound := resolveSound(n.Level)
	args := []string{
		"-title", n.Title,
		"-message", n.Body,
		"-sound", sound,
		"-group", "cly.every." + n.TaskName,
	}
	if icon != "" {
		args = append(args, "-appIcon", icon)
	}
	if err := Runner(ctx, "terminal-notifier", args...); err != nil {
		fmt.Fprintf(os.Stderr, "cly every notify: %v\n", err)
		return nil
	}
	return nil
}

// iconPath returns either a user override or the embedded extracted path.
func iconPath(l Level) (string, error) {
	if dir, err := ConfigDir(); err == nil {
		p := filepath.Join(dir, "icons", l.String()+".png")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("cly-every-%s.png", l.String()))
	if _, err := os.Stat(tmp); err == nil {
		return tmp, nil
	}
	data := iconBytes(l)
	if len(data) == 0 {
		return "", fmt.Errorf("no icon for level %s", l)
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	return tmp, nil
}

// resolveSound picks a sound name from sounds.toml, falling back to defaults.
func resolveSound(l Level) string {
	def := defaultSounds[l]
	dir, err := ConfigDir()
	if err != nil {
		return def
	}
	f, err := os.Open(filepath.Join(dir, "sounds.toml"))
	if err != nil {
		return def
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(strings.Trim(v, `"'`))
		if k == l.String() && v != "" {
			return v
		}
	}
	return def
}
