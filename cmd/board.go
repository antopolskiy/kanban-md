package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var boardCmd = &cobra.Command{
	Use:     "board",
	Aliases: []string{"summary"},
	Short:   "Show board summary",
	Long:    `Displays a summary of the board: task counts per status, WIP utilization, blocked and overdue counts, and priority distribution.`,
	RunE:    runBoard,
}

func init() {
	rootCmd.AddCommand(boardCmd)
}

func runBoard(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	tasks, err := task.ReadAll(cfg.TasksPath())
	if err != nil {
		return err
	}
	if tasks == nil {
		tasks = []*task.Task{}
	}

	summary := board.Summary(cfg, tasks)

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, summary)
	}

	output.OverviewTable(os.Stdout, summary)
	return nil
}
