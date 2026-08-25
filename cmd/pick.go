package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var pickCmd = &cobra.Command{
	Use:   "pick",
	Short: "Pick the next available task",
	Long: `Finds the highest-priority unclaimed, unblocked task and claims it in one command.
Claims are cooperative leases; this replaces the manual list/edit/move sequence.`,
	RunE: runPick,
}

func init() {
	pickCmd.Flags().String("claim", "", "agent name to claim as (required)")
	pickCmd.Flags().String("status", "", "status column to pick from (default: all non-terminal)")
	pickCmd.Flags().String("move", "", "also move the picked task to this status")
	pickCmd.Flags().StringSlice("tags", nil, "filter by tags (comma-separated, OR logic)")
	pickCmd.Flags().Int("parent", 0, "filter to children of this parent task ID")
	pickCmd.Flags().Bool("no-body", false, "suppress full task details after pick")
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
	noBody, _ := cmd.Flags().GetBool("no-body")

	var parent *int
	if cmd.Flags().Changed("parent") {
		v, _ := cmd.Flags().GetInt("parent")
		parent = &v
	}

	if err = validatePickFlags(cfg, statusFilter, moveTarget); err != nil {
		return err
	}

	picked, oldStatus, err := executePick(cfg, claimant, statusFilter, moveTarget, tags, parent)
	if err != nil {
		return err
	}

	return outputPickResultWithFields(picked, oldStatus, claimant, noBody, cfg.CompactFields())
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

func executePick(cfg *config.Config, claimant, statusFilter, moveTarget string, tags []string, parent *int) (*task.Task, string, error) {
	params := board.PickAndClaimParams{
		Claimant:     claimant,
		StatusFilter: statusFilter,
		MoveTarget:   moveTarget,
		Tags:         tags,
		Parent:       parent,
	}

	picked, oldStatus, warnings, err := board.PickAndClaim(cfg, params, time.Now())
	printWarnings(warnings)
	return picked, oldStatus, err
}

func outputPickResult(picked *task.Task, oldStatus, claimant string, noBody bool) error { //nolint:unparam // default adapter retained for command tests
	return outputPickResultWithFields(picked, oldStatus, claimant, noBody, nil)
}

func outputPickResultWithFields(picked *task.Task, oldStatus, claimant string, noBody bool, fields []string) error {
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
	if noBody {
		return nil
	}
	if _, err := fmt.Fprintln(os.Stdout); err != nil {
		return err
	}
	if outputFormat() == output.FormatCompact {
		output.TaskDetailCompactWithOptions(os.Stdout, picked, board.ChildSummary{}, output.TaskViewOptions{CompactFields: fields})
		return nil
	}
	return outputTaskDetail(picked)
}
