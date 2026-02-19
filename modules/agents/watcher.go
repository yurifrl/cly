package agents

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches directories for file changes and debounces events.
type Watcher struct {
	watcher       *fsnotify.Watcher
	onSync        func()
	onReverseSync func(path string)
	targetDirs    []string // watched target directories
	mu            sync.Mutex
	timer         *time.Timer
	maxTimer      *time.Timer
	revTimer      *time.Timer
	revMaxTimer   *time.Timer
	revPaths      map[string]struct{}
	wait          time.Duration
	maxWait       time.Duration
	// Self-sync prevention: ignore events from our own writes
	suppressUntil time.Time
}

const (
	defaultDebounceWait    = 100 * time.Millisecond
	defaultDebounceMaxWait = 500 * time.Millisecond
	selfSyncCooldown       = 200 * time.Millisecond
)

// NewWatcher creates a watcher that calls onSync after debounced file changes.
func NewWatcher(onSync func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher:  fw,
		onSync:   onSync,
		revPaths: make(map[string]struct{}),
		wait:     defaultDebounceWait,
		maxWait:  defaultDebounceMaxWait,
	}, nil
}

// SetReverseSync sets the callback for target→source reverse sync.
func (w *Watcher) SetReverseSync(fn func(path string)) {
	w.onReverseSync = fn
}

// AddTargetDir adds a target directory to watch for reverse sync.
func (w *Watcher) AddTargetDir(dir string) error {
	w.targetDirs = append(w.targetDirs, dir)
	return w.watcher.Add(dir)
}

// Add adds a directory to watch (source dir).
func (w *Watcher) Add(dir string) error {
	return w.watcher.Add(dir)
}

// SuppressBriefly prevents triggering callbacks for a short window.
// Call this right after performing writes to avoid self-sync loops.
func (w *Watcher) SuppressBriefly() {
	w.mu.Lock()
	w.suppressUntil = time.Now().Add(selfSyncCooldown)
	w.mu.Unlock()
}

func (w *Watcher) isSuppressed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return time.Now().Before(w.suppressUntil)
}

// isTargetPath checks if a path is under one of the watched target dirs.
func (w *Watcher) isTargetPath(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	for _, td := range w.targetDirs {
		tabsDir, err := filepath.Abs(td)
		if err != nil {
			tabsDir = td
		}
		if strings.HasPrefix(abs, tabsDir+string(filepath.Separator)) || abs == tabsDir {
			return true
		}
	}
	return false
}

// Run starts the event loop. Blocks until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	defer w.watcher.Close()

	for {
		select {
		case <-ctx.Done():
			w.stopTimers()
			return nil
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Chmod) {
				continue
			}
			if w.isSuppressed() {
				continue
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
				if w.onReverseSync != nil && w.isTargetPath(event.Name) {
					w.debounceReverse(event.Name)
				} else {
					w.debounce()
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			return err
		}
	}
}

func (w *Watcher) debounce() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.wait, w.fire)

	if w.maxTimer == nil {
		w.maxTimer = time.AfterFunc(w.maxWait, w.fire)
	}
}

func (w *Watcher) debounceReverse(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.revPaths[path] = struct{}{}

	if w.revTimer != nil {
		w.revTimer.Stop()
	}
	w.revTimer = time.AfterFunc(w.wait, w.fireReverse)

	if w.revMaxTimer == nil {
		w.revMaxTimer = time.AfterFunc(w.maxWait, w.fireReverse)
	}
}

func (w *Watcher) fire() {
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	if w.maxTimer != nil {
		w.maxTimer.Stop()
		w.maxTimer = nil
	}
	w.mu.Unlock()

	w.onSync()
}

func (w *Watcher) fireReverse() {
	w.mu.Lock()
	if w.revTimer != nil {
		w.revTimer.Stop()
		w.revTimer = nil
	}
	if w.revMaxTimer != nil {
		w.revMaxTimer.Stop()
		w.revMaxTimer = nil
	}
	paths := make([]string, 0, len(w.revPaths))
	for p := range w.revPaths {
		paths = append(paths, p)
	}
	w.revPaths = make(map[string]struct{})
	w.mu.Unlock()

	for _, p := range paths {
		w.onReverseSync(p)
	}
}

func (w *Watcher) stopTimers() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	if w.maxTimer != nil {
		w.maxTimer.Stop()
		w.maxTimer = nil
	}
	if w.revTimer != nil {
		w.revTimer.Stop()
		w.revTimer = nil
	}
	if w.revMaxTimer != nil {
		w.revMaxTimer.Stop()
		w.revMaxTimer = nil
	}
}
