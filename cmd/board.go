package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/antopolskiy/kanban-md/internal/watcher"
)

var flagWatch bool

var boardCmd = &cobra.Command{
	Use:     "board",
	Aliases: []string{"summary"},
	Short:   "Show board summary",
	Long: `Displays a summary of the board: task counts per status, WIP utilization,
blocked and overdue counts, and priority distribution.

Use --watch to keep the display live-updating. The board re-renders automatically
whenever task files change on disk (e.g., from another terminal or an AI agent).
Press Ctrl+C to stop.`,
	RunE: runBoard,
}

func init() {
	rootCmd.AddCommand(boardCmd)
	boardCmd.Flags().BoolVarP(&flagWatch, "watch", "w", false, "live-update the board on file changes")
	boardCmd.Flags().String("group-by", "", "group board by field ("+strings.Join(board.ValidGroupByFields(), ", ")+")")
}

func runBoard(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	groupBy, _ := cmd.Flags().GetString("group-by")
	if groupBy != "" && !slices.Contains(board.ValidGroupByFields(), groupBy) {
		return clierr.Newf(clierr.InvalidGroupBy, "invalid --group-by field %q; valid: %s",
			groupBy, strings.Join(board.ValidGroupByFields(), ", "))
	}

	// Render once.
	if err := renderBoard(cfg, groupBy); err != nil {
		return err
	}

	if !flagWatch {
		return nil
	}

	return watchBoard(cfg, groupBy)
}

func renderBoard(cfg *config.Config, groupBy string) error {
	tasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return err
	}
	printWarnings(warnings)
	if tasks == nil {
		tasks = []*task.Task{}
	}

	if groupBy != "" {
		return renderGroupedBoard(cfg, tasks, groupBy)
	}

	summary := board.Summary(cfg, tasks)

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, summary)
	}

	output.OverviewTable(os.Stdout, summary)
	return nil
}

func renderGroupedBoard(cfg *config.Config, tasks []*task.Task, groupBy string) error {
	grouped := board.GroupBy(tasks, groupBy, cfg)

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, grouped)
	}

	output.GroupedTable(os.Stdout, grouped)
	return nil
}

func watchBoard(cfg *config.Config, groupBy string) error {
	// Watch both the tasks directory and the config file's directory.
	watchPaths := []string{cfg.TasksPath(), cfg.Dir()}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w, err := watcher.New(watchPaths, func() {
		clearScreen()
		// Re-load config in case statuses/WIP limits changed.
		freshCfg, loadErr := config.Load(cfg.Dir())
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: reloading config: %v\n", loadErr)
			freshCfg = cfg
		}
		if renderErr := renderBoard(freshCfg, groupBy); renderErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: rendering board: %v\n", renderErr)
		}
	})
	if err != nil {
		return fmt.Errorf("starting file watcher: %w", err)
	}
	defer w.Close()

	fmt.Fprintln(os.Stderr, "Watching for changes... (Ctrl+C to stop)")

	w.Run(ctx, func(watchErr error) {
		fmt.Fprintf(os.Stderr, "Warning: file watcher: %v\n", watchErr)
	})

	return nil
}

// clearScreen sends ANSI escape codes to clear the terminal and move the
// cursor to the top-left corner.
func clearScreen() {
	fmt.Fprint(os.Stdout, "\033[2J\033[H")
}
