package board

import (
	"strings"

	"fmt"
	"path/filepath"
	"time"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
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
	if err := task.ValidateStatus(params.NewStatus, cfg.StatusNames()); err != nil {
		return nil, err
	}

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

// enforceCreateWIP checks WIP limits for a new task (currentStatus is empty).
func enforceCreateWIP(cfg *config.Config, t *task.Task) error {
	if t.Class != "" && len(cfg.Classes) > 0 {
		classConf := cfg.ClassByName(t.Class)
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
		if classConf != nil && classConf.BypassColumnWIP {
			return nil
		}
	}

	allTasks, _, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return fmt.Errorf("reading tasks for WIP check: %w", err)
	}
	counts := CountByStatus(allTasks)
	// Empty currentStatus: new task is not in any column yet.
	return CheckWIPLimit(cfg, counts, t.Status, "")
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

// CreateParams contains the parameters for a Create operation.
// Zero-value fields use config defaults (for Status, Priority, Class).
type CreateParams struct {
	Title     string
	Status    string // empty = config default
	Priority  string // empty = config default
	Class     string // empty = config default
	Assignee  string
	Tags      []string
	Body      string
	Due       *date.Date
	Estimate  string
	Parent    *int
	DependsOn []int
	Claimant  string // if non-empty, sets claim on the task
}

// CreateResult is returned after a successful create.
type CreateResult struct {
	Task *task.Task
	Path string
}

// Create creates a new task. The caller is responsible for:
//   - Acquiring a file lock to prevent concurrent creates
//   - Loading/reloading config to get the current NextID
//
// After Create returns, cfg.NextID is incremented and cfg is saved to disk.
func Create(cfg *config.Config, params CreateParams, now time.Time) (*CreateResult, error) {
	t := &task.Task{
		ID:       cfg.NextID,
		Title:    params.Title,
		Status:   cfg.Defaults.Status,
		Priority: cfg.Defaults.Priority,
		Class:    cfg.Defaults.Class,
		Created:  now,
		Updated:  now,
	}

	// Apply non-zero params, validating against config.
	if err := applyCreateParams(cfg, t, params, now); err != nil {
		return nil, err
	}

	// Validate dependency references.
	if err := validateDeps(cfg, t); err != nil {
		return nil, err
	}

	// WIP limit enforcement (class-aware). For new tasks, the "current"
	// status is empty because the task doesn't exist in any column yet.
	if err := enforceCreateWIP(cfg, t); err != nil {
		return nil, err
	}

	// Generate filename and write.
	slug := task.GenerateSlug(params.Title)
	filename := task.GenerateFilename(t.ID, slug)
	path := filepath.Join(cfg.TasksPath(), filename)
	t.File = path

	if err := task.Write(path, t); err != nil {
		return nil, fmt.Errorf("writing task: %w", err)
	}

	// Increment next_id and save config.
	cfg.NextID++
	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	LogMutation(cfg.Dir(), "create", t.ID, t.Title)

	return &CreateResult{Task: t, Path: path}, nil
}

// applyCreateParams applies non-zero CreateParams fields to the task.
func applyCreateParams(cfg *config.Config, t *task.Task, p CreateParams, now time.Time) error {
	if p.Status != "" {
		if err := task.ValidateStatus(p.Status, cfg.StatusNames()); err != nil {
			return err
		}
		t.Status = p.Status
	}
	if p.Priority != "" {
		if err := task.ValidatePriority(p.Priority, cfg.Priorities); err != nil {
			return err
		}
		t.Priority = p.Priority
	}
	if p.Class != "" {
		if err := task.ValidateClass(p.Class, cfg.ClassNames()); err != nil {
			return err
		}
		t.Class = p.Class
	}
	if p.Assignee != "" {
		t.Assignee = p.Assignee
	}
	if len(p.Tags) > 0 {
		t.Tags = p.Tags
	}
	if p.Body != "" {
		t.Body = p.Body
	}
	if p.Due != nil {
		t.Due = p.Due
	}
	if p.Estimate != "" {
		t.Estimate = p.Estimate
	}
	if p.Parent != nil {
		t.Parent = p.Parent
	}
	if len(p.DependsOn) > 0 {
		t.DependsOn = p.DependsOn
	}
	if p.Claimant != "" {
		t.ClaimedBy = p.Claimant
		t.ClaimedAt = &now
	}
	return nil
}

// validateDeps validates parent and dependency references for a task.
func validateDeps(cfg *config.Config, t *task.Task) error {
	if t.Parent != nil {
		if err := task.ValidateDependencyIDs(cfg.TasksPath(), t.ID, []int{*t.Parent}); err != nil {
			return fmt.Errorf("invalid parent: %w", err)
		}
	}
	if len(t.DependsOn) > 0 {
		if err := task.ValidateDependencyIDs(cfg.TasksPath(), t.ID, t.DependsOn); err != nil {
			return err
		}
	}
	return nil
}

// EditResult is returned after a successful edit.
type EditResult struct {
	Task    *task.Task
	NewPath string
}

// Edit modifies an existing task. It handles find, read, claim validation,
// applying changes via the caller-provided applyFn, post-validation (deps,
// require_claim, WIP limits on status change), write, and logging.
//
// applyFn receives the task to mutate in-place and returns whether any
// changes were made. If no changes were made, Edit returns a NoChanges error.
//
// If release is true, claim checks are bypassed (intent is to release a claim).
func Edit(cfg *config.Config, id int, claimant string, release bool,
	applyFn func(t *task.Task) (changed bool, err error), now time.Time,
) (*EditResult, error) {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return nil, err
	}

	t, err := task.Read(path)
	if err != nil {
		return nil, err
	}

	// Claim validation (release bypasses this).
	if !release {
		if err = task.CheckClaim(t, claimant, cfg.ClaimTimeoutDuration()); err != nil {
			return nil, err
		}
		// Enforce require_claim for the task's current status.
		if cfg.StatusRequiresClaim(t.Status) && claimant == "" {
			return nil, task.ValidateClaimRequired(t.Status)
		}
	}

	oldTitle := t.Title
	oldStatus := t.Status
	wasBlocked := t.Blocked
	wasClaimedBy := t.ClaimedBy

	// Apply caller-provided changes.
	changed, err := applyFn(t)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, clierr.New(clierr.NoChanges, "no changes specified")
	}

	// Post-validation.
	if err = validateEditPost(cfg, t, oldStatus, claimant); err != nil {
		return nil, err
	}

	t.Updated = now

	newPath, err := task.WriteAndRename(path, t, oldTitle)
	if err != nil {
		return nil, err
	}

	// Log transitions.
	LogMutation(cfg.Dir(), "edit", t.ID, t.Title)
	logEditTransitions(cfg, t, wasBlocked, wasClaimedBy)

	return &EditResult{Task: t, NewPath: newPath}, nil
}

// validateEditPost runs post-edit validations: deps, require_claim for new
// status, and WIP limits on status change.
func validateEditPost(cfg *config.Config, t *task.Task, oldStatus, claimant string) error {
	if err := validateDeps(cfg, t); err != nil {
		return err
	}
	// Enforce require_claim if status changed.
	if t.Status != oldStatus && cfg.StatusRequiresClaim(t.Status) && claimant == "" {
		return task.ValidateClaimRequired(t.Status)
	}
	// Check WIP limit if status changed (class-aware).
	if t.Status != oldStatus {
		// Temporarily set old status back for the WIP check, which uses
		// t.Status as the "current" status for the deduction calculation.
		newStatus := t.Status
		t.Status = oldStatus
		err := enforceMoveWIP(cfg, t, newStatus)
		t.Status = newStatus
		return err
	}
	return nil
}

// logEditTransitions logs block/unblock and claim/release transitions.
func logEditTransitions(cfg *config.Config, t *task.Task, wasBlocked bool, wasClaimedBy string) {
	if !wasBlocked && t.Blocked {
		LogMutation(cfg.Dir(), "block", t.ID, t.BlockReason)
	}
	if wasBlocked && !t.Blocked {
		LogMutation(cfg.Dir(), "unblock", t.ID, t.Title)
	}
	if wasClaimedBy == "" && t.ClaimedBy != "" {
		LogMutation(cfg.Dir(), "claim", t.ID, t.ClaimedBy)
	}
	if wasClaimedBy != "" && t.ClaimedBy == "" {
		LogMutation(cfg.Dir(), "release", t.ID, wasClaimedBy)
	}
}

// HandoffParams contains parameters for the Handoff operation.
type HandoffParams struct {
	ID           int
	Claimant     string
	Release      bool
	BlockReason  string
	Note         string
	AddTimestamp bool
}

// Handoff executes the handoff workflow for a task.
func Handoff(cfg *config.Config, params HandoffParams, now time.Time) (*task.Task, error) {
	if params.Claimant == "" {
		return nil, fmt.Errorf("claim name is required")
	}

	path, err := task.FindByID(cfg.TasksPath(), params.ID)
	if err != nil {
		return nil, err
	}

	t, err := task.Read(path)
	if err != nil {
		return nil, err
	}

	// Validate claim ownership.
	if err = task.CheckClaim(t, params.Claimant, cfg.ClaimTimeoutDuration()); err != nil {
		return nil, err
	}

	// Resolve target status: "review" must exist in config.
	const reviewStatus = "review"
	if err = task.ValidateStatus(reviewStatus, cfg.StatusNames()); err != nil {
		return nil, fmt.Errorf("board has no 'review' status; add one to use handoff")
	}

	// Move to review (skip if already there).
	oldStatus := t.Status
	if t.Status != reviewStatus {
		// Enforce require_claim for review.
		if cfg.StatusRequiresClaim(reviewStatus) && params.Claimant == "" {
			return nil, task.ValidateClaimRequired(reviewStatus)
		}
		if err = enforceClassWIP(cfg, t, reviewStatus); err != nil {
			return nil, err
		}
		t.Status = reviewStatus
		task.UpdateTimestamps(t, oldStatus, reviewStatus, cfg)
	}

	// Apply claim (refresh).
	t.ClaimedBy = params.Claimant
	t.ClaimedAt = &now

	// Optionally block.
	if params.BlockReason != "" {
		t.Blocked = true
		t.BlockReason = params.BlockReason
	}

	// Append note.
	if params.Note != "" {
		t.Body = AppendBody(t.Body, params.Note, params.AddTimestamp)
	}

	// Release claim if requested.
	if params.Release {
		t.ClaimedBy = ""
		t.ClaimedAt = nil
	}

	t.Updated = now

	if err = task.Write(path, t); err != nil {
		return nil, fmt.Errorf("writing task: %w", err)
	}

	// Log activity.
	if oldStatus != t.Status {
		LogMutation(cfg.Dir(), "move", t.ID, oldStatus+" -> "+t.Status)
	}
	LogMutation(cfg.Dir(), "handoff", t.ID, t.Title)
	if t.Blocked {
		LogMutation(cfg.Dir(), "block", t.ID, t.BlockReason)
	}
	if t.ClaimedBy == "" {
		LogMutation(cfg.Dir(), "release", t.ID, t.Title)
	}

	return t, nil
}

// AppendBody appends text to the existing body, optionally prefixed with a timestamp line.
func AppendBody(existing, text string, addTimestamp bool) string {
	var b strings.Builder
	if existing != "" {
		b.WriteString(strings.TrimRight(existing, "\n"))
		b.WriteString("\n\n")
	}
	if addTimestamp {
		b.WriteString(time.Now().Format("[[2006-01-02]] Mon 15:04"))
		b.WriteByte('\n')
	}
	b.WriteString(text)
	return b.String()
}

// PickAndClaimParams contains parameters for the PickAndClaim operation.
type PickAndClaimParams struct {
	Claimant     string
	StatusFilter string
	MoveTarget   string
	Tags         []string
}

// PickAndClaim finds the highest-priority task and atomically claims it.
func PickAndClaim(cfg *config.Config, params PickAndClaimParams, now time.Time) (*task.Task, string, error) {
	allTasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return nil, "", err
	}
	// Warnings are discarded or we should print them. Better to just print or ignore since CLI did printWarnings.
	// We'll ignore warnings here or log them.
	_ = warnings

	opts := PickOptions{
		ClaimTimeout: cfg.ClaimTimeoutDuration(),
		Tags:         params.Tags,
	}
	if params.StatusFilter != "" {
		opts.Statuses = []string{params.StatusFilter}
	}

	picked := Pick(cfg, allTasks, opts)
	if picked == nil {
		return nil, "", clierr.New(clierr.NothingToPick, "no unblocked, unclaimed tasks found")
	}

	// Claim the task.
	picked.ClaimedBy = params.Claimant
	picked.ClaimedAt = &now

	// Optionally move the task.
	oldStatus := ""
	if params.MoveTarget != "" && picked.Status != params.MoveTarget {
		if err := enforceClassWIP(cfg, picked, params.MoveTarget); err != nil {
			return nil, "", err
		}
		oldStatus = picked.Status
		task.UpdateTimestamps(picked, oldStatus, params.MoveTarget, cfg)
		picked.Status = params.MoveTarget
	}

	picked.Updated = now

	// Write the task back.
	path, err := task.FindByID(cfg.TasksPath(), picked.ID)
	if err != nil {
		return nil, "", err
	}
	if err = task.Write(path, picked); err != nil {
		return nil, "", fmt.Errorf("writing task: %w", err)
	}

	LogMutation(cfg.Dir(), "claim", picked.ID, params.Claimant)
	if oldStatus != "" {
		LogMutation(cfg.Dir(), "move", picked.ID, oldStatus+" -> "+picked.Status)
	}

	return picked, oldStatus, nil
}
