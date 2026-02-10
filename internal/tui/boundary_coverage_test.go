package tui_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/antopolskiy/kanban-md/internal/tui"
)

const statusArchived = "archived"

// setupSingleTaskBoard creates a minimal board with one task in the backlog
// at the given priority. Used by multiple boundary tests to avoid code duplication.
func setupSingleTaskBoard(t *testing.T, title, priority string) *tui.Board {
	t.Helper()

	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewDefault("Test Board")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{
		ID:       1,
		Title:    title,
		Status:   "backlog",
		Priority: priority,
		Updated:  testRefTime,
	}
	path := filepath.Join(tasksDir, task.GenerateFilename(1, title))
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	return b
}

// makeReadOnly sets a file to read-only and registers a cleanup to restore permissions.
func makeReadOnly(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0o444); err != nil { //nolint:gosec // test intentionally uses restrictive perms
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(path, 0o600)
	})
}

// restoreReadable restores a read-only file to readable/writable for verification.
func restoreReadable(path string) {
	_ = os.Chmod(path, 0o600)
}

// --- tickCmd: verify Init() returns a cmd that produces TickMsg ---

func TestBoundary_TickCmd_ProducesTickMsg(t *testing.T) {
	b, _ := setupTestBoard(t)
	cmd := b.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd, expected tickCmd")
	}
	// Execute the cmd — it returns a function wrapping tea.Tick.
	// tea.Tick returns a channel-based cmd; we cannot easily extract the
	// inner msg without running the full tea runtime. However, we can verify
	// that Update(TickMsg{}) returns another non-nil cmd (the follow-up tick),
	// proving the tick chain works end to end.
	m, followUp := b.Update(tui.TickMsg{})
	if followUp == nil {
		t.Error("expected follow-up tick cmd from TickMsg, got nil")
	}
	if m == nil {
		t.Error("expected non-nil model from TickMsg")
	}
}

// --- currentColumn: activeCol >= len(columns) returns nil ---

func TestBoundary_CurrentColumn_NoColumns(t *testing.T) {
	// Create a config with NO statuses to produce a board with zero columns.
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewDefault("No Columns Board")
	cfg.Statuses = nil // no statuses -> no columns
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	b := tui.NewBoard(cfg)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// With no columns, View() should show the "No statuses configured." message.
	v := b.View()
	if !containsStr(v, "No statuses") {
		t.Errorf("expected 'No statuses' message for board with no columns, got:\n%s", v)
	}

	// Navigation should not panic with zero columns.
	b = sendKey(b, "j")
	b = sendKey(b, "k")
	b = sendKey(b, "l")
	b = sendKey(b, "h")
	_ = b.View()

	// Move operations should not panic (selectedTask returns nil via currentColumn nil).
	b = sendKey(b, "N")
	b = sendKey(b, "P")
	b = sendKey(b, "+")
	b = sendKey(b, "-")

	// Create should be a no-op (currentColumn nil).
	b = sendKey(b, "c")
	v = b.View()
	if containsStr(v, "Create task in") {
		t.Error("expected no create dialog when there are no columns")
	}

	// Delete and move dialogs should not open (no task selected).
	b = sendKey(b, "d")
	b = sendKey(b, "m")
	_ = b.View()
}

// --- indexOf: boundary where status is at first position ---

func TestBoundary_MovePrev_AtFirstStatus(t *testing.T) {
	b := setupSingleTaskBoard(t, "First status task", "medium")

	// Task is at "backlog" (first board status). movePrev should show error.
	b = sendKey(b, "P")
	v := b.View()
	if !containsStr(v, "already at the first status") {
		t.Errorf("expected 'already at the first status' error, got:\n%s", v)
	}
}

// --- executeMove: write error path ---

func TestBoundary_ExecuteMove_WriteError(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Make the task file read-only so Write fails.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatal(err)
	}
	makeReadOnly(t, path)

	// Open move dialog and move to "todo".
	b = sendKey(b, "m")
	b = sendKey(b, "j") // cursor to "todo"
	b = sendSpecialKey(b, tea.KeyEnter)

	// The error from executeMove is set in b.err but immediately cleared by
	// the subsequent loadTasks() call. The key behavior is that the task
	// status gets reverted after the write failure.
	_ = b.View()

	// The task status should be reverted on disk (still backlog, not todo).
	restoreReadable(path)
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Status != "backlog" {
		t.Errorf("expected status reverted to 'backlog' after write error, got %q", tk.Status)
	}
}

// --- executeMove: move to terminal status (done) ---

func TestBoundary_ExecuteMove_ToTerminalStatus(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Open move dialog and move task 1 from backlog to done.
	b = sendKey(b, "m")
	// Navigate to "done" in the status list (backlog->todo->in-progress->review->done).
	b = sendKey(b, "j") // todo
	b = sendKey(b, "j") // in-progress
	b = sendKey(b, "j") // review
	b = sendKey(b, "j") // done
	b = sendSpecialKey(b, tea.KeyEnter)

	// Verify the task was moved to done.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Status != "done" {
		t.Errorf("expected status 'done', got %q", tk.Status)
	}

	_ = b.View()
}

// --- executeCreate: write error (read-only tasks directory) ---

func TestBoundary_ExecuteCreate_WriteError(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewDefault("Write Err Board")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{
		ID:       1,
		Title:    "Existing task",
		Status:   "backlog",
		Priority: "medium",
		Updated:  testRefTime,
	}
	path := filepath.Join(tasksDir, task.GenerateFilename(1, "existing-task"))
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if runtime.GOOS == "windows" {
		t.Skip("chmod does not restrict directory writes on Windows")
	}

	if err := os.Chmod(tasksDir, 0o555); err != nil { //nolint:gosec // test intentionally restricts dir
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(tasksDir, 0o750) //nolint:gosec // restore in cleanup
	})

	// Open create dialog and type a title.
	b = sendKey(b, "c")
	for _, ch := range "Fail task" {
		m, _ := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		b = m.(*tui.Board)
	}
	b = sendSpecialKey(b, tea.KeyEnter)

	v := b.View()
	if !containsStr(v, "creating task") {
		t.Errorf("expected 'creating task' error in view, got:\n%s", v)
	}
}

// --- executeCreate: config save error ---

func TestBoundary_ExecuteCreate_ConfigSaveError(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewDefault("Test Board")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{
		ID:       1,
		Title:    "Existing task",
		Status:   "backlog",
		Priority: "medium",
		Updated:  testRefTime,
	}
	path := filepath.Join(tasksDir, task.GenerateFilename(1, "existing-task"))
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}

	initialNextID := cfg.NextID

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Make config file read-only so cfg.Save() fails after task creation.
	cfgPath := filepath.Join(kanbanDir, "config.yml")
	makeReadOnly(t, cfgPath)

	// Open create dialog and type a title.
	b = sendKey(b, "c")
	for _, ch := range "Config fail" {
		m, _ := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		b = m.(*tui.Board)
	}
	b = sendSpecialKey(b, tea.KeyEnter)

	// The error from cfg.Save() is set but cleared by the subsequent
	// loadTasks() call. The task file was still created on disk (the write
	// succeeded), but the config wasn't updated. Verify the config on disk
	// still has the old NextID (because Save failed).
	restoreReadable(cfgPath)
	reloaded, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.NextID != initialNextID {
		t.Errorf("expected NextID on disk to be %d (unchanged after save error), got %d",
			initialNextID, reloaded.NextID)
	}

	// The task file should still exist (write succeeded before save).
	v := b.View()
	if !containsStr(v, "Config fail") {
		t.Error("expected created task visible despite config save error")
	}
}

// --- executeDelete: task.Read failure (corrupt file) ---

func TestBoundary_ExecuteDelete_ReadError(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Start delete dialog for task 1.
	b = sendKey(b, "d")
	v := b.View()
	if !containsStr(v, "Delete task?") {
		t.Fatalf("expected delete confirmation, got:\n%s", v)
	}

	// Corrupt the task file so task.Read fails.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not valid yaml frontmatter"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Confirm delete.
	b = sendKey(b, "y")
	v = b.View()
	if !containsStr(v, "reading task") {
		t.Errorf("expected 'reading task' error in view, got:\n%s", v)
	}
}

// --- executeDelete: task.Write failure ---

func TestBoundary_ExecuteDelete_WriteError(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Start delete dialog for task 1.
	b = sendKey(b, "d")

	// Make the task file read-only so Write fails during archiving.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatal(err)
	}
	makeReadOnly(t, path)

	// Confirm delete -- Write should fail, but loadTasks clears b.err.
	b = sendKey(b, "y")
	_ = b.View()

	// The task should NOT be archived on disk since the write failed.
	restoreReadable(path)
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Status == statusArchived {
		t.Error("expected task NOT to be archived after write error")
	}
}

// --- handleMoveStart: verifying cursor positioning ---

func TestBoundary_HandleMoveStart_CursorOnCurrentStatus(t *testing.T) {
	b := setupSingleTaskBoard(t, "Normal task", "medium")

	// Open move dialog. The cursor should start at the backlog position.
	b = sendKey(b, "m")
	v := b.View()
	if !containsStr(v, "Move #1") {
		t.Fatalf("expected move dialog, got:\n%s", v)
	}
	// The "(current)" marker should be next to "backlog".
	if !containsStr(v, "backlog (current)") {
		t.Error("expected '(current)' marker on backlog status")
	}
}

// --- handleCreateStart: on empty column (task list empty but column exists) ---

func TestBoundary_CreateFromEmptyColumn(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Navigate to "todo" column (empty in default setup).
	b = sendKey(b, "l")

	// Open create dialog.
	b = sendKey(b, "c")
	v := b.View()
	if !containsStr(v, "Create task in todo") {
		t.Errorf("expected create dialog for 'todo' column, got:\n%s", v)
	}

	// Cancel.
	b = sendSpecialKey(b, tea.KeyEscape)
	v = b.View()
	if containsStr(v, "Create task in") {
		t.Error("expected board view after cancel")
	}
}

// --- executePriorityChange: write error ---

func TestBoundary_PriorityChange_WriteError(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Make task file read-only so priority write fails.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatal(err)
	}
	makeReadOnly(t, path)

	// Raise priority -- should fail on write and revert.
	b = sendKey(b, "+")
	v := b.View()
	if !containsStr(v, "updating priority") {
		t.Errorf("expected 'updating priority' error in view, got:\n%s", v)
	}

	// Verify priority was reverted (still "high", not "critical").
	restoreReadable(path)
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Priority != "high" {
		t.Errorf("expected priority reverted to 'high', got %q", tk.Priority)
	}
}

// --- executePriorityChange: lower at already lowest shows error ---

func TestBoundary_PriorityAlreadyLowest(t *testing.T) {
	b := setupSingleTaskBoard(t, "Lowest task", "low")

	// Try to lower priority -- should show error.
	b = sendKey(b, "-")
	v := b.View()
	if !containsStr(v, "already at the lowest priority") {
		t.Errorf("expected 'already at the lowest priority' error, got:\n%s", v)
	}
}

// --- executePriorityChange: raise at already highest shows error ---

func TestBoundary_PriorityAlreadyHighest(t *testing.T) {
	b := setupSingleTaskBoard(t, "Critical task", "critical")

	// Try to raise priority -- should show error.
	b = sendKey(b, "+")
	v := b.View()
	if !containsStr(v, "already at the highest priority") {
		t.Errorf("expected 'already at the highest priority' error, got:\n%s", v)
	}
}

// --- Update: Ctrl+C from non-board views ---

func TestBoundary_CtrlC_FromDetailView(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Enter detail view.
	b = sendKey(b, "enter")
	v := b.View()
	if !containsStr(v, "Status:") {
		t.Fatal("expected detail view")
	}

	// Ctrl+C from detail view should quit.
	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit cmd from Ctrl+C in detail view")
	}
}

func TestBoundary_CtrlC_FromMoveView(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Open move dialog.
	b = sendKey(b, "m")
	v := b.View()
	if !containsStr(v, "Move") {
		t.Fatal("expected move dialog")
	}

	// Ctrl+C from move dialog should quit.
	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit cmd from Ctrl+C in move dialog")
	}
}

func TestBoundary_CtrlC_FromDeleteView(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Open delete confirmation.
	b = sendKey(b, "d")
	v := b.View()
	if !containsStr(v, "Delete task?") {
		t.Fatal("expected delete confirmation")
	}

	// Ctrl+C from delete confirmation should quit.
	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit cmd from Ctrl+C in delete dialog")
	}
}

func TestBoundary_CtrlC_FromHelpView(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Open help.
	b = sendKey(b, "?")
	v := b.View()
	if !containsStr(v, "Keyboard Shortcuts") {
		t.Fatal("expected help view")
	}

	// Ctrl+C from help view should quit.
	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit cmd from Ctrl+C in help view")
	}
}

func TestBoundary_CtrlC_FromCreateView(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Open create dialog.
	b = sendKey(b, "c")
	v := b.View()
	if !containsStr(v, "Create task in") {
		t.Fatal("expected create dialog")
	}

	// Ctrl+C from create dialog should quit.
	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit cmd from Ctrl+C in create dialog")
	}
}

// --- Move dialog: cursor boundary (can't go above first or below last) ---

func TestBoundary_MoveDialog_CursorBounds(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Open move dialog.
	b = sendKey(b, "m")

	// Move cursor up past the top (cursor should stay at 0).
	b = sendKey(b, "k")
	b = sendKey(b, "k")
	v := b.View()
	// The ">" cursor should still be on the first status.
	if !containsStr(v, "> backlog") {
		t.Error("expected cursor on backlog after pressing k at top")
	}

	// Move cursor down to the last status.
	names := []string{"backlog", "todo", "in-progress", "review", "done", statusArchived}
	for range len(names) - 1 {
		b = sendKey(b, "j")
	}
	// Move down past the last status.
	b = sendKey(b, "j")
	b = sendKey(b, "j")
	v = b.View()
	// The ">" cursor should be on the last status.
	if !containsStr(v, "> "+statusArchived) {
		t.Errorf("expected cursor on last status after pressing j past end, got:\n%s", v)
	}

	// Cancel.
	b = sendSpecialKey(b, tea.KeyEsc)
	_ = b.View()
}

// --- Move dialog: cancel with 'q' key ---

func TestBoundary_MoveDialog_CancelWithQ(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "m")
	v := b.View()
	if !containsStr(v, "Move") {
		t.Fatal("expected move dialog")
	}

	b = sendKey(b, "q")
	v = b.View()
	// Should be back in board view.
	if !containsStr(v, "Task A") {
		t.Error("expected board view after canceling move with q")
	}
}

// --- Delete confirmation: cancel with 'N' (uppercase) ---

func TestBoundary_DeleteCancel_UppercaseN(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "d")
	b = sendKey(b, "N")
	v := b.View()
	if !containsStr(v, "Task A") {
		t.Error("expected board view after canceling delete with N")
	}
}

// --- Delete confirmation: cancel with 'q' ---

func TestBoundary_DeleteCancel_WithQ(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "d")
	b = sendKey(b, "q")
	v := b.View()
	if !containsStr(v, "Task A") {
		t.Error("expected board view after canceling delete with q")
	}
}

// --- Delete confirmation: confirm with 'Y' (uppercase) ---

func TestBoundary_DeleteConfirm_UppercaseY(t *testing.T) {
	b, cfg := setupTestBoard(t)

	b = sendKey(b, "d")
	b = sendKey(b, "Y")

	// Task 1 should be archived.
	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("expected task 1 to exist: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Status != statusArchived {
		t.Errorf("expected status %q, got %q", statusArchived, tk.Status)
	}

	_ = b.View()
}

// --- executeMove: nil task guard ---

func TestBoundary_ExecuteMove_NilTask(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Navigate to an empty column first.
	b = sendKey(b, "l") // todo (empty)

	// Directly press N/P -- selectedTask is nil, should be no-op.
	b = sendKey(b, "N")
	b = sendKey(b, "P")
	v := b.View()
	if v == "" {
		t.Error("expected non-empty view")
	}
}

// --- Create dialog: backspace on empty input ---

func TestBoundary_Create_BackspaceOnEmpty(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")
	// Backspace on empty input should not panic.
	b = sendSpecialKey(b, tea.KeyBackspace)
	v := b.View()
	if !containsStr(v, "Create task in") {
		t.Error("expected create dialog still showing after backspace on empty")
	}
}

// --- View: columnWidth with zero width ---

func TestBoundary_ColumnWidth_ZeroWidth(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Set width to 0.
	b.Update(tea.WindowSizeMsg{Width: 0, Height: 40})
	v := b.View()
	// Width=0 triggers "Loading..." in View().
	if v != viewLoading {
		t.Errorf("expected %q with zero width, got: %q", viewLoading, v[:min(len(v), 100)])
	}
}

// --- handleNavigation: arrow key aliases ---

func TestBoundary_Navigation_ArrowKeys(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Test arrow key equivalents: left/right/up/down.
	b = sendKey(b, "right")
	b = sendKey(b, "right")
	b = sendKey(b, "left")
	b = sendKey(b, "down")
	b = sendKey(b, "up")
	v := b.View()
	if v == "" || v == viewLoading {
		t.Error("expected valid view after arrow key navigation")
	}
}

// --- loadTasks: handles corrupt/unreadable task gracefully ---

func TestBoundary_LoadTasks_LenientParsing(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	tasksDir := filepath.Join(kanbanDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewDefault("Test Board")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Create a valid task.
	tk := &task.Task{
		ID:       1,
		Title:    "Valid task",
		Status:   "backlog",
		Priority: "medium",
		Updated:  testRefTime,
	}
	path := filepath.Join(tasksDir, task.GenerateFilename(1, "valid-task"))
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}

	// Create a corrupt file in the tasks directory.
	corruptPath := filepath.Join(tasksDir, "002-corrupt.md")
	if err := os.WriteFile(corruptPath, []byte("not valid frontmatter"), 0o600); err != nil {
		t.Fatal(err)
	}

	b := tui.NewBoard(cfg)
	b.SetNow(testNow)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Board should still load (ReadAllLenient skips corrupt files).
	v := b.View()
	if !containsStr(v, "Valid task") {
		t.Errorf("expected 'Valid task' visible despite corrupt file, got:\n%s", v)
	}
}

// --- handleBoardKey: q key quits ---

func TestBoundary_QKeyQuits(t *testing.T) {
	b, _ := setupTestBoard(t)
	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected quit cmd from 'q' key")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

// --- WatchPaths: same dir and tasks dir ---

func TestBoundary_WatchPaths_SameDir(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := dir

	cfg := config.NewDefault("Test Board")
	cfg.SetDir(kanbanDir)
	cfg.TasksDir = "." // tasks in same dir as kanban
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	b := tui.NewBoard(cfg)

	paths := b.WatchPaths()
	// When Dir() == TasksPath(), should only return 1 path.
	if cfg.Dir() == cfg.TasksPath() && len(paths) != 1 {
		t.Errorf("expected 1 watch path when Dir == TasksPath, got %d: %v", len(paths), paths)
	}
}

// --- Detail view: nil detailTask guard ---

func TestBoundary_DetailView_NilTask(t *testing.T) {
	// Verify the board handles entering detail from an empty column gracefully.
	b, _ := setupTestBoard(t)
	b = sendKey(b, "l") // move to empty "todo" column
	b = sendSpecialKey(b, tea.KeyEnter)
	v := b.View()
	// Should stay on board view (enter is a no-op for empty column).
	if containsStr(v, "No task selected") {
		t.Error("should not show 'No task selected' from board navigation")
	}
}

// --- Due date displayed on card ---

func TestBoundary_CardShowsDueDate(t *testing.T) {
	b := setupMetadataBoard(t)

	// Navigate to in-progress column where the metadata task lives.
	b = sendKey(b, "l") // todo
	b = sendKey(b, "l") // in-progress

	v := b.View()
	// Card should display the due date.
	if !containsStr(v, "due:2026-03-15") {
		t.Errorf("expected due date on card, got:\n%s", v)
	}
}

// --- Blocked card style is applied ---

func TestBoundary_BlockedCardRendered(t *testing.T) {
	b := setupMetadataBoard(t)

	// Navigate to in-progress column.
	b = sendKey(b, "l") // todo
	b = sendKey(b, "l") // in-progress

	// The task is blocked, so BLOCKED should appear in detail view.
	b = sendSpecialKey(b, tea.KeyEnter)
	v := b.View()
	if !containsStr(v, "BLOCKED") {
		t.Error("expected BLOCKED indicator in detail view for blocked task")
	}
}
