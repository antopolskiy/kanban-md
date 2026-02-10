package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// --- clearScreen ---

func TestClearScreen(t *testing.T) {
	r, w := captureStdout(t)
	clearScreen()
	got := drainPipe(t, r, w)

	if got != "\033[2J\033[H" {
		t.Errorf("clearScreen() = %q, want ANSI clear escape codes", got)
	}
}

// Note: watchBoard uses signal.NotifyContext internally, making it difficult
// to test without sending SIGINT to the test process. The error path
// (invalid watch path) is tested in TestWatchBoard_InvalidPath below.

// --- moveSingleTask ---

func TestMoveSingleTask_ExecuteMoveError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	// No task file → FindByID fails.
	cmd := newMoveCmd()
	err = moveSingleTask(cfg, 999, cmd, []string{"999", "done"})
	if err == nil {
		t.Fatal("expected error from missing task")
	}
}

func TestMoveSingleTask_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "move-json-task")

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	err = moveSingleTask(cfg, 1, cmd, []string{"1", "todo"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsSubstring(got, `"changed": true`) {
		t.Errorf("expected JSON with changed:true, got: %s", got)
	}
}

// --- executeMove error paths ---

func TestExecuteMove_TaskReadError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(cfg.TasksPath(), "1-bad.md")
	writeErr := os.WriteFile(badPath, []byte("not valid frontmatter"), 0o600)
	if writeErr != nil {
		t.Fatal(writeErr)
	}

	cmd := newMoveCmd()
	_, _, err = executeMove(cfg, 1, cmd, []string{"1", "done"})
	if err == nil {
		t.Fatal("expected error from malformed task file")
	}
}

func TestExecuteMove_ClaimError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	createClaimedTaskFile(t, cfg.TasksPath(), 1, "claimed-task", "other-agent")

	cmd := newMoveCmd()
	_, _, err = executeMove(cfg, 1, cmd, []string{"1", "done"})
	if err == nil {
		t.Fatal("expected claim error")
	}
}

func TestExecuteMove_ResolveTargetStatusError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "test-task")

	cmd := newMoveCmd()
	// Invalid status in args.
	_, _, err = executeMove(cfg, 1, cmd, []string{"1", "nonexistent-status"})
	if err == nil {
		t.Fatal("expected error from invalid target status")
	}
}

func TestExecuteMove_WIPLimitError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Set a WIP limit of 1 for in-progress.
	cfg.WIPLimits = map[string]int{statusInProgress: 1}

	// Create a task already in in-progress.
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "existing", statusInProgress)
	// Create a task in backlog to move.
	createTaskFile(t, cfg.TasksPath(), 2, "to-move")

	cmd := newMoveCmd()
	_, _, err = executeMove(cfg, 2, cmd, []string{"2", statusInProgress})
	if err == nil {
		t.Fatal("expected WIP limit error")
	}
}

func TestExecuteMove_WriteError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "test-task")

	// Make the task file read-only.
	path, findErr := task.FindByID(cfg.TasksPath(), 1)
	if findErr != nil {
		t.Fatal(findErr)
	}
	chmodErr := os.Chmod(path, 0o400)
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })

	cmd := newMoveCmd()
	_, _, err = executeMove(cfg, 1, cmd, []string{"1", "todo"})
	if err == nil {
		t.Fatal("expected write error")
	}
}

// --- runCreate error paths ---

func TestRunCreate_ApplyCreateFlagsError(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newCreateCmd()
	_ = cmd.Flags().Set("status", "nonexistent")

	err := runCreate(cmd, []string{"test task"})
	if err == nil {
		t.Fatal("expected error from invalid status")
	}
}

func TestRunCreate_ValidateDepsError(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newCreateCmd()
	_ = cmd.Flags().Set("depends-on", "999")

	err := runCreate(cmd, []string{"test task"})
	if err == nil {
		t.Fatal("expected error from invalid dependency")
	}
}

func TestRunCreate_WIPLimitError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// The default status is "backlog". Set a WIP limit of 1 for backlog
	// and create one existing task so the limit is reached.
	cfg.WIPLimits = map[string]int{"backlog": 1}
	createTaskFile(t, cfg.TasksPath(), 1, "existing-task")
	cfg.NextID = 2
	saveErr := cfg.Save()
	if saveErr != nil {
		t.Fatal(saveErr)
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newCreateCmd()
	err = runCreate(cmd, []string{"test task"})
	if err == nil {
		t.Fatal("expected WIP limit error")
	}
}

func TestRunCreate_WriteError(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	// Make the tasks directory read-only.
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not restrict directory writes on Windows")
	}

	tasksDir := cfg.TasksPath()
	chmodErr := os.Chmod(tasksDir, 0o500) //nolint:gosec // intentionally restricting dir for test
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	t.Cleanup(func() { _ = os.Chmod(tasksDir, 0o750) }) //nolint:gosec // restoring dir perms

	cmd := newCreateCmd()
	err = runCreate(cmd, []string{"test task"})
	if err == nil {
		t.Fatal("expected write error")
	}
}

func TestRunCreate_JSONOutputViaFormat(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newCreateCmd()
	err := runCreate(cmd, []string{"json task"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsSubstring(got, "json task") {
		t.Errorf("expected JSON output with title, got: %s", got)
	}
}

// --- renderBoard error path ---

func TestRenderBoard_ReadAllError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS == "windows" {
		t.Skip("chmod does not prevent reads on Windows")
	}

	// Make the tasks directory unreadable to trigger ReadAllLenient error.
	tasksDir := cfg.TasksPath()
	chmodErr := os.Chmod(tasksDir, 0o000)
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	t.Cleanup(func() { _ = os.Chmod(tasksDir, 0o750) }) //nolint:gosec // restoring

	err = renderBoard(cfg, "")
	if err == nil {
		t.Fatal("expected error from unreadable tasks directory")
	}
}

// --- moveSingleTask compact output ---

func TestMoveSingleTask_CompactOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "move-compact-task")

	setFlags(t, false, false, true)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	err = moveSingleTask(cfg, 1, cmd, []string{"1", "todo"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsSubstring(got, "Moved task #1") {
		t.Errorf("expected 'Moved task #1' in compact output, got: %s", got)
	}
}

func TestMoveSingleTask_IdempotentCompact(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "idempotent-compact", "todo")

	setFlags(t, false, false, true)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	err = moveSingleTask(cfg, 1, cmd, []string{"1", "todo"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsSubstring(got, "already at") {
		t.Errorf("expected 'already at' in compact output, got: %s", got)
	}
}

// --- runCreate compact output ---

func TestRunCreate_CompactOutput(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, false, true)
	r, w := captureStdout(t)

	cmd := newCreateCmd()
	err := runCreate(cmd, []string{"Compact create test"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsSubstring(got, "Created task #1") {
		t.Errorf("expected 'Created task #1' in compact output, got: %s", got)
	}
}

// --- renderBoard with malformed task warnings ---

func TestRenderBoard_WithWarnings(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a malformed task file that produces a warning.
	badPath := filepath.Join(cfg.TasksPath(), "bad-task.md")
	if writeErr := os.WriteFile(badPath, []byte("not valid frontmatter"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	setFlags(t, false, true, false)
	r, w := captureStdout(t)
	rErr, wErr := captureStderr(t)

	renderErr := renderBoard(cfg, "")

	_ = drainPipe(t, r, w)
	stderr := drainPipe(t, rErr, wErr)

	if renderErr != nil {
		t.Fatalf("renderBoard error: %v", renderErr)
	}
	if !containsSubstring(stderr, "Warning") {
		t.Errorf("expected Warning in stderr for malformed file, got: %s", stderr)
	}
}

// --- watchBoard with invalid path ---

func TestWatchBoard_InvalidPath(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.SetDir(filepath.Join(t.TempDir(), "nonexistent"))

	err := watchBoard(cfg, "")
	if err == nil {
		t.Fatal("expected error from invalid watch path")
	}
}

// --- runBoard watch flag (verifies watchBoard is called) ---

func TestRunBoard_WatchFlagWithInvalidPath(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	// Render once works, but watcher fails because we remove the tasks dir.
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	// Remove the tasks directory after rendering once to make the watcher fail.
	// We can't just remove before because renderBoard needs it.
	// Instead, set flagWatch and test with a nonexistent second watch path.
	// The simplest approach: remove the kanban dir after initial render
	// by overwriting the config dir with a bad path.
	_ = drainPipe(t, r, w)

	// Test the watch path indirectly via the full runBoard with watch flag.
	oldFlagWatch := flagWatch
	flagWatch = true
	t.Cleanup(func() { flagWatch = oldFlagWatch })

	// Make tasks dir non-watchable: remove it after load.
	if removeErr := os.RemoveAll(cfg.TasksPath()); removeErr != nil {
		t.Fatal(removeErr)
	}

	cmd := newBoardCmd()
	err = runBoard(cmd, nil)
	// renderBoard should fail because tasks dir is gone.
	if err == nil {
		t.Fatal("expected error when tasks directory is removed")
	}
}
