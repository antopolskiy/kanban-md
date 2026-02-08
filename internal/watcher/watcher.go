// Package watcher provides debounced file system watching for kanban board directories.
package watcher

import (
	"context"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounceDelay is the time to wait after the last file event before triggering
// a callback. This coalesces rapid changes (e.g., batch operations) into a
// single notification.
const debounceDelay = 100 * time.Millisecond

// Watcher watches kanban board directories for changes and invokes a callback
// with debouncing.
type Watcher struct {
	fsw      *fsnotify.Watcher
	mu       sync.Mutex
	timer    *time.Timer
	callback func()
}

// New creates a Watcher that monitors the given paths for changes.
// The callback is invoked (debounced) whenever a file change is detected.
func New(paths []string, callback func()) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, p := range paths {
		if err := fsw.Add(p); err != nil {
			_ = fsw.Close()
			return nil, err
		}
	}

	return &Watcher{
		fsw:      fsw,
		callback: callback,
	}, nil
}

// Run starts the watch loop. It blocks until the context is canceled.
// Errors from the underlying watcher are passed to the optional errFn callback.
func (w *Watcher) Run(ctx context.Context, errFn func(error)) {
	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			if w.timer != nil {
				w.timer.Stop()
			}
			w.mu.Unlock()
			return
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			// Only react to meaningful operations.
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			w.debounce()
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			if errFn != nil {
				errFn(err)
			}
		}
	}
}

// Close stops the underlying filesystem watcher.
func (w *Watcher) Close() error {
	return w.fsw.Close()
}

func (w *Watcher) debounce() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(debounceDelay, w.callback)
}
