// Package notify is the cly-every desktop notification subsystem. It is
// a thin shim over pkg/notify that maps every's three transition levels
// (failing/recovered/gaveup) to titles + sounds.
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

	// ConfigDir resolves ~/.config/cly/every. Override in tests.
	ConfigDir = defaultConfigDir

	shared     pkgnotify.Notifier
	sharedOnce sync.Once
)

// Shared returns the process-wide notifier.
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
