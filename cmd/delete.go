package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var deleteCmd = &cobra.Command{
	Use:     "delete ID",
	Aliases: []string{"rm"},
	Short:   "Delete a task",
	Long:    `Deletes a task file. Prompts for confirmation in interactive mode.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

func init() {
	deleteCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	// Warn if other tasks reference this one as a dependency or parent.
	warnDependents(cfg.TasksPath(), t.ID)

	force, _ := cmd.Flags().GetBool("force")

	// Require confirmation in TTY mode unless --force.
	if !force {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return errors.New("cannot prompt for confirmation (not a terminal); use --force")
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

func warnDependents(tasksDir string, id int) {
	dependents := board.FindDependents(tasksDir, id)
	for _, msg := range dependents {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
	}
}
