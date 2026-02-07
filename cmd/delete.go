package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
	Short:   "Delete a task",
	Long: `Deletes a task file. Prompts for confirmation in interactive mode.
Multiple IDs can be provided as a comma-separated list (requires --force).`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
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

	force, _ := cmd.Flags().GetBool("force")

	// Batch mode requires --force.
	if len(ids) > 1 && !force {
		return clierr.New(clierr.ConfirmationReq,
			"batch delete requires --force")
	}

	// Single ID: preserve exact current behavior.
	if len(ids) == 1 {
		return deleteSingleTask(cfg, ids[0], force)
	}

	// Batch mode (force is guaranteed true here).
	return runBatch(ids, func(id int) error {
		return deleteSingleCore(cfg, id)
	})
}

// deleteSingleTask handles a single task delete with full output and prompting.
func deleteSingleTask(cfg *config.Config, id int, force bool) error {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	// Warn if other tasks reference this one as a dependency or parent.
	warnDependents(cfg.TasksPath(), t.ID)

	// Require confirmation in TTY mode unless --force.
	if !force {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return clierr.New(clierr.ConfirmationReq,
				"cannot prompt for confirmation (not a terminal); use --force")
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

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting task file: %w", err)
	}

	logActivity(cfg, "delete", t.ID, t.Title)

	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(map[string]interface{}{
			"status": "deleted",
			"id":     t.ID,
			"title":  t.Title,
		})
	}

	output.Messagef("Deleted task #%d: %s", t.ID, t.Title)
	return nil
}

// deleteSingleCore performs the core delete logic without output (for batch mode).
func deleteSingleCore(cfg *config.Config, id int) error {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting task file: %w", err)
	}

	logActivity(cfg, "delete", t.ID, t.Title)
	return nil
}

func warnDependents(tasksDir string, id int) {
	dependents := board.FindDependents(tasksDir, id)
	for _, msg := range dependents {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
	}
}
