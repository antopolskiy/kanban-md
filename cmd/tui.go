package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/tui"
	"github.com/antopolskiy/kanban-md/internal/watcher"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive board UI",
	Long: `Launches the interactive terminal UI for browsing and managing the
kanban board. The board live-reloads when task files change on disk.

Navigate with arrow keys or vim-style h/j/k/l, press ? for help.`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

// RunTUI launches the interactive TUI.
func RunTUI(dir string) error {
	if dir != "" {
		flagDir = dir
	}
	return runTUI(tuiCmd, nil)
}

func runTUI(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		if isBoardNotFound(err) {
			cfg, err = offerInitTUI()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	model := tui.NewBoard(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startTUIWatcher(ctx, model, p)

	_, err = p.Run()
	return err
}

func isBoardNotFound(err error) bool {
	var cliErr *clierr.Error
	return errors.As(err, &cliErr) && cliErr.Code == clierr.BoardNotFound
}

func offerInitTUI() (*config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	name := filepath.Base(cwd)
	kanbanDir := filepath.Join(cwd, config.DefaultDir)

	fmt.Printf("No kanban board found. Create one in %s? [Y/n] ", kanbanDir)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "" && answer != "y" && answer != "yes" {
		return nil, errors.New("no board found — run 'kanban-md init' to create one")
	}

	cfg, err := config.Init(kanbanDir, name)
	if err != nil {
		return nil, fmt.Errorf("initializing board: %w", err)
	}

	fmt.Printf("Board %q created in %s\n", name, kanbanDir)
	return cfg, nil
}

func startTUIWatcher(ctx context.Context, model *tui.Board, p *tea.Program) {
	paths := model.WatchPaths()
	w, err := watcher.New(paths, func() {
		p.Send(tui.ReloadMsg{})
	})
	if err != nil {
		return // non-fatal: TUI works without live refresh
	}
	defer w.Close()
	w.Run(ctx, nil)
}
