package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
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
	result, err := board.Archive(cfg, id, "", time.Now())
	if err != nil {
		return nil, "", err
	}
	return result.Task, result.OldStatus, nil
}
