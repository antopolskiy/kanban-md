package board

import (
	"fmt"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ArchiveResult is returned after a successful archive.
type ArchiveResult struct {
	Task      *task.Task
	OldStatus string
}

// Archive soft-deletes a task by moving it to the archived status.
func Archive(cfg *config.Config, id int, claimant string, now time.Time) (*ArchiveResult, error) {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return nil, err
	}

	t, err := task.Read(path)
	if err != nil {
		return nil, err
	}

	// Validate claim ownership.
	if err := task.CheckClaim(t, claimant, cfg.ClaimTimeoutDuration()); err != nil {
		return nil, err
	}

	targetStatus := config.ArchivedStatus

	// Idempotent: if already archived, return unchanged.
	if t.Status == targetStatus {
		return &ArchiveResult{Task: t, OldStatus: ""}, nil
	}

	oldStatus := t.Status
	t.Status = targetStatus
	task.UpdateTimestamps(t, oldStatus, targetStatus, cfg)
	t.Updated = now

	if err := task.Write(path, t); err != nil {
		return nil, fmt.Errorf("writing task: %w", err)
	}

	LogMutation(cfg.Dir(), "move", t.ID, oldStatus+" -> "+targetStatus)

	return &ArchiveResult{Task: t, OldStatus: oldStatus}, nil
}
