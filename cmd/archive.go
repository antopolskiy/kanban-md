package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var archiveCmd = &cobra.Command{
	Use:   "archive ID[,ID,...]",
	Short: "Archive a task (soft-delete)",
	Long: `Moves tasks to the archived status. Archived tasks are hidden from
normal commands (list, board, metrics, context, TUI) but remain on disk.
Use 'kanban-md list --archived' to see them.
Multiple IDs can be provided as a comma-separated list.`,
	Args: cobra.ExactArgs(1),
	RunE: runArchive,
}

func init() {
	rootCmd.AddCommand(archiveCmd)
}

func runArchive(_ *cobra.Command, args []string) error {
	ids, err := parseIDs(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(ids) == 1 {
		return archiveSingleTask(cfg, ids[0])
	}

	return runBatch(ids, func(id int) error {
		return executeArchive(cfg, id)
	})
}

func archiveSingleTask(cfg *config.Config, id int) error {
	t, oldStatus, err := executeArchiveCore(cfg, id)
	if err != nil {
		return err
	}

	if oldStatus == "" {
		if outputFormat() == output.FormatJSON {
			return output.JSON(os.Stdout, moveResult{Task: t, Changed: false})
		}
		output.Messagef(os.Stdout, "Task #%d is already archived", t.ID)
		return nil
	}

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, moveResult{Task: t, Changed: true})
	}
	output.Messagef(os.Stdout, "Archived task #%d: %s", id, t.Title)
	return nil
}

func executeArchive(cfg *config.Config, id int) error {
	_, _, err := executeArchiveCore(cfg, id)
	return err
}

func executeArchiveCore(cfg *config.Config, id int) (*task.Task, string, error) {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return nil, "", err
	}

	t, err := task.Read(path)
	if err != nil {
		return nil, "", err
	}

	targetStatus := config.ArchivedStatus

	// Idempotent: if already archived, return unchanged.
	if t.Status == targetStatus {
		return t, "", nil
	}

	oldStatus := t.Status
	t.Status = targetStatus
	task.UpdateTimestamps(t, oldStatus, targetStatus, cfg)
	t.Updated = time.Now()

	if err := task.Write(path, t); err != nil {
		return nil, "", fmt.Errorf("writing task: %w", err)
	}

	logActivity(cfg, "move", id, oldStatus+" -> "+targetStatus)
	return t, oldStatus, nil
}
