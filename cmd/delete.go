package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var deleteCmd = &cobra.Command{
	Use:     "delete ID[,ID,...]",
	Aliases: []string{"rm"},
	Short:   "Soft-delete a task",
	Long: `Soft-deletes a task by moving it to archived status. Prompts for confirmation in interactive mode.
Multiple IDs can be provided as a comma-separated list (requires --yes).`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	ids, err := parseIDs(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	yes, _ := cmd.Flags().GetBool("yes")

	// Batch mode requires --yes.
	if len(ids) > 1 && !yes {
		return clierr.New(clierr.ConfirmationReq,
			"batch delete requires --yes")
	}

	// Single ID: preserve exact current behavior.
	if len(ids) == 1 {
		return deleteSingleTask(cfg, ids[0], yes)
	}

	// Batch mode (yes is guaranteed true here).
	return runBatch(ids, func(id int) error {
		return executeDelete(cfg, id)
	})
}

// deleteSingleTask handles a single task delete with confirmation and output.
func deleteSingleTask(cfg *config.Config, id int, yes bool) error {
	// Pre-read the task for confirmation prompt (before the actual delete).
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}
	t, err := task.Read(path)
	if err != nil {
		return err
	}

	// Require confirmation in TTY mode unless --yes.
	if !yes {
		if !term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // Fd returns uintptr, int cast is safe for terminal check
			return clierr.New(clierr.ConfirmationReq,
				"cannot prompt for confirmation (not a terminal); use --yes")
		}
		fmt.Fprintf(os.Stderr, "Delete task #%d %q? [y/N] ", t.ID, t.Title)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, "Canceled.")
			return nil
		}
	}

	result, err := board.Delete(cfg, id, "", time.Now())
	if err != nil {
		return err
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, map[string]interface{}{
			"status": "deleted",
			"id":     result.Task.ID,
			"title":  result.Task.Title,
		})
	}

	output.Messagef(os.Stdout, "Deleted task #%d: %s", result.Task.ID, result.Task.Title)
	return nil
}

// executeDelete performs the core delete via the shared board.Delete.
func executeDelete(cfg *config.Config, id int) error {
	result, err := board.Delete(cfg, id, "", time.Now())
	if err != nil {
		return err
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}
	return nil
}

// softDeleteAndLog archives the task and logs the delete action.
// Kept for test compatibility; production paths use board.Delete directly.
func softDeleteAndLog(cfg *config.Config, _ string, t *task.Task) error {
	result, err := board.Delete(cfg, t.ID, "", time.Now())
	if err != nil {
		return err
	}
	// Update the caller's task pointer to reflect the archived state.
	*t = *result.Task
	return nil
}

func warnDependents(tasksDir string, id int) {
	dependents := board.FindDependents(tasksDir, id)
	for _, msg := range dependents {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
	}
}
