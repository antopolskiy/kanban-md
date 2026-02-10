package filelock_test

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/filelock"
)

func TestLockExclusive(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".lock")

	unlock, err := filelock.Lock(lockPath)
	if err != nil {
		t.Fatalf("Lock() error: %v", err)
	}
	if err := unlock(); err != nil {
		t.Fatalf("unlock() error: %v", err)
	}
}

func TestLockConcurrent(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".lock")

	const goroutines = 10
	var counter int64
	var maxConcurrent int64
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()

			unlock, err := filelock.Lock(lockPath)
			if err != nil {
				t.Errorf("Lock() error: %v", err)
				return
			}

			// If the lock works, only one goroutine should be in
			// this section at a time.
			cur := atomic.AddInt64(&counter, 1)
			if cur > 1 {
				// Another goroutine is inside the lock — record it.
				for {
					old := atomic.LoadInt64(&maxConcurrent)
					if cur <= old {
						break
					}
					if atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
						break
					}
				}
			}
			atomic.AddInt64(&counter, -1)

			if err := unlock(); err != nil {
				t.Errorf("unlock() error: %v", err)
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	const lockTimeout = 30 * time.Second
	select {
	case <-done:
		// All goroutines completed.
	case <-time.After(lockTimeout):
		t.Fatal("timed out waiting for concurrent lock test — possible deadlock")
	}

	if mc := atomic.LoadInt64(&maxConcurrent); mc > 1 {
		t.Errorf("max concurrent holders = %d, want 1", mc)
	}
}
