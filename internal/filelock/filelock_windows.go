//go:build windows

package filelock

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
	lockRetryInterval       = time.Millisecond
)

func lockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	for {
		err := windows.LockFileEx(
			windows.Handle(f.Fd()),
			lockfileExclusiveLock|lockfileFailImmediately,
			0, // reserved
			1, // lock 1 byte
			0, // high word
			ol,
		)
		if err == nil {
			return nil
		}
		// ERROR_LOCK_VIOLATION means another handle holds the lock.
		// Sleep briefly to yield to the Go scheduler and retry.
		// Without LOCKFILE_FAIL_IMMEDIATELY, LockFileEx blocks the OS thread,
		// which can starve goroutines and cause deadlocks.
		if !errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return err
		}
		time.Sleep(lockRetryInterval)
	}
}

func unlockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0, // reserved
		1, // unlock 1 byte
		0, // high word
		ol,
	)
}
