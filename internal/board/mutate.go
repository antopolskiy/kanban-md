package board

import (
	"fmt"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// DeleteResult is returned after a successful soft-delete (archive).
type DeleteResult struct {
	Task     *task.Task
	Warnings []string // dependent task warnings
}

// Delete soft-deletes (archives) a task. It validates claim ownership and
// collects warnings about dependent tasks. The operation is idempotent —
// archiving an already-archived task is a no-op.
func Delete(cfg *config.Config, id int, claimant string, now time.Time) (*DeleteResult, error) {
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

	// Collect dependent warnings (best-effort).
	warnings := FindDependents(cfg.TasksPath(), id)

	// Idempotent: already archived → no-op.
	if t.Status == config.ArchivedStatus {
		return &DeleteResult{Task: t, Warnings: warnings}, nil
	}

	oldStatus := t.Status
	t.Status = config.ArchivedStatus
	task.UpdateTimestamps(t, oldStatus, t.Status, cfg)
	t.Updated = now

	if err := task.Write(path, t); err != nil {
		return nil, fmt.Errorf("writing task: %w", err)
	}

	LogMutation(cfg.Dir(), "delete", t.ID, t.Title)

	return &DeleteResult{Task: t, Warnings: warnings}, nil
}

// MoveParams contains the parameters for a Move operation.
type MoveParams struct {
	ID        int
	NewStatus string
	Claimant  string // for claim validation; also set as claim if SetClaim is true
	SetClaim  bool   // whether to update the task's claim fields
}

// MoveResult is returned after a successful move.
type MoveResult struct {
	Task      *task.Task
	OldStatus string   // empty if idempotent (already at target status)
	Warnings  []string // e.g., "task is blocked"
}

// Move changes a task's status. It validates claim ownership, enforces WIP
// limits (including class-of-service awareness), and checks require_claim
// for the target status. The operation is idempotent — moving to the current
// status is a no-op.
func Move(cfg *config.Config, params MoveParams, now time.Time) (*MoveResult, error) {
	path, err := task.FindByID(cfg.TasksPath(), params.ID)
	if err != nil {
		return nil, err
	}

	t, err := task.Read(path)
	if err != nil {
		return nil, err
	}

	// Validate claim ownership.
	if err := task.CheckClaim(t, params.Claimant, cfg.ClaimTimeoutDuration()); err != nil {
		return nil, err
	}

	// Idempotent: already at target status.
	if t.Status == params.NewStatus {
		return &MoveResult{Task: t}, nil
	}

	// Enforce require_claim for target status.
	if cfg.StatusRequiresClaim(params.NewStatus) && params.Claimant == "" {
		return nil, task.ValidateClaimRequired(params.NewStatus)
	}

	// WIP limit enforcement (class-aware).
	if err := enforceMoveWIP(cfg, t, params.NewStatus); err != nil {
		return nil, err
	}

	// Collect warnings.
	var warnings []string
	if t.Blocked {
		warnings = append(warnings, fmt.Sprintf("task #%d is blocked (%s)", t.ID, t.BlockReason))
	}

	oldStatus := t.Status
	t.Status = params.NewStatus
	task.UpdateTimestamps(t, oldStatus, params.NewStatus, cfg)

	// Apply claim if requested.
	if params.SetClaim && params.Claimant != "" {
		t.ClaimedBy = params.Claimant
		t.ClaimedAt = &now
	}

	t.Updated = now

	if err := task.Write(path, t); err != nil {
		return nil, fmt.Errorf("writing task: %w", err)
	}

	LogMutation(cfg.Dir(), "move", t.ID, oldStatus+" -> "+params.NewStatus)

	return &MoveResult{Task: t, OldStatus: oldStatus, Warnings: warnings}, nil
}

// enforceMoveWIP checks WIP limits, considering class of service.
func enforceMoveWIP(cfg *config.Config, t *task.Task, newStatus string) error {
	if t.Class != "" && len(cfg.Classes) > 0 {
		return enforceClassWIP(cfg, t, newStatus)
	}

	allTasks, _, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return fmt.Errorf("reading tasks for WIP check: %w", err)
	}
	counts := CountByStatus(allTasks)
	return CheckWIPLimit(cfg, counts, newStatus, t.Status)
}

// enforceClassWIP checks class-level and column-level WIP limits.
func enforceClassWIP(cfg *config.Config, t *task.Task, newStatus string) error {
	classConf := cfg.ClassByName(t.Class)

	// Check class-level board-wide WIP limit.
	if classConf != nil && classConf.WIPLimit > 0 {
		allTasks, _, err := task.ReadAllLenient(cfg.TasksPath())
		if err != nil {
			return fmt.Errorf("reading tasks for class WIP check: %w", err)
		}
		count := countByClass(allTasks, t.Class, t.ID)
		if count >= classConf.WIPLimit {
			return task.ValidateClassWIPExceeded(t.Class, classConf.WIPLimit, count)
		}
	}

	// If class bypasses column WIP, skip column check.
	if classConf != nil && classConf.BypassColumnWIP {
		return nil
	}

	// Normal column WIP check.
	allTasks, _, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return fmt.Errorf("reading tasks for WIP check: %w", err)
	}
	counts := CountByStatus(allTasks)
	return CheckWIPLimit(cfg, counts, newStatus, t.Status)
}

// countByClass counts tasks with a given class, excluding a specific task ID.
func countByClass(tasks []*task.Task, class string, excludeID int) int {
	count := 0
	for _, t := range tasks {
		if t.Class == class && t.ID != excludeID {
			count++
		}
	}
	return count
}
