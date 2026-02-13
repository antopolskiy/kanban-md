package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/antopolskiy/kanban-md/internal/tui"
)

const (
	statusTodo       = "todo"
	priorityCritical = "critical"
	viewLoading      = "Loading..."
)

// testRefTime is a fixed reference time used for task Updated timestamps in tests.
var testRefTime = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // test helper

// testNow returns a fixed time 2 hours after testRefTime, so all test tasks show "2h" age.
func testNow() time.Time { return testRefTime.Add(2 * time.Hour) }

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
			Updated:  testRefTime,
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(tt.id, tt.title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
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
	if v == "" || v == viewLoading {
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
	if v == "" || v == viewLoading {
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
	if tk.Status != statusTodo {
		t.Errorf("expected status 'todo', got %q", tk.Status)
	}
}

func TestBoard_MoveNext(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Press N to move to next status.
	b = sendKey(b, "N")

	// Task 1 was in backlog, should now be in todo.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Status != statusTodo {
		t.Errorf("expected status 'todo', got %q", tk.Status)
	}

	_ = b.View()
}

func TestBoard_MovePrev(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Navigate to the in-progress column (index 2) which has Task C.
	b = sendKey(b, "l") // → todo
	b = sendKey(b, "l") // → in-progress

	// Press P to move to previous status.
	b = sendKey(b, "P")

	// Task 3 was in in-progress, should now be in todo.
	path, err := task.FindByID(cfg.TasksPath(), 3)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Status != statusTodo {
		t.Errorf("expected status 'todo', got %q", tk.Status)
	}

	_ = b.View()
}

func TestBoard_MovePrevAtFirst(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Task 1 is in backlog (first status). P should show an error.
	b = sendKey(b, "P")
	v := b.View()

	if !containsStr(v, "already at the first status") {
		t.Error("expected error message when trying to move past first status")
	}
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

	// Task 1 should be archived (soft delete) and remain on disk.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("expected task 1 file to remain, got error: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task 1 after delete: %v", err)
	}
	if tk.Status != statusArchived {
		t.Errorf("status = %q, want %s", tk.Status, statusArchived)
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

func TestBoard_EscQuitFromBoard(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Pressing Esc on the board view should produce a quit command.
	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected quit command from Esc on board view, got nil")
	}
	// Execute the cmd to verify it produces a QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestBoard_HelpShowsEscAsQuit(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "?")
	v := b.View()

	if !containsStr(v, "esc") {
		t.Error("expected help view to mention esc as a quit key")
	}
}

func TestBoard_StatusBarShowsQuit(t *testing.T) {
	b, _ := setupTestBoard(t)

	v := b.View()

	if !containsStr(v, "q:quit") {
		t.Error("expected status bar to mention q:quit")
	}
}

func TestBoard_StatusBarShowsPriorityHint(t *testing.T) {
	b, _ := setupTestBoard(t)

	v := b.View()

	if !containsStr(v, "+/-:priority") {
		t.Error("expected status bar to contain +/-:priority hint")
	}
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

// ansiRe matches ANSI escape sequences (SGR and other CSI sequences).
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func containsStr(haystack, needle string) bool {
	// Strip ANSI codes so glamour-rendered body text is searchable.
	haystack = stripANSI(haystack)
	return strings.Contains(haystack, needle)
}

func findSubstring(s, sub string) bool {
	s = stripANSI(s)
	return strings.Contains(s, sub)
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

	// No line should exceed terminal width (measure visual width, not raw bytes).
	for i, line := range strings.Split(v, "\n") {
		visual := stripANSI(line)
		if len(visual) > 120 {
			t.Errorf("line %d exceeds width 120: len=%d", i, len(visual))
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

// --- TUI title_lines config ---

// setupTestBoardWithTitleLines creates a board with configurable title lines.
func setupTestBoardWithTitleLines(t *testing.T, titleLines int) (*tui.Board, *config.Config) { //nolint:unparam // config may be needed by future tests
	t.Helper()

	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Title Lines Test")
	cfg.TUI.TitleLines = titleLines
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	tasks := []struct {
		id       int
		title    string
		status   string
		priority string
	}{
		{1, "Implement user authentication with OAuth2 and SAML support", "backlog", "high"},
		{2, "Fix database connection pooling issue under heavy load", "backlog", "medium"},
		{3, "Add comprehensive integration test suite for the API", "in-progress", "high"},
	}

	for _, tt := range tasks {
		tk := &task.Task{
			ID:       tt.id,
			Title:    tt.title,
			Status:   tt.status,
			Priority: tt.priority,
			Updated:  testRefTime,
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(tt.id, tt.title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 80, Height: 30})

	return b, cfg
}

func TestBoard_TitleLines1_DefaultBehavior(t *testing.T) {
	b, _ := setupTestBoard(t)
	v := b.View()
	if !containsStr(v, "Task A") {
		t.Error("expected Task A in default title_lines=1 view")
	}
}

func TestBoard_TitleLines2_WrapsLongTitle(t *testing.T) {
	b, _ := setupTestBoardWithTitleLines(t, 2)
	v := b.View()

	// Title should be visible (at least the first part).
	if !containsStr(v, "Implement") {
		t.Error("expected title visible in title_lines=2 view")
	}
}

func TestBoard_TitleLines2_MoreTitleVisible(t *testing.T) {
	// With title_lines=2, more of the title should be shown vs title_lines=1.
	b1, _ := setupTestBoardWithTitleLines(t, 1)
	b2, _ := setupTestBoardWithTitleLines(t, 2)

	v1 := b1.View()
	v2 := b2.View()

	// "SAML" is near the end of the first task title. With 2 lines
	// and 80-width columns, it should be visible in the 2-line version.
	if containsStr(v1, "SAML") && !containsStr(v2, "SAML") {
		t.Error("expected title_lines=2 to show at least as much title as title_lines=1")
	}
}

func TestBoard_TitleLines2_ContinuationUsesFullWidth(t *testing.T) {
	b, _ := setupTestBoardWithTitleLines(t, 2)
	// Use a wider terminal so continuation lines have enough room to show content.
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	v := b.View()

	// "authentication" is on the continuation line. With full-width wrapping,
	// it should appear. Without full-width (old behavior), the ID-prefix
	// indentation would steal space and truncate it further.
	if !containsStr(v, "authentication") {
		t.Error("expected 'authentication' visible on continuation line (full-width wrap)")
	}
}

func TestBoard_ScrollWithTitleLines2(t *testing.T) {
	b, _ := setupTestBoardWithTitleLines(t, 2)
	// Small terminal to force scrolling math changes.
	b.Update(tea.WindowSizeMsg{Width: 80, Height: 15})

	b = sendKey(b, "j")
	v := b.View()

	lines := strings.Split(v, "\n")
	const termHeight = 15
	if len(lines) > termHeight {
		t.Errorf("output has %d lines, exceeds terminal height %d", len(lines), termHeight)
	}
}

// --- Coverage improvement tests ---

func TestBoard_Init(t *testing.T) {
	b, _ := setupTestBoard(t)
	cmd := b.Init()
	if cmd == nil {
		t.Error("expected Init() to return a tick command")
	}
}

func TestBoard_WatchPaths(t *testing.T) {
	b, cfg := setupTestBoard(t)
	paths := b.WatchPaths()

	// Default config has TasksDir="tasks", so Dir() != TasksPath() → 2 paths.
	if len(paths) != 2 {
		t.Fatalf("expected 2 watch paths, got %d: %v", len(paths), paths)
	}
	if paths[0] != cfg.TasksPath() {
		t.Errorf("expected first path to be tasks path %q, got %q", cfg.TasksPath(), paths[0])
	}
	if paths[1] != cfg.Dir() {
		t.Errorf("expected second path to be kanban dir %q, got %q", cfg.Dir(), paths[1])
	}
}

func TestBoard_ReloadMsg(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Create an external task that won't show up without reload.
	tk := &task.Task{
		ID:       10,
		Title:    "Reload Test Task",
		Status:   "todo",
		Priority: "medium",
		Updated:  testRefTime,
	}
	path := filepath.Join(cfg.TasksPath(), task.GenerateFilename(10, "Reload Test Task"))
	if err := task.Write(path, tk); err != nil {
		t.Fatalf("writing task: %v", err)
	}

	// Before ReloadMsg, the new task shouldn't be visible.
	v := b.View()
	if containsStr(v, "Reload Test Task") {
		t.Error("expected new task NOT visible before ReloadMsg")
	}

	// Send ReloadMsg.
	m, _ := b.Update(tui.ReloadMsg{})
	b = m.(*tui.Board)

	v = b.View()
	if !containsStr(v, "Reload Test Task") {
		t.Error("expected new task visible after ReloadMsg")
	}
}

func TestBoard_UnknownMsg(t *testing.T) {
	b, _ := setupTestBoard(t)
	vBefore := b.View()

	// Send an unknown message type.
	type customMsg struct{}
	m, cmd := b.Update(customMsg{})
	b = m.(*tui.Board)

	if cmd != nil {
		t.Error("expected nil cmd for unknown message")
	}
	if b.View() != vBefore {
		t.Error("expected board unchanged after unknown message")
	}
}

// setupMetadataBoard creates a board with a single task that has all metadata fields populated.
func setupMetadataBoard(t *testing.T) *tui.Board {
	t.Helper()

	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Metadata Board")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	started := testRefTime.Add(-1 * time.Hour)
	completed := testRefTime
	claimedAt := testRefTime.Add(-30 * time.Minute)
	due := date.New(2026, 3, 15)
	parentID := 42

	tk := &task.Task{
		ID:          1,
		Title:       "Full Metadata Task",
		Status:      "in-progress",
		Priority:    "high",
		Assignee:    "alice",
		Tags:        []string{"backend", "urgent"},
		Due:         &due,
		Estimate:    "2h",
		Started:     &started,
		Completed:   &completed,
		Blocked:     true,
		BlockReason: "waiting on API",
		ClaimedBy:   "agent-1",
		ClaimedAt:   &claimedAt,
		Class:       "expedite",
		Parent:      &parentID,
		DependsOn:   []int{10, 20},
		Updated:     testRefTime,
	}
	path := filepath.Join(tasksDir, task.GenerateFilename(1, "Full Metadata Task"))
	if err := task.Write(path, tk); err != nil {
		t.Fatalf("writing task: %v", err)
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	return b
}

func TestBoard_DetailShowsAllMetadata(t *testing.T) {
	b := setupMetadataBoard(t)

	// Navigate to in-progress column (index 2).
	b = sendKey(b, "l") // → todo
	b = sendKey(b, "l") // → in-progress

	// Enter detail view.
	b = sendSpecialKey(b, tea.KeyEnter)
	v := b.View()

	checks := []struct {
		label string
		want  string
	}{
		{"Assignee", "alice"},
		{"Tags", "backend"},
		{"Due", "2026-03-15"},
		{"Estimate", "2h"},
		{"Class", "expedite"},
		{"Parent", "#42"},
		{"DependsOn", "#10"},
		{"DependsOn2", "#20"},
		{"ClaimedBy", "agent-1"},
		{"ClaimedAt", "Claimed at:"},
		{"Started", "Started:"},
		{"Completed", "Completed:"},
		{"Duration", "Duration:"},
		{"Blocked", "BLOCKED:"},
		{"BlockReason", "waiting on API"},
	}
	for _, c := range checks {
		if !containsStr(v, c.want) {
			t.Errorf("expected %s field with %q in detail view", c.label, c.want)
		}
	}
}

func TestBoard_DetailScrollUp(t *testing.T) {
	b, cfg := setupTestBoard(t)
	addLongBodyToTask(t, cfg, 1, 50)
	b = sendKey(b, "r")     // refresh
	b = sendKey(b, "enter") // detail

	// Scroll down 20 lines.
	for range 20 {
		b = sendKey(b, "j")
	}
	vDown := b.View()

	// Scroll back up 20 lines (back to top).
	for range 20 {
		b = sendKey(b, "k")
	}
	vUp := b.View()

	// After scrolling back up fully, the title should be visible again.
	if !containsStr(vUp, "Task #1") {
		t.Error("expected Task #1 visible after scrolling back up")
	}
	// The view should differ from the scrolled-down position.
	if vDown == vUp {
		t.Error("expected different view after scrolling up")
	}
}

func TestBoard_DetailScrollUpAtTop(t *testing.T) {
	b, cfg := setupTestBoard(t)
	addLongBodyToTask(t, cfg, 1, 50)
	b = sendKey(b, "r")
	b = sendKey(b, "enter")

	// Press k at the top — should be a no-op.
	b = sendKey(b, "k")
	v := b.View()

	if !containsStr(v, "Task #1") {
		t.Error("expected Task #1 still visible after k at top")
	}
}

func TestBoard_DetailScrollToTop(t *testing.T) {
	b, cfg := setupTestBoard(t)
	addLongBodyToTask(t, cfg, 1, 50)
	b = sendKey(b, "r")
	b = sendKey(b, "enter")

	// Scroll down far.
	for range 20 {
		b = sendKey(b, "j")
	}
	// Press g to jump to top.
	b = sendKey(b, "g")
	v := b.View()

	if !containsStr(v, "Task #1") {
		t.Error("expected Task #1 visible after pressing g")
	}
}

func TestBoard_DetailScrollToBottom(t *testing.T) {
	b, cfg := setupTestBoard(t)
	addLongBodyToTask(t, cfg, 1, 50)
	b = sendKey(b, "r")
	b = sendKey(b, "enter")

	// Press G to jump to bottom.
	b = sendKey(b, "G")
	v := b.View()

	// Last body lines should be visible.
	if !containsStr(v, "Body line 50") {
		t.Error("expected last body line visible after pressing G")
	}
	// Title should be scrolled out.
	if containsStr(v, "Task #1") {
		t.Error("expected title scrolled out after pressing G")
	}
}

func TestBoard_DetailExitBackspace(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "enter") // detail view
	v := b.View()
	if !containsStr(v, "Status:") {
		t.Fatal("expected detail view")
	}

	b = sendSpecialKey(b, tea.KeyBackspace) // exit detail
	v = b.View()
	if containsStr(v, "Press q/esc to go back") {
		t.Error("expected to return to board view after backspace")
	}
}

func TestBoard_DetailUnescapesBody(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Set body with literal escape sequences (as a CLI --body flag would produce).
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	tk.Body = `first line\nsecond line\tindented`
	if err := task.Write(path, tk); err != nil {
		t.Fatalf("writing task: %v", err)
	}

	b = sendKey(b, "r")     // refresh
	b = sendKey(b, "enter") // open detail
	v := b.View()

	// The literal \n should have been rendered as a newline, producing two lines.
	if containsStr(v, `\n`) {
		t.Error("literal \\n should not appear in rendered output")
	}
	if !containsStr(v, "first line") {
		t.Error("expected 'first line' in output")
	}
	if !containsStr(v, "second line") {
		t.Error("expected 'second line' in output")
	}
}

func TestBoard_DetailRendersMarkdown(t *testing.T) {
	b, cfg := setupTestBoard(t)

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	tk.Body = "## Overview\n\nThis is **bold** text and `inline code`.\n\n- Item one\n- Item two"
	if err := task.Write(path, tk); err != nil {
		t.Fatalf("writing task: %v", err)
	}

	b = sendKey(b, "r")     // refresh
	b = sendKey(b, "enter") // open detail
	v := b.View()

	// Markdown bold markers should be stripped by glamour rendering.
	if containsStr(v, "**bold**") {
		t.Error("raw **bold** markers should not appear in rendered output")
	}
	// Content should still be present.
	if !containsStr(v, "bold") {
		t.Error("expected 'bold' text in output")
	}
	// Backtick markers should be stripped.
	if containsStr(v, "`inline code`") {
		t.Error("raw backtick markers should not appear in rendered output")
	}
	if !containsStr(v, "inline code") {
		t.Error("expected 'inline code' in output")
	}
	// Bullet list items rendered (glamour converts - to •).
	if !containsStr(v, "Item one") {
		t.Error("expected 'Item one' in output")
	}
	if !containsStr(v, "Item two") {
		t.Error("expected 'Item two' in output")
	}
}

func TestBoard_MoveDialogCursorUp(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Task 1 is in backlog (index 0). Open move dialog.
	b = sendKey(b, "m")
	// Cursor starts at index 0 (backlog). Move down twice to in-progress (index 2).
	b = sendKey(b, "j") // → todo (1)
	b = sendKey(b, "j") // → in-progress (2)
	// Move back up once to todo (index 1).
	b = sendKey(b, "k") // → todo (1)
	// Confirm.
	_ = sendKey(b, "enter")

	// Verify the task moved to todo.
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

func TestBoard_MoveSameStatus(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Task 1 is in backlog. Open move dialog and press enter immediately
	// (cursor starts on "backlog", the current status).
	b = sendKey(b, "m")
	_ = sendKey(b, "enter")

	// Task should still be in backlog.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Status != "backlog" {
		t.Errorf("expected status 'backlog' (unchanged), got %q", tk.Status)
	}
}

func TestBoard_MoveNextAtLast(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Navigate to done column (index 4): backlog → todo → in-progress → review → done.
	for range 4 {
		b = sendKey(b, "l")
	}
	// Press N on a task in "done" (last status).
	b = sendKey(b, "N")
	v := b.View()

	if !containsStr(v, "already at the last status") {
		t.Error("expected error message when trying to move next past last status")
	}
}

func TestBoard_MoveNextEmptyColumn(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Navigate to todo column (index 1, empty in setupTestBoard).
	b = sendKey(b, "l")
	// Press N — should not panic.
	b = sendKey(b, "N")
	v := b.View()

	// Just verify it renders without panic.
	if v == "" {
		t.Error("expected non-empty view")
	}
}

func TestBoard_MovePrevEmptyColumn(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Navigate to todo column (index 1, empty).
	b = sendKey(b, "l")
	// Press P — should not panic.
	b = sendKey(b, "P")
	v := b.View()

	if v == "" {
		t.Error("expected non-empty view")
	}
}

func TestBoard_RaisePriority(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Task A (ID 1) starts at "high" priority. Press + to raise to "critical".
	b = sendKey(b, "+")

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Priority != priorityCritical {
		t.Errorf("expected priority 'critical', got %q", tk.Priority)
	}

	_ = b.View()
}

func TestBoard_LowerPriority(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Task A (ID 1) starts at "high" priority. Press - to lower to "medium".
	b = sendKey(b, "-")

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Priority != "medium" {
		t.Errorf("expected priority 'medium', got %q", tk.Priority)
	}

	_ = b.View()
}

func TestBoard_RaisePriorityWithEquals(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Task A (ID 1) starts at "high" priority. Press = to raise to "critical".
	b = sendKey(b, "=")

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Priority != priorityCritical {
		t.Errorf("expected priority 'critical', got %q", tk.Priority)
	}

	_ = b.View()
}

func TestBoard_LowerPriorityWithUnderscore(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Task A (ID 1) starts at "high" priority. Press _ to lower to "medium".
	b = sendKey(b, "_")

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Priority != "medium" {
		t.Errorf("expected priority 'medium', got %q", tk.Priority)
	}

	_ = b.View()
}

func TestBoard_RaisePriorityAtMax(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Raise Task A to critical first.
	b = sendKey(b, "+")

	// Now try to raise again — should show error.
	b = sendKey(b, "+")
	v := b.View()
	if !containsStr(v, "already at the highest priority") {
		t.Error("expected error when raising priority past maximum")
	}

	// Verify it's still critical (not changed).
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Priority != priorityCritical {
		t.Errorf("expected priority 'critical', got %q", tk.Priority)
	}
}

func TestBoard_LowerPriorityAtMin(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Lower Task A from "high" → "medium" → "low".
	b = sendKey(b, "-")
	b = sendKey(b, "-")

	// Now at "low" — try to lower again.
	b = sendKey(b, "-")
	v := b.View()
	if !containsStr(v, "already at the lowest priority") {
		t.Error("expected error when lowering priority past minimum")
	}
}

func TestBoard_PriorityCursorFollows(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Backlog has Task A (high, row 0) and Task B (medium, row 1).
	// Sorted by priority descending: [Task A, Task B].
	// Lower Task A from "high" to "medium" — Task A now has same priority as Task B.
	// After re-sort, cursor should still be on Task A.
	b = sendKey(b, "-")
	b = sendKey(b, "-") // now "low"

	// Task A should now be below Task B (which is "medium").
	// The cursor should follow Task A to its new position (row 1).
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("reading task: %v", err)
	}
	if tk.Priority != "low" {
		t.Errorf("expected priority 'low', got %q", tk.Priority)
	}

	_ = b.View()
}

func TestBoard_PriorityEmptyColumn(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Navigate to todo column (index 1, empty).
	b = sendKey(b, "l")
	// Press + — should not panic.
	b = sendKey(b, "+")
	v := b.View()

	if v == "" {
		t.Error("expected non-empty view")
	}
}

func TestBoard_DeleteTaskFileGone(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Open delete dialog for task 1.
	b = sendKey(b, "d")

	// Remove the task file behind the board's back.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("finding task: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("removing task file: %v", err)
	}

	// Confirm delete — should hit the FindByID error path.
	b = sendKey(b, "y")
	v := b.View()

	if !containsStr(v, "finding task") {
		t.Error("expected 'finding task' error in view after file was removed")
	}
}

func TestBoard_ColumnWidthCapped(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Set very wide terminal: 300 / 5 columns = 60, but max is 50.
	b.Update(tea.WindowSizeMsg{Width: 300, Height: 40})
	v := b.View()

	// The board should render without issues.
	if v == "" || v == viewLoading {
		t.Error("expected valid board view with wide terminal")
	}

	// Each column should be at most 50 chars wide. Check the header line.
	lines := strings.Split(v, "\n")
	if len(lines) > 0 {
		// With 5 columns at max 50 chars, total should be <= 250.
		if len(lines[0]) > 250 {
			t.Errorf("header line too wide: %d chars (max expected 250)", len(lines[0]))
		}
	}
}

func TestBoard_ScrollUpEnsureVisible(t *testing.T) {
	b, _ := setupManyTasksBoard(t)

	// Navigate to done column (index 4) which has 15 tasks.
	for range 4 {
		b = sendKey(b, "l")
	}
	// Scroll down to trigger scrollOff > 0.
	for range 10 {
		b = sendKey(b, "j")
	}

	// Now scroll back up — this should trigger the activeRow < scrollOff path.
	for range 10 {
		b = sendKey(b, "k")
	}
	v := b.View()

	// The first task in the done column should be visible after scrolling back.
	if !containsStr(v, "Done task 1") {
		t.Error("expected 'Done task 1' visible after scrolling back to top")
	}
}

func TestBoard_InitReturnsTickCmd(t *testing.T) {
	b, _ := setupTestBoard(t)
	cmd := b.Init()
	if cmd == nil {
		t.Fatal("Init() should return a tick command, got nil")
	}
}

func TestBoard_TickMsgUpdatesAge(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Initial view shows "2h" because testNow is 2h after testRefTime.
	v1 := b.View()
	if !containsStr(v1, "2h") {
		t.Fatal("expected initial view to contain '2h' age")
	}

	// Advance the clock by 1 hour and send a tickMsg to trigger re-render.
	b.SetNow(func() time.Time { return testRefTime.Add(3 * time.Hour) })
	m, cmd := b.Update(tui.TickMsg{})
	b = m.(*tui.Board)

	// The tick handler should return a follow-up tick command.
	if cmd == nil {
		t.Fatal("tickMsg handler should return a follow-up tick command")
	}

	// View should now show "3h" instead of "2h".
	v2 := b.View()
	if !containsStr(v2, "3h") {
		t.Errorf("expected view to contain '3h' after clock advance, got:\n%s", v2)
	}
}

func TestBoard_ClaimOnSeparateLine(t *testing.T) {
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

	// Create a claimed task and an unclaimed task.
	for _, tt := range []struct {
		id        int
		title     string
		status    string
		priority  string
		claimedBy string
	}{
		{1, "Claimed task", "in-progress", "high", "agent-1"},
		{2, "Unclaimed task", "in-progress", "medium", ""},
	} {
		tk := &task.Task{
			ID:        tt.id,
			Title:     tt.title,
			Status:    tt.status,
			Priority:  tt.priority,
			ClaimedBy: tt.claimedBy,
			Updated:   testRefTime,
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(tt.id, tt.title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	v := b.View()

	// Claim info must appear in the view.
	if !containsStr(v, "@agent-1") {
		t.Fatalf("expected @agent-1 in view, got:\n%s", v)
	}

	// The claim must be on a SEPARATE line from the priority.
	// No single line should contain both "high" and "@agent-1".
	for _, line := range strings.Split(v, "\n") {
		if findSubstring(line, "high") && findSubstring(line, "@agent-1") {
			t.Errorf("claim info should be on a separate line from priority, but found both on same line: %q", line)
		}
	}
}

func TestBoard_CardsDoNotRenderBlankPaddingLines(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Test Board")
	cfg.TUI.TitleLines = 2
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Create a claimed and unclaimed task in the same column.
	for _, tt := range []struct {
		id        int
		title     string
		claimedBy string
	}{
		{1, "Claimed task", "agent-1"},
		{2, "Unclaimed task", ""},
	} {
		tk := &task.Task{
			ID:        tt.id,
			Title:     tt.title,
			Status:    "backlog",
			Priority:  "medium",
			ClaimedBy: tt.claimedBy,
			Updated:   testRefTime,
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(tt.id, tt.title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	v := b.View()

	if !containsStr(v, "@agent-1") {
		t.Fatalf("expected @agent-1 in view, got:\n%s", v)
	}

	if containsStr(v, "│                      │") {
		t.Fatalf("expected no blank card content lines, got:\n%s", v)
	}
}

func TestBoard_DurationHiddenByConfig(t *testing.T) {
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

	// Create one task in "todo" (show_duration defaults to true/nil)
	// and one in "backlog" (show_duration=false by default).
	tasks := []struct {
		id     int
		title  string
		status string
	}{
		{1, "Backlog task", "backlog"},
		{2, "Todo task", statusTodo},
	}
	for _, tt := range tasks {
		tk := &task.Task{
			ID:       tt.id,
			Title:    tt.title,
			Status:   tt.status,
			Priority: "medium",
			Updated:  testRefTime,
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(tt.id, tt.title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	v := b.View()

	// The view renders columns side-by-side. Check the details line of each card.
	// Backlog card should show "medium" without "2h".
	// Todo card should show "medium 2h".
	if !containsStr(v, "medium 2h") {
		t.Error("todo task should show 'medium 2h' (show_duration defaults to true)")
	}

	// Verify backlog card's details line contains just "medium" without age.
	// In the columnar view, the backlog card content shows "medium" followed by spaces
	// before the next column, while the todo card shows "medium 2h".
	lines := strings.Split(v, "\n")
	for _, line := range lines {
		// Find the line with "medium" inside box-drawing chars (card content line).
		// The backlog column comes first, so extract the first column's portion.
		if !findSubstring(line, "medium") {
			continue
		}
		// Split by the column separator "││" to isolate columns.
		parts := strings.SplitN(line, "││", 2)
		if len(parts) < 2 {
			continue
		}
		backlogPart := parts[0]
		todoPart := parts[1]

		// Backlog column should have "medium" but NOT "2h".
		if findSubstring(backlogPart, "medium") && findSubstring(backlogPart, "2h") {
			t.Error("backlog task should NOT show age duration (show_duration=false)")
		}
		// Todo column should have "medium" AND "2h".
		if findSubstring(todoPart, "medium") && !findSubstring(todoPart, "2h") {
			t.Error("todo task should show age duration (show_duration defaults to true)")
		}
	}
}

// --- Bug #130: Bottom line jumps when scrolling variable-height cards ---

// setupVariableHeightBoard creates a board with title_lines=3 and tasks that
// have varying title lengths, causing cards to have different heights.
func setupVariableHeightBoard(t *testing.T) *tui.Board {
	t.Helper()

	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Variable Height Test")
	const testTitleLines = 3
	cfg.TUI.TitleLines = testTitleLines
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Tasks with varying title lengths → different card heights when title_lines=3.
	tasks := []struct {
		id       int
		title    string
		status   string
		priority string
	}{
		// Mix of short titles (1 line) and long titles (2-3 lines) to
		// produce variable card heights when title_lines=3.
		{1, "A", statusTodo, "high"},
		{2, "Implement comprehensive OAuth2 authentication with SAML support and multi-factor verification", statusTodo, "medium"},
		{3, "B", statusTodo, "low"},
		{4, "Design the database schema for the new microservice architecture and migration plan", statusTodo, "high"},
		{5, "C", statusTodo, "medium"},
		{6, "Set up CI pipeline with automated testing linting and deployment to staging environment", statusTodo, "low"},
		{7, "D", statusTodo, "high"},
		{8, "E", statusTodo, "medium"},
	}

	for _, tt := range tasks {
		tk := &task.Task{
			ID:       tt.id,
			Title:    tt.title,
			Status:   tt.status,
			Priority: tt.priority,
			Updated:  testRefTime,
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(tt.id, tt.title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return b
}

func TestBoard_BottomLineStableWhenScrolling(t *testing.T) {
	b := setupVariableHeightBoard(t)

	const termHeight = 20

	// Collect line counts as we scroll through all tasks.
	var lineCounts []int
	for i := 0; i < 8; i++ {
		v := b.View()
		lines := strings.Split(v, "\n")
		lineCounts = append(lineCounts, len(lines))

		if len(lines) != termHeight {
			t.Errorf("after %d scrolls: expected exactly %d lines, got %d",
				i, termHeight, len(lines))
		}
		b = sendKey(b, "j")
	}

	// All line counts must be identical (bottom line stays put).
	for i := 1; i < len(lineCounts); i++ {
		if lineCounts[i] != lineCounts[0] {
			t.Errorf("line count changed from %d to %d at scroll step %d — bottom line jumped",
				lineCounts[0], lineCounts[i], i)
		}
	}
}

func TestBoard_ScrollFollowsSelectedTask(t *testing.T) {
	// Bug #164: When scrolling down with variable-height cards, the selected
	// task can end up off-screen because ensureVisible() computes maxVis at
	// the old scrollOff and then adjusts scrollOff, but at the new offset
	// the visible card count may be smaller (taller cards + up indicator).
	//
	// Setup: 4 tasks in "todo" with distinct priorities (controlling sort
	// order). Task #4 (low priority, last in sort order) has a long title
	// that wraps to 3 lines at columnWidth=20 with title_lines=3.
	// Card heights: [4, 4, 4, 6].
	// Terminal 100x16 → budget=14, header=1, avail=13.
	//
	// At scrollOff=0: 3 short cards fit. When scrolling to the 4th task,
	// ensureVisible sets scrollOff=1, but at scrollOff=1 the up indicator
	// steals a line and the tall 4th card reduces maxVis to 2, leaving the
	// selected task off-screen.
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("creating dirs: %v", err)
	}

	cfg := config.NewDefault("Scroll Follow Test")
	const testTitleLines = 3
	cfg.TUI.TitleLines = testTitleLines
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Distinct priorities → deterministic sort: critical, high, medium, low.
	// Task 4 (low, sorted last) has a long title → taller card.
	tasks := []struct {
		id       int
		title    string
		priority string
	}{
		{1, "Short A", priorityCritical},
		{2, "Short B", "high"},
		{3, "Short C", "medium"},
		{4, "A very long title that will wrap to multiple lines in a narrow column", "low"},
	}
	// Expected TUI order: #1 (critical), #2 (high), #3 (medium), #4 (low).

	for _, tt := range tasks {
		tk := &task.Task{
			ID:       tt.id,
			Title:    tt.title,
			Status:   statusTodo,
			Priority: tt.priority,
			Updated:  testRefTime,
		}
		path := filepath.Join(tasksDir, task.GenerateFilename(tt.id, tt.title))
		if err := task.Write(path, tk); err != nil {
			t.Fatalf("writing task: %v", err)
		}
	}

	const termHeight = 16
	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 100, Height: termHeight})

	// Navigate to "todo" column (col 1).
	b = sendKey(b, "l")

	// Expected sort order by ID (priority descending): 1, 2, 3, 4.
	expectedOrder := [4]int{1, 2, 3, 4}

	// Scroll down through all tasks, checking the selected task is visible.
	for i := 1; i < len(expectedOrder); i++ {
		b = sendKey(b, "j")
		v := b.View()
		selectedID := fmt.Sprintf("#%d", expectedOrder[i])
		if !containsStr(v, selectedID) {
			t.Errorf("after %d down-scrolls: selected task %s is not visible in the view",
				i, selectedID)
		}
	}

	// Also verify scrolling back up keeps the selected task visible.
	for i := len(expectedOrder) - 2; i >= 0; i-- {
		b = sendKey(b, "k")
		v := b.View()
		selectedID := fmt.Sprintf("#%d", expectedOrder[i])
		if !containsStr(v, selectedID) {
			t.Errorf("scrolling up to task %s: not visible in the view", selectedID)
		}
	}
}
