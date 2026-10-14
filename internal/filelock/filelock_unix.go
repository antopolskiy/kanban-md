//go:build !windows

package filelock

import (
	"os"
	"syscall"
)

// lockFile uses flock(2). On macOS, fcntl(2) documents flock as the whole-file
// advisory lock to use when library code will open and close files: POSIX
// record locks (F_SETLKW) are released when any descriptor for that file is
// closed, so they do not survive a task write.
func lockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX) //nolint:gosec // Fd returns uintptr, int cast is safe for flock
}

func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:gosec // Fd returns uintptr, int cast is safe for flock
}
