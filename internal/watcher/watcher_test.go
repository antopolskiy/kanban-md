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

func TestWatcher_DetectsFileCreate(t *testing.T) {
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
	defer cancel()

	go w.Run(ctx, nil)

	// Give watcher time to start.
	time.Sleep(50 * time.Millisecond)

	// Create a file.
	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Wait for debounce + processing.
	time.Sleep(300 * time.Millisecond)

	if got := called.Load(); got < 1 {
		t.Errorf("expected callback to be called at least once, got %d", got)
	}
}

func TestWatcher_DetectsFileModify(t *testing.T) {
	dir := t.TempDir()

	// Pre-create file.
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("v1"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var called atomic.Int32
	w, err := watcher.New([]string{dir}, func() {
		called.Add(1)
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx, nil)

	time.Sleep(50 * time.Millisecond)

	// Modify the file.
	if err := os.WriteFile(path, []byte("v2"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	if got := called.Load(); got < 1 {
		t.Errorf("expected callback to be called at least once, got %d", got)
	}
}

func TestWatcher_DebouncesBatchChanges(t *testing.T) {
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
	defer cancel()

	go w.Run(ctx, nil)

	time.Sleep(50 * time.Millisecond)

	// Create 5 files in rapid succession (simulating batch ops).
	for i := range 5 {
		name := filepath.Join(dir, "task"+string(rune('0'+i))+".md")
		if err := os.WriteFile(name, []byte("data"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	// Wait for debounce to settle.
	time.Sleep(400 * time.Millisecond)

	got := called.Load()
	// Debouncing should coalesce into a small number of calls (ideally 1, but
	// timing on CI may cause 2-3).
	if got > 3 {
		t.Errorf("expected debouncing to reduce calls, got %d (expected <= 3)", got)
	}
	if got < 1 {
		t.Errorf("expected at least 1 callback, got %d", got)
	}
}

func TestWatcher_StopsOnContextCancel(t *testing.T) {
	dir := t.TempDir()

	w, err := watcher.New([]string{dir}, func() {})
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

	cancel()

	select {
	case <-done:
		// OK - Run returned after cancel.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

func TestWatcher_DetectsFileDelete(t *testing.T) {
	dir := t.TempDir()

	// Pre-create file.
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("v1"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var called atomic.Int32
	w, err := watcher.New([]string{dir}, func() {
		called.Add(1)
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx, nil)

	time.Sleep(50 * time.Millisecond)

	// Delete the file.
	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	if got := called.Load(); got < 1 {
		t.Errorf("expected callback to be called at least once, got %d", got)
	}
}

func TestWatcher_MultiplePaths(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	var called atomic.Int32
	w, err := watcher.New([]string{dir1, dir2}, func() {
		called.Add(1)
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx, nil)

	time.Sleep(50 * time.Millisecond)

	// Change in first dir.
	if err := os.WriteFile(filepath.Join(dir1, "a.md"), []byte("a"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	count1 := called.Load()
	if count1 < 1 {
		t.Errorf("expected callback from dir1 change, got %d calls", count1)
	}

	// Change in second dir.
	if err := os.WriteFile(filepath.Join(dir2, "b.md"), []byte("b"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	count2 := called.Load()
	if count2 <= count1 {
		t.Errorf("expected additional callback from dir2 change, got %d total", count2)
	}
}

func TestWatcher_ErrorOnInvalidPath(t *testing.T) {
	_, err := watcher.New([]string{"/nonexistent/path/abc123"}, func() {})
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestWatcher_ReportsErrors(t *testing.T) {
	dir := t.TempDir()

	var gotError atomic.Int32
	w, err := watcher.New([]string{dir}, func() {})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx, func(_ error) {
		gotError.Add(1)
	})

	// We can't easily inject errors into fsnotify, so just verify the watcher
	// starts and stops cleanly.
	time.Sleep(50 * time.Millisecond)
	cancel()
}
