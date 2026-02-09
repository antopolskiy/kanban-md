// tui-showcase renders the TUI board with full ANSI256 colors for screenshot generation.
//
// Usage:
//
//	go run ./cmd/tui-showcase | freeze -o assets/tui-screenshot.png \
//	  --language ansi --font.size 14 --theme "Catppuccin Mocha" --padding 20 --window
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/antopolskiy/kanban-md/internal/tui"
)

func main() {
	// Force 256-color output regardless of terminal detection.
	lipgloss.SetColorProfile(termenv.ANSI256)

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dir, err := os.MkdirTemp("", "tui-showcase-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		return err
	}

	cfg := config.NewDefault("My Project")
	cfg.TUI.TitleLines = 2
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		return err
	}

	refNow := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	claimedAt := &refNow

	const (
		twoWeeks  = 14 * 24 * time.Hour
		oneWeek   = 7 * 24 * time.Hour
		threeDays = 3 * 24 * time.Hour
		oneDay    = 24 * time.Hour
		recentAge = 5 * time.Hour
	)

	tasks := []task.Task{
		// Backlog (8) — 2 weeks old
		{ID: 1, Title: "Performance testing", Status: "backlog", Priority: "low", Tags: []string{"testing"}, Updated: refNow.Add(-twoWeeks)},
		{ID: 2, Title: "Mobile responsive layout", Status: "backlog", Priority: "medium", Tags: []string{"frontend"}, Updated: refNow.Add(-twoWeeks)},
		{ID: 3, Title: "Database migration tool", Status: "backlog", Priority: "low", Tags: []string{"backend"}, Updated: refNow.Add(-twoWeeks)},
		{ID: 15, Title: "Add search autocomplete", Status: "backlog", Priority: "medium", Tags: []string{"frontend"}, Updated: refNow.Add(-twoWeeks)},
		{ID: 16, Title: "Email notification service", Status: "backlog", Priority: "high", Tags: []string{"backend"}, Updated: refNow.Add(-twoWeeks)},
		{ID: 17, Title: "Localization support", Status: "backlog", Priority: "low", Tags: []string{"i18n"}, Updated: refNow.Add(-twoWeeks)},
		{ID: 18, Title: "Audit logging", Status: "backlog", Priority: "medium", Tags: []string{"security"}, Updated: refNow.Add(-twoWeeks)},
		{ID: 19, Title: "Export to CSV", Status: "backlog", Priority: "low", Tags: []string{"feature"}, Updated: refNow.Add(-twoWeeks)},
		// Todo (3) — 3 days old
		{ID: 4, Title: "Add rate limiting", Status: "todo", Priority: "medium", Tags: []string{"backend"}, Updated: refNow.Add(-threeDays)},
		{ID: 5, Title: "Set up monitoring", Status: "todo", Priority: "medium", Tags: []string{"devops"}, Updated: refNow.Add(-threeDays)},
		{ID: 6, Title: "Write integration tests", Status: "todo", Priority: "high", Tags: []string{"testing"}, Updated: refNow.Add(-threeDays)},
		// In-progress (3) — 5 hours old, each claimed by a unique agent
		{ID: 7, Title: "Build dashboard UI", Status: "in-progress", Priority: "high", Tags: []string{"frontend"}, ClaimedBy: "frost-maple", ClaimedAt: claimedAt, Updated: refNow.Add(-recentAge)},
		{ID: 8, Title: "Write API docs", Status: "in-progress", Priority: "medium", Tags: []string{"docs"}, ClaimedBy: "amber-swift", ClaimedAt: claimedAt, Updated: refNow.Add(-recentAge)},
		{ID: 9, Title: "Fix auth token refresh", Status: "in-progress", Priority: "critical", Tags: []string{"security"}, ClaimedBy: "coral-dusk", ClaimedAt: claimedAt, Updated: refNow.Add(-recentAge)},
		// Review (2) — 1 day old, each claimed by a unique agent
		{ID: 10, Title: "Implement user auth", Status: "review", Priority: "critical", Tags: []string{"backend"}, ClaimedBy: "sage-river", ClaimedAt: claimedAt, Updated: refNow.Add(-oneDay)},
		{ID: 11, Title: "Design REST API schema", Status: "review", Priority: "high", Tags: []string{"api"}, ClaimedBy: "neon-drift", ClaimedAt: claimedAt, Updated: refNow.Add(-oneDay)},
		// Done (3) — 1 week old
		{ID: 12, Title: "Set up CI pipeline", Status: "done", Priority: "high", Tags: []string{"devops"}, Updated: refNow.Add(-oneWeek)},
		{ID: 13, Title: "Create project scaffold", Status: "done", Priority: "high", Tags: []string{"setup"}, Updated: refNow.Add(-oneWeek)},
		{ID: 14, Title: "Define database schema", Status: "done", Priority: "medium", Tags: []string{"backend"}, Updated: refNow.Add(-oneWeek)},
	}

	for i := range tasks {
		tk := &tasks[i]
		path := filepath.Join(tasksDir, task.GenerateFilename(tk.ID, tk.Title))
		if err := task.Write(path, tk); err != nil {
			return err
		}
	}

	b := tui.NewBoard(cfg)
	b.SetNow(func() time.Time { return refNow })
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 36})

	fmt.Print(b.View())
	return nil
}
