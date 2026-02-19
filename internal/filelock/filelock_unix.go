//go:build !windows

package filelock

import (
	"os"
	"syscall"
)

func lockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX) //nolint:gosec // Fd returns uintptr, int cast is safe for flock
}

func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:gosec // Fd returns uintptr, int cast is safe for flock
}
