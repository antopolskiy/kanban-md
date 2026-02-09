package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/antopolskiy/kanban-md/internal/tui"
)

// setupTestBoard creates a temp kanban directory with a config and tasks,
// then returns a Board model ready for testing.
func setupTestBoard(t *testing.T) (*tui.Board, *config.Config) {
	t.Helper()

	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Test Board")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Create test tasks.
	tasks := []struct {
		id       int
		title    string
		status   string
		priority string
	}{
		{1, "Task A", "backlog", "high"},
		{2, "Task B", "backlog", "medium"},
		{3, "Task C", "in-progress", "high"},
		{4, "Task D", "done", "low"},
	}

	for _, tt := range tasks {
		tk := &task.Task{
			ID:       tt.id,
			Title:    tt.title,
			Status:   tt.status,
			Priority: tt.priority,
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(tt.id, tt.title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	b := tui.NewBoard(cfg)
	// Simulate window size.
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	return b, cfg
}

func sendKey(b *tui.Board, k string) *tui.Board {
	m, _ := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
	return m.(*tui.Board)
}

func sendSpecialKey(b *tui.Board, k tea.KeyType) *tui.Board {
	m, _ := b.Update(tea.KeyMsg{Type: k})
	return m.(*tui.Board)
}

func TestBoard_InitialState(t *testing.T) {
	b, _ := setupTestBoard(t)
	v := b.View()

	// Should show all status columns.
	if v == "" || v == "Loading..." {
		t.Error("expected board view, got empty or loading")
	}

	// Board should contain task titles.
	if !containsStr(v, "Task A") {
		t.Error("expected Task A in view")
	}
	if !containsStr(v, "Task C") {
		t.Error("expected Task C in view")
	}
}

func TestBoard_NavigateColumns(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Move right twice, then left twice — should not panic.
	b = sendKey(b, "l")
	b = sendKey(b, "l")
	b = sendKey(b, "h")
	b = sendKey(b, "h")

	// Moving left past column 0 should not panic.
	b = sendKey(b, "h")

	// View should render without issues.
	v := b.View()
	if v == "" || v == "Loading..." {
		t.Error("expected valid board view after navigation")
	}
}

func TestBoard_NavigateRows(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Column 0 (backlog) has 2 tasks. Move down.
	b = sendKey(b, "j")

	// Move down again should not crash (already at last).
	b = sendKey(b, "j")

	// Move up back.
	b = sendKey(b, "k")

	// Should not panic.
	_ = b.View()
}

func TestBoard_EnterDetail(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Press enter to see detail.
	b = sendKey(b, "enter")
	v := b.View()

	// Detail view should show task fields.
	if !containsStr(v, "Status:") {
		t.Error("expected detail view with Status field")
	}

	// Press esc to go back.
	b = sendSpecialKey(b, tea.KeyEsc)
	v = b.View()

	// Should be back to board.
	if containsStr(v, "Press q/esc to go back") {
		t.Error("expected to return to board view")
	}
}

func TestBoard_MoveDialog(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Press m to open move dialog.
	b = sendKey(b, "m")
	v := b.View()

	if !containsStr(v, "Move #") {
		t.Error("expected move dialog")
	}
	if !containsStr(v, "(current)") {
		t.Error("expected current status marker in move dialog")
	}

	// Press esc to cancel.
	b = sendSpecialKey(b, tea.KeyEsc)
	_ = b.View()
}

func TestBoard_MoveTask(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Select task in backlog, move to todo.
	b = sendKey(b, "m")

	// Move cursor down to "todo".
	b = sendKey(b, "j")

	// Press enter to confirm.
	_ = sendKey(b, "enter")

	// Verify the task was actually moved.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Status != "todo" {
		t.Errorf("expected status 'todo', got %q", tk.Status)
	}
}

func TestBoard_MoveNext(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Press M (shift-m) to move to next status.
	b = sendKey(b, "M")

	// Task 1 was in backlog, should now be in todo.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Status != "todo" {
		t.Errorf("expected status 'todo', got %q", tk.Status)
	}

	_ = b.View()
}

func TestBoard_DeleteTask(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Press d to start delete.
	b = sendKey(b, "d")
	v := b.View()

	if !containsStr(v, "Delete task?") {
		t.Error("expected delete confirmation dialog")
	}

	// Press y to confirm.
	b = sendKey(b, "y")

	// Task 1 should be gone.
	_, err := task.FindByID(cfg.TasksPath(), 1)
	if err == nil {
		t.Error("expected task 1 to be deleted")
	}

	_ = b.View()
}

func TestBoard_DeleteCancel(t *testing.T) {
	b, cfg := setupTestBoard(t)

	b = sendKey(b, "d")
	b = sendKey(b, "n")

	// Task should still exist.
	_, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Error("expected task 1 to still exist after cancel")
	}

	_ = b.View()
}

func TestBoard_HelpView(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "?")
	v := b.View()

	if !containsStr(v, "Keyboard Shortcuts") {
		t.Error("expected help view")
	}

	// Any key should close help.
	b = sendKey(b, "q")
	v = b.View()

	if containsStr(v, "Keyboard Shortcuts") {
		t.Error("expected help view to close")
	}
}

func TestBoard_Refresh(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Create a new task externally.
	tk := &task.Task{
		ID:       5,
		Title:    "External Task",
		Status:   "todo",
		Priority: "medium",
	}
	path := filepath.Join(cfg.TasksPath(), task.GenerateFilename(5, "External Task"))
	if err := task.Write(path, tk); err != nil {
		t.Fatalf("writing task: %v", err)
	}

	// Press r to refresh.
	b = sendKey(b, "r")
	v := b.View()

	if !containsStr(v, "External Task") {
		t.Error("expected External Task in view after refresh")
	}
}

func TestBoard_EmptyBoard(t *testing.T) {
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

	v := b.View()
	if !containsStr(v, "(empty)") {
		t.Error("expected empty column indicator")
	}
}

func TestBoard_ClaimedByDisplayed(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Test Board")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Create a task with ClaimedBy set.
	tk := &task.Task{
		ID:        1,
		Title:     "Claimed task",
		Status:    "in-progress",
		Priority:  "high",
		ClaimedBy: "agent-1",
	}
	path := filepath.Join(tasksDir, task.GenerateFilename(1, "Claimed task"))
	if err := task.Write(path, tk); err != nil {
		t.Fatalf("writing task: %v", err)
	}

	b := tui.NewBoard(cfg)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	v := b.View()
	if !containsStr(v, "@agent-1") {
		t.Error("expected @agent-1 in board view for claimed task")
	}

	// Also check detail view.
	b = sendSpecialKey(b, tea.KeyEnter)
	v = b.View()
	if !containsStr(v, "agent-1") {
		t.Error("expected agent-1 in detail view for claimed task")
	}
}

func containsStr(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 &&
		haystack != needle && // avoid trivial match
		findSubstring(haystack, needle)
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// addLongBodyToTask modifies a task file to have a multi-line body.
func addLongBodyToTask(t *testing.T, cfg *config.Config, taskID, lineCount int) { //nolint:unparam // helper accepts any task ID
	t.Helper()
	path, err := task.FindByID(cfg.TasksPath(), taskID)
	if err != nil {
		t.Fatalf("finding task %d: %v", taskID, err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task %d: %v", taskID, err)
	}
	var bodyLines []string
	for i := 1; i <= lineCount; i++ {
		bodyLines = append(bodyLines, fmt.Sprintf("Body line %d content here", i))
	}
	tk.Body = strings.Join(bodyLines, "\n")
	if err := task.Write(path, tk); err != nil {
		t.Fatalf("writing task %d: %v", taskID, err)
	}
}

// --- Bug #55: Detail view starts at bottom, scrolling doesn't work ---

func TestBoard_DetailStartsAtTop(t *testing.T) {
	b, cfg := setupTestBoard(t)
	addLongBodyToTask(t, cfg, 1, 50)

	b = sendKey(b, "r")     // refresh to pick up body change
	b = sendKey(b, "enter") // enter detail view
	v := b.View()

	// Should show title at the top.
	if !containsStr(v, "Task #1") {
		t.Error("expected Task #1 in detail view")
	}
	// First body line should be visible.
	if !containsStr(v, "Body line 1") {
		t.Error("expected first body line visible")
	}
}

func TestBoard_DetailFitsTerminal(t *testing.T) {
	b, cfg := setupTestBoard(t)
	addLongBodyToTask(t, cfg, 1, 50) // 50 body lines + metadata > 40 lines

	b = sendKey(b, "r")
	b = sendKey(b, "enter")
	v := b.View()

	lines := strings.Split(v, "\n")
	if len(lines) > 40 {
		t.Errorf("detail view has %d lines, exceeds terminal height 40", len(lines))
	}
}

func TestBoard_DetailScrollDown(t *testing.T) {
	b, cfg := setupTestBoard(t)
	addLongBodyToTask(t, cfg, 1, 50)

	b = sendKey(b, "r")
	b = sendKey(b, "enter")

	// Scroll down 20 lines.
	for range 20 {
		b = sendKey(b, "j")
	}
	v := b.View()

	// After scrolling, the first body line should be gone.
	if containsStr(v, "Body line 1 content") {
		t.Error("expected first body line to be scrolled out of view")
	}
}

// --- Bug #56: Detail view doesn't wrap long lines ---

func TestBoard_DetailWrapsLongLines(t *testing.T) {
	b, cfg := setupTestBoard(t)

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	tk.Body = strings.Repeat("word ", 50) // 250 chars, exceeds width of 120
	if err := task.Write(path, tk); err != nil {
		t.Fatalf("writing task: %v", err)
	}

	b = sendKey(b, "r")
	b = sendKey(b, "enter")
	v := b.View()

	// No line should exceed terminal width.
	for i, line := range strings.Split(v, "\n") {
		if len(line) > 120 {
			t.Errorf("line %d exceeds width 120: len=%d", i, len(line))
		}
	}
}

// --- Bug #58: Column headers disappear when scrolling ---

func TestBoard_ScrollHeaderVisible(t *testing.T) {
	b, _ := setupManyTasksBoard(t)
	// Use height 24 where indicators cause overflow.
	b.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	// Navigate to done column (index 4).
	for range 4 {
		b = sendKey(b, "l")
	}
	// Scroll down to trigger both up and down indicators.
	for range 5 {
		b = sendKey(b, "j")
	}
	v := b.View()

	// Total output lines must not exceed terminal height.
	lines := strings.Split(v, "\n")
	if len(lines) > 24 {
		t.Errorf("output has %d lines, exceeds terminal height 24", len(lines))
	}

	// Header row should be the first line and contain all column names.
	if len(lines) > 0 && !containsStr(lines[0], "done") {
		t.Errorf("expected 'done' header on first line, got %q", lines[0])
	}
}
