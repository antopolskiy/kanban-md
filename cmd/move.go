package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var moveCmd = &cobra.Command{
	Use:   "move ID[,ID,...] [STATUS]",
	Short: "Move a task to a different status",
	Long: `Changes the status of a task. Provide the new status directly,
or use --next/--prev to move along the configured status order.
Multiple IDs can be provided as a comma-separated list.`,
	Args: cobra.RangeArgs(1, 2), //nolint:mnd // 1 or 2 positional args
	RunE: runMove,
}

func init() {
	moveCmd.Flags().Bool("next", false, "move to next status")
	moveCmd.Flags().Bool("prev", false, "move to previous status")
	moveCmd.Flags().String("claim", "", "claim task for an agent during move")
	rootCmd.AddCommand(moveCmd)
}

func runMove(cmd *cobra.Command, args []string) error {
	ids, err := parseIDs(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Single ID: preserve exact current behavior.
	if len(ids) == 1 {
		return moveSingleTask(cfg, ids[0], cmd, args)
	}

	// Batch mode.
	return runBatch(ids, func(id int) error {
		_, _, err := executeMove(cfg, id, cmd, args)
		return err
	})
}

// moveResult wraps a task with a changed flag for JSON output.
type moveResult struct {
	*task.Task
	Changed bool `json:"changed"`
}

// moveSingleTask handles a single task move with full output.
func moveSingleTask(cfg *config.Config, id int, cmd *cobra.Command, args []string) error {
	t, oldStatus, err := executeMove(cfg, id, cmd, args)
	if err != nil {
		return err
	}

	// Idempotent: status didn't change.
	if oldStatus == "" {
		return outputMoveResult(t, false)
	}

	if outputFormat() == output.FormatJSON {
		return outputMoveResult(t, true)
	}

	output.Messagef(os.Stdout, "Moved task #%d: %s -> %s", id, oldStatus, t.Status)
	return nil
}

// executeMove performs the core move: find, read, resolve, wip check, write, log.
// Returns (task, oldStatus, error). If the task was already at the target status
// (idempotent), oldStatus is empty and the task is returned unchanged.
func executeMove(cfg *config.Config, id int, cmd *cobra.Command, args []string) (*task.Task, string, error) {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return nil, "", err
	}

	t, err := task.Read(path)
	if err != nil {
		return nil, "", err
	}

	claimant, _ := cmd.Flags().GetString("claim")
	if err = validateMoveClaim(cfg, t, claimant); err != nil {
		return nil, "", err
	}

	newStatus, err := resolveTargetStatus(cmd, args, t, cfg)
	if err != nil {
		return nil, "", err
	}

	// Idempotent: if already at target status, succeed without writing.
	if t.Status == newStatus {
		return t, "", nil
	}

	// Enforce require_claim for target status.
	if cfg.StatusRequiresClaim(newStatus) && claimant == "" {
		return nil, "", task.ValidateClaimRequired(newStatus)
	}

	if err = enforceMoveWIP(cfg, t, newStatus); err != nil {
		return nil, "", err
	}

	// Warn when moving a blocked task.
	if t.Blocked {
		fmt.Fprintf(os.Stderr, "Warning: task #%d is blocked (%s)\n", t.ID, t.BlockReason)
	}

	oldStatus := t.Status
	t.Status = newStatus
	task.UpdateTimestamps(t, oldStatus, newStatus, cfg)
	applyMoveClaim(cmd, t, claimant)
	t.Updated = time.Now()

	if err := task.Write(path, t); err != nil {
		return nil, "", fmt.Errorf("writing task: %w", err)
	}

	logActivity(cfg, "move", id, oldStatus+" -> "+newStatus)
	return t, oldStatus, nil
}

// validateMoveClaim checks claim ownership before allowing a move.
func validateMoveClaim(cfg *config.Config, t *task.Task, claimant string) error {
	return checkClaim(t, claimant, cfg.ClaimTimeoutDuration())
}

// enforceMoveWIP checks WIP limits, considering class of service.
func enforceMoveWIP(cfg *config.Config, t *task.Task, newStatus string) error {
	if t.Class != "" && len(cfg.Classes) > 0 {
		return enforceWIPLimitForClass(cfg, t, t.Status, newStatus)
	}
	return enforceWIPLimit(cfg, t.Status, newStatus)
}

// applyMoveClaim sets the claim on the task if --claim flag was provided.
func applyMoveClaim(cmd *cobra.Command, t *task.Task, claimant string) {
	if cmd.Flags().Changed("claim") && claimant != "" {
		now := time.Now()
		t.ClaimedBy = claimant
		t.ClaimedAt = &now
	}
}

func resolveTargetStatus(cmd *cobra.Command, args []string, t *task.Task, cfg *config.Config) (string, error) {
	next, _ := cmd.Flags().GetBool("next")
	prev, _ := cmd.Flags().GetBool("prev")

	switch {
	case len(args) == 2: //nolint:mnd // positional arg
		status := args[1] //nolint:gosec // args length checked by case guard
		if err := task.ValidateStatus(status, cfg.StatusNames()); err != nil {
			return "", err
		}
		return status, nil
	case next:
		names := cfg.StatusNames()
		idx := cfg.StatusIndex(t.Status)
		if idx < 0 || idx >= len(names)-1 {
			return "", task.ValidateBoundaryError(t.ID, t.Status, "last")
		}
		return names[idx+1], nil
	case prev:
		names := cfg.StatusNames()
		idx := cfg.StatusIndex(t.Status)
		if idx <= 0 {
			return "", task.ValidateBoundaryError(t.ID, t.Status, "first")
		}
		return names[idx-1], nil
	default:
		return "", clierr.New(clierr.InvalidInput, "provide a target status or use --next/--prev")
	}
}

// enforceWIPLimit checks if the target status has room.
func enforceWIPLimit(cfg *config.Config, currentStatus, targetStatus string) error {
	limit := cfg.WIPLimit(targetStatus)
	if limit == 0 {
		return nil
	}

	allTasks, _, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return fmt.Errorf("reading tasks for WIP check: %w", err)
	}

	counts := board.CountByStatus(allTasks)
	return checkWIPLimit(cfg, counts, targetStatus, currentStatus)
}

// enforceWIPLimitForClass checks WIP limits considering class of service.
// Expedite tasks bypass column WIP limits but have their own board-wide limit.
func enforceWIPLimitForClass(cfg *config.Config, t *task.Task, currentStatus, targetStatus string) error {
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
	return enforceWIPLimit(cfg, currentStatus, targetStatus)
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

func outputMoveResult(t *task.Task, changed bool) error {
	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(os.Stdout, moveResult{Task: t, Changed: changed})
	}
	if !changed {
		output.Messagef(os.Stdout, "Task #%d is already at %s", t.ID, t.Status)
	}
	return nil
}
