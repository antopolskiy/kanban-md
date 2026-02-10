package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/watcher"
)

// TestWatcher_CancelWithPendingDebounce verifies context cancel with a
// pending debounce timer doesn't hang or panic.
func TestWatcher_CancelWithPendingDebounce(t *testing.T) {
	dir := t.TempDir()

	var called atomic.Int32
	w, err := watcher.New([]string{dir}, func() {
		called.Add(1)
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx, nil)
		close(done)
	}()

	// Give watcher time to start.
	time.Sleep(50 * time.Millisecond)
	// Trigger a file change to start the debounce timer.
	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Cancel immediately — debounce timer should be pending.
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK - Run returned cleanly even with pending timer.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel with pending debounce")
	}
}

// TestWatcher_NewWithMultiplePathsOneInvalid verifies that New fails if any
// path is invalid, and the watcher is cleaned up.
func TestWatcher_NewWithMultiplePathsOneInvalid(t *testing.T) {
	validDir := t.TempDir()
	_, err := watcher.New([]string{validDir, "/nonexistent/path"}, func() {})
	if err == nil {
		t.Fatal("expected error when one path is invalid")
	}
}

// TestWatcher_CloseStopsWatching verifies Close terminates the watcher.
func TestWatcher_CloseStopsWatching(t *testing.T) {
	dir := t.TempDir()

	w, err := watcher.New([]string{dir}, func() {})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Close immediately.
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Run should return quickly after Close (events channel closed).
	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		w.Run(ctx, nil)
		close(done)
	}()

	select {
	case <-done:
		// OK.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Close")
	}
}

// TestWatcher_RunWithErrCallback verifies the error callback is registered.
// We can't easily inject fsnotify errors, so just verify clean start/stop
// with a non-nil error callback.
func TestWatcher_RunWithErrCallback(t *testing.T) {
	dir := t.TempDir()

	w, err := watcher.New([]string{dir}, func() {})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx, func(err error) {
			t.Logf("watcher error: %v", err)
		})
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK.
	case <-time.After(2 * time.Second):
		t.Fatal("Run with errFn did not return after cancel")
	}
}
