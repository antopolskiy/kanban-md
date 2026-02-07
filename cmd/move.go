package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var moveCmd = &cobra.Command{
	Use:   "move ID [STATUS]",
	Short: "Move a task to a different status",
	Long: `Changes the status of a task. Provide the new status directly,
or use --next/--prev to move along the configured status order.`,
	Args: cobra.RangeArgs(1, 2), //nolint:mnd // 1 or 2 positional args
	RunE: runMove,
}

func init() {
	moveCmd.Flags().Bool("next", false, "move to next status")
	moveCmd.Flags().Bool("prev", false, "move to previous status")
	rootCmd.AddCommand(moveCmd)
}

func runMove(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid task ID %q: %w", args[0], err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

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

	// Warn when moving a blocked task.
	if t.Blocked {
		fmt.Fprintf(os.Stderr, "Warning: task #%d is blocked (%s)\n", t.ID, t.BlockReason)
	}

	oldStatus := t.Status
	t.Status = newStatus
	t.Updated = time.Now()

	if err := task.Write(path, t); err != nil {
		return fmt.Errorf("writing task: %w", err)
	}

	if outputFormat() == output.FormatJSON {
		return outputMoveResult(t, true)
	}

	output.Messagef("Moved task #%d: %s → %s", id, oldStatus, newStatus)
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
			return "", fmt.Errorf("task #%d is already at the last status (%s)", t.ID, t.Status)
		}
		return cfg.Statuses[idx+1], nil
	case prev:
		idx := cfg.StatusIndex(t.Status)
		if idx <= 0 {
			return "", fmt.Errorf("task #%d is already at the first status (%s)", t.ID, t.Status)
		}
		return cfg.Statuses[idx-1], nil
	default:
		return "", errors.New("provide a target status or use --next/--prev")
	}
}

// moveResult wraps a task with a changed flag for JSON output.
type moveResult struct {
	*task.Task
	Changed bool `json:"changed"`
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
