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
	moveCmd.Flags().BoolP("force", "f", false, "override WIP limit")
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
	force, _ := cmd.Flags().GetBool("force")
	return runBatch(ids, func(id int) error {
		return moveSingleCore(cfg, id, cmd, args, force)
	})
}

// moveSingleTask handles a single task move with full output.
func moveSingleTask(cfg *config.Config, id int, cmd *cobra.Command, args []string) error {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	newStatus, err := resolveTargetStatus(cmd, args, t, cfg)
	if err != nil {
		return err
	}

	// Idempotent: if already at target status, succeed without writing.
	if t.Status == newStatus {
		return outputMoveResult(t, false)
	}

	// Check WIP limit.
	force, _ := cmd.Flags().GetBool("force")
	if err := enforceWIPLimit(cfg, t.Status, newStatus, force); err != nil {
		return err
	}

	// Warn when moving a blocked task.
	if t.Blocked {
		fmt.Fprintf(os.Stderr, "Warning: task #%d is blocked (%s)\n", t.ID, t.BlockReason)
	}

	oldStatus := t.Status
	t.Status = newStatus
	now := time.Now()
	updateTimestamps(t, oldStatus, newStatus, cfg)
	t.Updated = now

	if err := task.Write(path, t); err != nil {
		return fmt.Errorf("writing task: %w", err)
	}

	logActivity(cfg, "move", id, oldStatus+" -> "+newStatus)

	if outputFormat() == output.FormatJSON {
		return outputMoveResult(t, true)
	}

	output.Messagef("Moved task #%d: %s → %s", id, oldStatus, newStatus)
	return nil
}

// moveSingleCore performs the core move logic without output (for batch mode).
func moveSingleCore(cfg *config.Config, id int, cmd *cobra.Command, args []string, force bool) error {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	newStatus, err := resolveTargetStatus(cmd, args, t, cfg)
	if err != nil {
		return err
	}

	if t.Status == newStatus {
		return nil // idempotent
	}

	if err := enforceWIPLimit(cfg, t.Status, newStatus, force); err != nil {
		return err
	}

	oldStatus := t.Status
	t.Status = newStatus
	updateTimestamps(t, oldStatus, newStatus, cfg)
	t.Updated = time.Now()

	if err := task.Write(path, t); err != nil {
		return fmt.Errorf("writing task: %w", err)
	}

	logActivity(cfg, "move", id, oldStatus+" -> "+newStatus)
	return nil
}

func resolveTargetStatus(cmd *cobra.Command, args []string, t *task.Task, cfg *config.Config) (string, error) {
	next, _ := cmd.Flags().GetBool("next")
	prev, _ := cmd.Flags().GetBool("prev")

	switch {
	case len(args) == 2: //nolint:mnd // positional arg
		status := args[1]
		if err := task.ValidateStatus(status, cfg.Statuses); err != nil {
			return "", err
		}
		return status, nil
	case next:
		idx := cfg.StatusIndex(t.Status)
		if idx < 0 || idx >= len(cfg.Statuses)-1 {
			return "", task.ValidateBoundaryError(t.ID, t.Status, "last")
		}
		return cfg.Statuses[idx+1], nil
	case prev:
		idx := cfg.StatusIndex(t.Status)
		if idx <= 0 {
			return "", task.ValidateBoundaryError(t.ID, t.Status, "first")
		}
		return cfg.Statuses[idx-1], nil
	default:
		return "", clierr.New(clierr.InvalidInput, "provide a target status or use --next/--prev")
	}
}

// enforceWIPLimit checks if the target status has room. If force is true,
// it warns instead of erroring.
func enforceWIPLimit(cfg *config.Config, currentStatus, targetStatus string, force bool) error {
	limit := cfg.WIPLimit(targetStatus)
	if limit == 0 {
		return nil
	}

	allTasks, err := task.ReadAll(cfg.TasksPath())
	if err != nil {
		return fmt.Errorf("reading tasks for WIP check: %w", err)
	}

	counts := board.CountByStatus(allTasks)
	if err := checkWIPLimit(cfg, counts, targetStatus, currentStatus); err != nil {
		if force {
			fmt.Fprintf(os.Stderr, "Warning: %s (overridden with --force)\n", err)
			return nil
		}
		return err
	}
	return nil
}

// moveResult wraps a task with a changed flag for JSON output.
type moveResult struct {
	*task.Task
	Changed bool `json:"changed"`
}

// updateTimestamps sets Started and Completed based on the status transition.
func updateTimestamps(t *task.Task, oldStatus, newStatus string, cfg *config.Config) {
	now := time.Now()
	initialStatus := cfg.Statuses[0]

	// Set Started on first move out of initial status (never overwrite).
	if t.Started == nil && oldStatus == initialStatus && newStatus != initialStatus {
		t.Started = &now
	}

	// Set/clear Completed based on terminal status.
	if cfg.IsTerminalStatus(newStatus) {
		t.Completed = &now
		// Direct move to terminal: also set Started if nil.
		if t.Started == nil {
			t.Started = &now
		}
	} else if cfg.IsTerminalStatus(oldStatus) {
		// Reopening: clear Completed, preserve Started.
		t.Completed = nil
	}
}

func outputMoveResult(t *task.Task, changed bool) error {
	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(moveResult{Task: t, Changed: changed})
	}
	if !changed {
		output.Messagef("Task #%d is already at %s", t.ID, t.Status)
	}
	return nil
}
