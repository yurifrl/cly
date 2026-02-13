package agents

import (
	"context"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches directories for file changes and debounces events.
type Watcher struct {
	watcher   *fsnotify.Watcher
	onSync    func()
	mu        sync.Mutex
	timer     *time.Timer
	maxTimer  *time.Timer
	wait      time.Duration
	maxWait   time.Duration
}

const (
	defaultDebounceWait    = 100 * time.Millisecond
	defaultDebounceMaxWait = 500 * time.Millisecond
)

// NewWatcher creates a watcher that calls onSync after debounced file changes.
func NewWatcher(onSync func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher: fw,
		onSync:  onSync,
		wait:    defaultDebounceWait,
		maxWait: defaultDebounceMaxWait,
	}, nil
}

// Add adds a directory to watch.
func (w *Watcher) Add(dir string) error {
	return w.watcher.Add(dir)
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
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
				w.debounce()
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
}
