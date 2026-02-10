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

var pickCmd = &cobra.Command{
	Use:   "pick",
	Short: "Pick the next available task",
	Long: `Atomically finds the highest-priority unclaimed, unblocked task and claims it.
Replaces the multi-step list/edit/move pattern with a single command.`,
	RunE: runPick,
}

func init() {
	pickCmd.Flags().String("claim", "", "agent name to claim as (required)")
	pickCmd.Flags().String("status", "", "status column to pick from (default: all non-terminal)")
	pickCmd.Flags().String("move", "", "also move the picked task to this status")
	pickCmd.Flags().StringSlice("tags", nil, "filter by tags (comma-separated, OR logic)")
	_ = pickCmd.MarkFlagRequired("claim")
	rootCmd.AddCommand(pickCmd)
}

func runPick(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	claimant, _ := cmd.Flags().GetString("claim")
	statusFilter, _ := cmd.Flags().GetString("status")
	moveTarget, _ := cmd.Flags().GetString("move")
	tags, _ := cmd.Flags().GetStringSlice("tags")

	if err = validatePickFlags(cfg, statusFilter, moveTarget); err != nil {
		return err
	}

	picked, oldStatus, err := executePick(cfg, claimant, statusFilter, moveTarget, tags)
	if err != nil {
		return err
	}

	return outputPickResult(picked, oldStatus, claimant)
}

func validatePickFlags(cfg *config.Config, statusFilter, moveTarget string) error {
	if statusFilter != "" {
		if err := task.ValidateStatus(statusFilter, cfg.StatusNames()); err != nil {
			return err
		}
	}
	if moveTarget != "" {
		if err := task.ValidateStatus(moveTarget, cfg.StatusNames()); err != nil {
			return err
		}
	}
	return nil
}

func executePick(cfg *config.Config, claimant, statusFilter, moveTarget string, tags []string) (*task.Task, string, error) {
	allTasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return nil, "", err
	}
	printWarnings(warnings)

	opts := board.PickOptions{
		ClaimTimeout: cfg.ClaimTimeoutDuration(),
		Tags:         tags,
	}
	if statusFilter != "" {
		opts.Statuses = []string{statusFilter}
	}

	picked := board.Pick(cfg, allTasks, opts)
	if picked == nil {
		return nil, "", clierr.New(clierr.NothingToPick, "no unblocked, unclaimed tasks found")
	}

	// Claim the task.
	now := time.Now()
	picked.ClaimedBy = claimant
	picked.ClaimedAt = &now

	// Optionally move the task.
	oldStatus := ""
	if moveTarget != "" && picked.Status != moveTarget {
		if moveErr := enforceWIPLimit(cfg, picked.Status, moveTarget); moveErr != nil {
			return nil, "", moveErr
		}
		oldStatus = picked.Status
		task.UpdateTimestamps(picked, oldStatus, moveTarget, cfg)
		picked.Status = moveTarget
	}

	picked.Updated = time.Now()

	// Write the task back.
	path, err := task.FindByID(cfg.TasksPath(), picked.ID)
	if err != nil {
		return nil, "", err
	}
	if err = task.Write(path, picked); err != nil {
		return nil, "", fmt.Errorf("writing task: %w", err)
	}

	logActivity(cfg, "claim", picked.ID, claimant)
	if oldStatus != "" {
		logActivity(cfg, "move", picked.ID, oldStatus+" -> "+picked.Status)
	}

	return picked, oldStatus, nil
}

func outputPickResult(picked *task.Task, oldStatus, claimant string) error {
	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, picked)
	}
	if oldStatus != "" {
		output.Messagef(os.Stdout, "Picked and moved task #%d: %s (%s -> %s, claimed by %s)",
			picked.ID, picked.Title, oldStatus, picked.Status, claimant)
	} else {
		output.Messagef(os.Stdout, "Picked task #%d: %s (%s, claimed by %s)",
			picked.ID, picked.Title, picked.Status, claimant)
	}
	return nil
}
