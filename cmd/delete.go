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
	Short:   "Delete a task",
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
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	// Check claim before allowing delete.
	if err = checkClaim(t, "", cfg.ClaimTimeoutDuration()); err != nil {
		return err
	}

	// Warn if other tasks reference this one as a dependency or parent.
	warnDependents(cfg.TasksPath(), t.ID)

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

	if err := softDeleteAndLog(cfg, path, t); err != nil {
		return err
	}

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, map[string]interface{}{
			"status": "deleted",
			"id":     t.ID,
			"title":  t.Title,
		})
	}

	output.Messagef(os.Stdout, "Deleted task #%d: %s", t.ID, t.Title)
	return nil
}

// executeDelete performs the core delete: find, read, claim check, warn dependents, remove, log.
func executeDelete(cfg *config.Config, id int) error {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	if err = checkClaim(t, "", cfg.ClaimTimeoutDuration()); err != nil {
		return err
	}

	warnDependents(cfg.TasksPath(), t.ID)
	return softDeleteAndLog(cfg, path, t)
}

// softDeleteAndLog archives the task and logs the delete action.
func softDeleteAndLog(cfg *config.Config, path string, t *task.Task) error {
	if t.Status == config.ArchivedStatus {
		return nil
	}

	oldStatus := t.Status
	t.Status = config.ArchivedStatus
	task.UpdateTimestamps(t, oldStatus, t.Status, cfg)
	t.Updated = time.Now()

	if err := task.Write(path, t); err != nil {
		return fmt.Errorf("writing task: %w", err)
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
