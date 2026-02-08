// Package cmd implements the kanban-md CLI commands.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
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
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
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
	_, err := rootCmd.ExecuteC()
	if err == nil {
		return
	}

	// Handle SilentError — exit with code, no output.
	var silent *clierr.SilentError
	if errors.As(err, &silent) {
		os.Exit(silent.Code)
	}

	// Determine if JSON mode is active.
	jsonMode := flagJSON
	if !jsonMode {
		jsonMode = os.Getenv("KANBAN_OUTPUT") == "json"
	}

	if jsonMode {
		var cliErr *clierr.Error
		if errors.As(err, &cliErr) {
			output.JSONError(os.Stdout, cliErr.Code, cliErr.Message, cliErr.Details)
			os.Exit(cliErr.ExitCode())
		}
		// Unknown error — wrap as INTERNAL_ERROR.
		output.JSONError(os.Stdout, clierr.InternalError, err.Error(), nil)
		os.Exit(2) //nolint:mnd // exit code 2 for internal errors
	}

	// Non-JSON mode: print to stderr.
	fmt.Fprintln(os.Stderr, err)
	var cliErr *clierr.Error
	if errors.As(err, &cliErr) {
		os.Exit(cliErr.ExitCode())
	}
	os.Exit(1)
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
			return task.ValidateSelfReference(depID)
		}
		if _, err := task.FindByID(tasksDir, depID); err != nil {
			return task.ValidateDependencyNotFound(depID)
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
		return task.ValidateWIPLimit(targetStatus, limit, count)
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

// parseIDs splits a comma-separated ID string into deduplicated int IDs.
func parseIDs(arg string) ([]int, error) {
	parts := strings.Split(arg, ",")
	seen := make(map[int]bool, len(parts))
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, task.ValidateTaskID(p)
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return nil, clierr.New(clierr.InvalidTaskID, "no valid task IDs provided")
	}
	return ids, nil
}

// runBatch executes fn for each ID and collects results. Returns a SilentError
// with exit code 1 if any operation failed (after outputting results).
func runBatch(ids []int, fn func(int) error) error {
	results := make([]output.BatchResult, 0, len(ids))
	anyFailed := false

	for _, id := range ids {
		err := fn(id)
		if err != nil {
			anyFailed = true
			var cliErr *clierr.Error
			if errors.As(err, &cliErr) {
				results = append(results, output.BatchResult{ID: id, OK: false, Error: cliErr.Message, Code: cliErr.Code})
			} else {
				results = append(results, output.BatchResult{ID: id, OK: false, Error: err.Error()})
			}
		} else {
			results = append(results, output.BatchResult{ID: id, OK: true})
		}
	}

	if outputFormat() == output.FormatJSON {
		if err := output.JSON(os.Stdout, results); err != nil {
			return err
		}
	} else {
		var succeeded int
		for _, r := range results {
			if r.OK {
				succeeded++
			} else {
				fmt.Fprintf(os.Stderr, "Error: task #%d: %s\n", r.ID, r.Error)
			}
		}
		output.Messagef(os.Stdout, "Completed %d/%d operations", succeeded, len(ids))
	}

	if anyFailed {
		return &clierr.SilentError{Code: 1}
	}
	return nil
}
