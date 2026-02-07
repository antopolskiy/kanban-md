package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var showCmd = &cobra.Command{
	Use:   "show ID",
	Short: "Show task details",
	Long:  `Displays full details of a single task including its markdown body.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(_ *cobra.Command, args []string) error {
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

	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(t)
	}

	output.TaskDetail(t)
	return nil
}
