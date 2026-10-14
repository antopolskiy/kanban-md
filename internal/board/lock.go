package board

import (
	"fmt"
	"path/filepath"

	"github.com/antopolskiy/kanban-md/internal/filelock"
)

// LockBoard acquires the board coordination file. Mutations that read a task
// and write it back must hold this lock for the whole sequence so two callers
// cannot claim the same task.
//
// The lock is the advisory lock from internal/filelock. On macOS that is
// flock(2), which fcntl(2) recommends for whole-file locks: POSIX record locks
// are dropped when any descriptor for the locked file is closed, and a task
// write opens and closes its file.
func LockBoard(dir string) (func() error, error) {
	unlock, err := filelock.Lock(filepath.Join(dir, ".lock"))
	if err != nil {
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}
	return unlock, nil
}
