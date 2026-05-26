// Package notify is the cly-every desktop notification subsystem. It is
// a thin shim over pkg/notify, mapping every's three transition levels
// (failing/recovered/gaveup) to default Action sets and routing to the
// shared notifier (native macOS bundle, beeep, zellij).
package notify

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	pkgnotify "github.com/yurifrl/cly/pkg/notify"
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

// Notification is the input to Send. The TaskName is used to derive a
// stable group ID ("cly.every.<task>") so action callbacks can be routed
// back to the right task.
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

	// ConfigDir resolves ~/.config/cly/every. Override in tests.
	ConfigDir = defaultConfigDir

	// Default actions per transition level. Empty for recovered = passive
	// info, no buttons, banner auto-dismisses.
	defaultActions = map[Level][]pkgnotify.Action{
		LevelFailing: {
			{ID: "snooze", Title: "Snooze 5m"},
			{ID: "dismiss", Title: "Dismiss"},
		},
		LevelRecovered: nil,
		LevelGaveUp: {
			{ID: "retry", Title: "Retry"},
			{ID: "dismiss", Title: "Dismiss"},
		},
	}

	// shared is the lazily-initialised pkg/notify backend. One per process.
	shared     pkgnotify.Notifier
	sharedOnce sync.Once
)

// Shared returns the process-wide notifier. The first call initialises the
// pkg/notify backend (native macOS where available, beeep otherwise).
func Shared() pkgnotify.Notifier {
	sharedOnce.Do(func() {
		shared = pkgnotify.New("every", false, false, false, "")
	})
	return shared
}

// SetShared injects a notifier (used by tests).
func SetShared(n pkgnotify.Notifier) {
	sharedOnce.Do(func() {})
	shared = n
}

// GroupFor returns the canonical notification group for a task name.
func GroupFor(taskName string) string {
	return "cly.every." + taskName
}

// Send delivers a notification through the shared pkg/notify backend.
// It is a no-crash wrapper: any underlying error is dropped.
func Send(ctx context.Context, n Notification) error {
	notifier := Shared()
	if notifier == nil || !notifier.Available() {
		return nil
	}
	_ = notifier.Send(ctx, pkgnotify.Notification{
		Title:   n.Title,
		Message: n.Body,
		Sound:   resolveSound(n.Level),
		Group:   GroupFor(n.TaskName),
		Actions: defaultActions[n.Level],
	})
	return nil
}

func defaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cly", "every"), nil
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
