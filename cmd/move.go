package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

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

	next, _ := cmd.Flags().GetBool("next")
	prev, _ := cmd.Flags().GetBool("prev")

	var newStatus string

	switch {
	case len(args) == 2: //nolint:mnd // positional arg
		newStatus = args[1]
		if err := task.ValidateStatus(newStatus, cfg.Statuses); err != nil {
			return err
		}
	case next:
		idx := cfg.StatusIndex(t.Status)
		if idx < 0 || idx >= len(cfg.Statuses)-1 {
			return fmt.Errorf("task #%d is already at the last status (%s)", id, t.Status)
		}
		newStatus = cfg.Statuses[idx+1]
	case prev:
		idx := cfg.StatusIndex(t.Status)
		if idx <= 0 {
			return fmt.Errorf("task #%d is already at the first status (%s)", id, t.Status)
		}
		newStatus = cfg.Statuses[idx-1]
	default:
		return errors.New("provide a target status or use --next/--prev")
	}

	oldStatus := t.Status
	t.Status = newStatus
	t.Updated = time.Now()

	if err := task.Write(path, t); err != nil {
		return fmt.Errorf("writing task: %w", err)
	}

	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(t)
	}

	output.Messagef("Moved task #%d: %s → %s", id, oldStatus, newStatus)
	return nil
}
