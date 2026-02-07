// Package cmd implements the kanban-md CLI commands.
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

// version is set at build time via ldflags.
var version = "dev"

// Global flags.
var (
	flagJSON    bool
	flagTable   bool
	flagDir     string
	flagNoColor bool
)

var rootCmd = &cobra.Command{
	Use:   "kanban-md",
	Short: "A file-based Kanban tool powered by Markdown",
	Long: `kanban-md is a CLI tool for managing Kanban boards using plain Markdown files.
Tasks are stored as individual files with YAML frontmatter, making them
easy to read, edit, and version-control. Designed for AI agents and humans alike.`,
	Version: version,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		if flagNoColor || os.Getenv("NO_COLOR") != "" {
			output.DisableColor()
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolVar(&flagTable, "table", false, "output as table")
	rootCmd.PersistentFlags().StringVar(&flagDir, "dir", "", "path to kanban directory")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable color output")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadConfig finds and loads the kanban config.
func loadConfig() (*config.Config, error) {
	if flagDir != "" {
		return config.Load(flagDir)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	dir, err := config.FindDir(cwd)
	if err != nil {
		return nil, err
	}

	return config.Load(dir)
}

// outputFormat returns the detected output format from flags/env/TTY.
func outputFormat() output.Format {
	return output.Detect(flagJSON, flagTable)
}

// validateDepIDs checks that all dependency IDs exist and none are self-referencing.
func validateDepIDs(tasksDir string, selfID int, ids []int) error {
	for _, depID := range ids {
		if depID == selfID {
			return fmt.Errorf("task cannot depend on itself (ID %d)", depID)
		}
		if _, err := task.FindByID(tasksDir, depID); err != nil {
			return fmt.Errorf("dependency task #%d not found", depID)
		}
	}
	return nil
}

// checkWIPLimit verifies that adding a task to targetStatus would not exceed
// the WIP limit. currentTaskStatus is the task's current status (empty for new tasks).
// Returns nil if within limits, or an error describing the violation.
func checkWIPLimit(cfg *config.Config, statusCounts map[string]int, targetStatus, currentTaskStatus string) error {
	limit := cfg.WIPLimit(targetStatus)
	if limit == 0 {
		return nil
	}

	count := statusCounts[targetStatus]

	// If the task is already in the target status, it doesn't add to the count.
	if currentTaskStatus == targetStatus {
		return nil
	}

	if count >= limit {
		return fmt.Errorf("WIP limit reached for %q (%d/%d)", targetStatus, count, limit)
	}
	return nil
}

// logActivity appends an entry to the activity log. Errors are warned to
// stderr but never fail the command.
func logActivity(cfg *config.Config, action string, taskID int, detail string) {
	entry := board.LogEntry{
		Timestamp: time.Now(),
		Action:    action,
		TaskID:    taskID,
		Detail:    detail,
	}
	if err := board.AppendLog(cfg.Dir(), entry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write activity log: %v\n", err)
	}
}
