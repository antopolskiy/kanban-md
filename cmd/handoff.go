package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var handoffCmd = &cobra.Command{
	Use:   "handoff ID",
	Short: "Hand off a task (move to review with notes)",
	Long: `Moves a task to review status, appends a handoff note, and optionally
blocks the task and/or releases the claim. Designed for multi-agent workflows
where standardized handoffs prevent information loss.`,
	Args: cobra.ExactArgs(1),
	RunE: runHandoff,
}

func init() {
	handoffCmd.Flags().String("claim", "", "claim task for an agent (required)")
	handoffCmd.Flags().String("note", "", "handoff note to append to body")
	handoffCmd.Flags().BoolP("timestamp", "t", false, "prefix a timestamp line to the note")
	handoffCmd.Flags().String("block", "", "mark task as blocked with reason")
	handoffCmd.Flags().Bool("release", false, "release claim after handoff")
	rootCmd.AddCommand(handoffCmd)
}

func runHandoff(cmd *cobra.Command, args []string) error {
	ids, err := parseIDs(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(ids) == 1 {
		return handoffSingleTask(cfg, ids[0], cmd)
	}

	return runBatch(ids, func(id int) error {
		_, err := executeHandoff(cfg, id, cmd)
		return err
	})
}

func handoffSingleTask(cfg *config.Config, id int, cmd *cobra.Command) error {
	t, err := executeHandoff(cfg, id, cmd)
	if err != nil {
		return err
	}

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, t)
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Handed off task #%d -> review", t.ID))
	if t.Blocked {
		parts = append(parts, fmt.Sprintf("(blocked: %s)", t.BlockReason))
	}
	if t.ClaimedBy == "" {
		parts = append(parts, "(claim released)")
	}
	output.Messagef(os.Stdout, "%s", strings.Join(parts, " "))
	return nil
}

func executeHandoff(cfg *config.Config, id int, cmd *cobra.Command) (*task.Task, error) {
	claimant, _ := cmd.Flags().GetString("claim")
	release, _ := cmd.Flags().GetBool("release")
	blockReason, _ := cmd.Flags().GetString("block")
	note, _ := cmd.Flags().GetString("note")
	addTimestamp, _ := cmd.Flags().GetBool("timestamp")

	if claimant == "" {
		return nil, clierr.New(clierr.InvalidInput, "claim name is required (use --claim NAME)")
	}

	if cmd.Flags().Changed("block") && blockReason == "" {
		return nil, clierr.New(clierr.InvalidInput, "block reason is required (use --block REASON)")
	}

	params := board.HandoffParams{
		ID:           id,
		Claimant:     claimant,
		Release:      release,
		BlockReason:  blockReason,
		Note:         note,
		AddTimestamp: addTimestamp,
	}

	return board.Handoff(cfg, params, time.Now())
}
