package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const defaultContextDays = 7

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Generate a board context summary",
	Long: `Generates a markdown summary of the board state, useful for embedding
in CLAUDE.md, AGENTS.md, or other context files.

Use --write-to to write the context to a file. If the file already contains
a kanban-md context block (delimited by HTML comment markers), only that
block is replaced — other content is preserved.`,
	RunE: runContext,
}

func init() {
	contextCmd.Flags().String("write-to", "", "write context to file (create or update in-place)")
	contextCmd.Flags().StringSlice("sections", nil, "comma-separated section filter (in-progress,blocked,ready,overdue,recently-completed)")
	contextCmd.Flags().Int("days", defaultContextDays, "recently completed lookback in days")
	rootCmd.AddCommand(contextCmd)
}

func runContext(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	allTasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return fmt.Errorf("reading tasks: %w", err)
	}
	printWarnings(warnings)

	// Exclude archived tasks from context.
	tasks := make([]*task.Task, 0, len(allTasks))
	for _, t := range allTasks {
		if !cfg.IsArchivedStatus(t.Status) {
			tasks = append(tasks, t)
		}
	}

	sections, _ := cmd.Flags().GetStringSlice("sections")
	days, _ := cmd.Flags().GetInt("days")

	opts := board.ContextOptions{
		Sections: sections,
		Days:     days,
	}

	data := board.GenerateContext(cfg, tasks, opts, time.Now())

	writeTo, _ := cmd.Flags().GetString("write-to")
	if writeTo != "" {
		md := board.RenderContextMarkdown(data)
		if err := board.WriteContextToFile(writeTo, md); err != nil {
			return fmt.Errorf("writing context file: %w", err)
		}
		output.Messagef(os.Stdout, "Context written to %s", writeTo)
		return nil
	}

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, data)
	}

	// Table/auto mode: render markdown to stdout.
	fmt.Print(board.RenderContextMarkdown(data))
	return nil
}
