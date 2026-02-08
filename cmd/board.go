package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
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
}

func runBoard(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Render once.
	if err := renderBoard(cfg); err != nil {
		return err
	}

	if !flagWatch {
		return nil
	}

	return watchBoard(cfg)
}

func renderBoard(cfg *config.Config) error {
	tasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return err
	}
	printWarnings(warnings)
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

func watchBoard(cfg *config.Config) error {
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
		if renderErr := renderBoard(freshCfg); renderErr != nil {
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
