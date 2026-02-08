package tui_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/antopolskiy/kanban-md/internal/tui"
)

var update = flag.Bool("update", false, "update golden files")

func init() {
	// Strip all ANSI codes so golden files are plain text.
	lipgloss.SetColorProfile(termenv.Ascii)
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")

	if *update {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatalf("creating testdata dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("writing golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path) //nolint:gosec // test golden file path is safe
	if err != nil {
		t.Fatalf("reading golden file %s (run with -update to create): %v", path, err)
	}

	if got != string(want) {
		// Write "got" file for easy comparison.
		gotPath := path + ".got"
		_ = os.WriteFile(gotPath, []byte(got), 0o600)
		t.Errorf("output does not match golden file %s\n  got file: %s\n  run with -update to accept changes", path, gotPath)
	}
}

func TestSnapshot_BoardView(t *testing.T) {
	b, _ := setupTestBoard(t)
	assertGolden(t, "board_view", b.View())
}

func TestSnapshot_DetailView(t *testing.T) {
	b, _ := setupTestBoard(t)
	b = sendKey(b, "enter") // open detail
	assertGolden(t, "detail_view", b.View())
}

func TestSnapshot_MoveDialog(t *testing.T) {
	b, _ := setupTestBoard(t)
	b = sendKey(b, "m") // open move dialog
	assertGolden(t, "move_dialog", b.View())
}

func TestSnapshot_DeleteConfirm(t *testing.T) {
	b, _ := setupTestBoard(t)
	b = sendKey(b, "d") // open delete confirm
	assertGolden(t, "delete_confirm", b.View())
}

func TestSnapshot_HelpView(t *testing.T) {
	b, _ := setupTestBoard(t)
	b = sendKey(b, "?") // open help
	assertGolden(t, "help_view", b.View())
}

func TestSnapshot_BoardView80(t *testing.T) {
	b, _ := setupTestBoard80(t)
	assertGolden(t, "board_view_80", b.View())
}

func setupTestBoard80(t *testing.T) (*tui.Board, *config.Config) {
	t.Helper()
	b, cfg := setupTestBoard(t)
	b.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return b, cfg
}

func TestSnapshot_BoardView60(t *testing.T) {
	b, _ := setupTestBoard(t)
	b.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	assertGolden(t, "board_view_60", b.View())
}

func TestSnapshot_ScrollDown(t *testing.T) {
	b, _ := setupManyTasksBoard(t)
	// Navigate to the "done" column (index 4) which has 15 tasks.
	for range 4 {
		b = sendKey(b, "l")
	}
	// Scroll down past the visible window.
	for range 10 {
		b = sendKey(b, "j")
	}
	assertGolden(t, "scroll_down", b.View())
}

func TestSnapshot_ManyTasks(t *testing.T) {
	b, _ := setupManyTasksBoard(t)
	assertGolden(t, "many_tasks", b.View())
}

func setupManyTasksBoard(t *testing.T) (*tui.Board, *config.Config) { //nolint:unparam // matches setupTestBoard signature
	t.Helper()

	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Scroll Test")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Create 15 tasks in "done" to force scrolling at height 30.
	priorities := [4]string{"critical", "high", "medium", "low"}
	for i := 1; i <= 15; i++ {
		tk := &task.Task{
			ID:       i,
			Title:    fmt.Sprintf("Done task %d", i),
			Status:   "done",
			Priority: priorities[i%len(priorities)], //nolint:gosec // test data, no overflow
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(i, tk.Title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	// Create 3 tasks in "backlog".
	for i := 16; i <= 18; i++ {
		tk := &task.Task{
			ID:       i,
			Title:    fmt.Sprintf("Backlog task %d", i),
			Status:   "backlog",
			Priority: "medium",
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(i, tk.Title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	b := tui.NewBoard(cfg)
	b.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return b, cfg
}

func TestSnapshot_DetailLongBody(t *testing.T) {
	b, cfg := setupTestBoard(t)
	addLongBodyToTask(t, cfg, 1, 30)
	b = sendKey(b, "r")
	b = sendKey(b, "enter")
	assertGolden(t, "detail_long_body", b.View())
}

func TestSnapshot_ScrollBothIndicators(t *testing.T) {
	b, _ := setupManyTasksBoard(t)
	// Use height 24 to test tight layout with both indicators.
	b.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	// Navigate to done column and scroll to middle.
	for range 4 {
		b = sendKey(b, "l")
	}
	for range 5 {
		b = sendKey(b, "j")
	}
	assertGolden(t, "scroll_both_indicators", b.View())
}

func TestSnapshot_EmptyBoard(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Empty Board")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	b := tui.NewBoard(cfg)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assertGolden(t, "empty_board", b.View())
}
