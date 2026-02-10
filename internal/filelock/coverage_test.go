package filelock_test

import (
	"path/filepath"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/filelock"
)

func TestLock_InvalidPath(t *testing.T) {
	// Try to lock a file in a nonexistent directory.
	_, err := filelock.Lock("/nonexistent/dir/.lock")
	if err == nil {
		t.Fatal("expected error for invalid lock path")
	}
}

func TestLock_UnlockReturnsNil(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".lock")

	unlock, err := filelock.Lock(lockPath)
	if err != nil {
		t.Fatalf("Lock() error: %v", err)
	}

	// Unlock should succeed and return nil.
	if err := unlock(); err != nil {
		t.Errorf("unlock() error: %v", err)
	}
}

func TestLock_SequentialLockUnlock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".lock")

	// Lock, unlock, then lock again — should work without error.
	unlock1, err := filelock.Lock(lockPath)
	if err != nil {
		t.Fatalf("first Lock() error: %v", err)
	}
	if unlockErr := unlock1(); unlockErr != nil {
		t.Fatalf("first unlock() error: %v", unlockErr)
	}

	unlock2, err := filelock.Lock(lockPath)
	if err != nil {
		t.Fatalf("second Lock() error: %v", err)
	}
	if unlockErr := unlock2(); unlockErr != nil {
		t.Fatalf("second unlock() error: %v", unlockErr)
	}
}

func TestLock_DoubleUnlockReturnsError(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".lock")

	unlock, err := filelock.Lock(lockPath)
	if err != nil {
		t.Fatalf("Lock() error: %v", err)
	}

	// First unlock should succeed.
	if unlockErr := unlock(); unlockErr != nil {
		t.Fatalf("first unlock() error: %v", unlockErr)
	}

	// Second unlock operates on a closed fd — should return an error.
	err = unlock()
	if err == nil {
		t.Error("expected error on double unlock (closed fd)")
	}
}
