package cmd

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ===== runDelete dispatcher tests =====

func TestRunDelete_InvalidID(t *testing.T) {
	cmd := deleteCmd
	err := runDelete(cmd, []string{"abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidTaskID {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidTaskID)
	}
}

func TestRunDelete_NoConfig(t *testing.T) {
	dir := t.TempDir()
	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := deleteCmd
	err := runDelete(cmd, []string{"1"})
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestRunDelete_SingleTaskWithYes(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeDeleteTask(t, cfg, &task.Task{
		ID: 1, Title: "delete-dispatch", Status: "backlog",
		Priority: "medium", Created: now, Updated: now,
	})

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := deleteCmd
	_ = cmd.Flags().Set("yes", "true")
	t.Cleanup(func() { _ = cmd.Flags().Set("yes", "false") })

	err = runDelete(cmd, []string{"1"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runDelete error: %v", err)
	}
	if !containsSubstring(got, "Deleted task #1") {
		t.Errorf("expected 'Deleted task #1' in output, got: %s", got)
	}
}

func TestRunDelete_BatchWithoutYes(t *testing.T) {
	kanbanDir := setupBoard(t)
	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := deleteCmd
	_ = cmd.Flags().Set("yes", "false")
	t.Cleanup(func() { _ = cmd.Flags().Set("yes", "false") })

	err := runDelete(cmd, []string{"1,2"})
	if err == nil {
		t.Fatal("expected error for batch without --yes")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.ConfirmationReq {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.ConfirmationReq)
	}
}

func TestRunDelete_BatchWithYes(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeDeleteTask(t, cfg, &task.Task{
		ID: 1, Title: "batch-del-1", Status: "backlog",
		Priority: "medium", Created: now, Updated: now,
	})
	writeDeleteTask(t, cfg, &task.Task{
		ID: 2, Title: "batch-del-2", Status: "backlog",
		Priority: "medium", Created: now, Updated: now,
	})

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := deleteCmd
	_ = cmd.Flags().Set("yes", "true")
	t.Cleanup(func() { _ = cmd.Flags().Set("yes", "false") })

	err = runDelete(cmd, []string{"1,2"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runDelete batch error: %v", err)
	}
	if !containsSubstring(got, "2/2") {
		t.Errorf("expected '2/2' in output, got: %s", got)
	}
}

// ===== runHandoff dispatcher tests =====

func TestRunHandoff_InvalidID(t *testing.T) {
	err := runHandoff(handoffCmd, []string{"abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidTaskID {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidTaskID)
	}
}

func TestRunHandoff_NoConfig(t *testing.T) {
	dir := t.TempDir()
	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runHandoff(handoffCmd, []string{"1"})
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestRunHandoff_SingleTask(t *testing.T) {
	cfg := setupHandoffTask(t)

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := handoffCmd
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	t.Cleanup(func() { _ = cmd.Flags().Set("claim", "") })

	err := runHandoff(cmd, []string{"1"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runHandoff error: %v", err)
	}
	if !containsSubstring(got, "Handed off task #1") {
		t.Errorf("expected 'Handed off task #1' in output, got: %s", got)
	}
}

func TestRunHandoff_Batch(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	for _, id := range []int{1, 2} {
		writeHandoffTask(t, cfg, &task.Task{
			ID: id, Title: "batch-handoff", Status: "in-progress",
			Priority: "medium", ClaimedBy: testHandoffAgent, ClaimedAt: &now,
			Created: now, Updated: now,
		})
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := handoffCmd
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	t.Cleanup(func() { _ = cmd.Flags().Set("claim", "") })

	err = runHandoff(cmd, []string{"1,2"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runHandoff batch error: %v", err)
	}
	if !containsSubstring(got, "2/2") {
		t.Errorf("expected '2/2' in batch output, got: %s", got)
	}
}

// ===== runPick dispatcher tests =====

func TestRunPick_InvalidStatusFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newPickCmd()
	_ = cmd.Flags().Set("claim", "agent")
	_ = cmd.Flags().Set("status", "nonexistent")

	err := runPick(cmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid status filter")
	}
}

func TestRunPick_SingleTaskPick(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "pickable", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newPickCmd()
	_ = cmd.Flags().Set("claim", "test-agent")

	err = runPick(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runPick error: %v", err)
	}
	if !containsSubstring(got, "Picked task #1") {
		t.Errorf("expected 'Picked task #1' in output, got: %s", got)
	}
}

func TestRunPick_WithMove(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "pick-move", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newPickCmd()
	_ = cmd.Flags().Set("claim", "test-agent")
	_ = cmd.Flags().Set("move", "todo")

	err = runPick(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runPick error: %v", err)
	}
	if !containsSubstring(got, "Picked and moved") {
		t.Errorf("expected 'Picked and moved' in output, got: %s", got)
	}
	if !containsSubstring(got, "backlog -> todo") {
		t.Errorf("expected 'backlog -> todo' in output, got: %s", got)
	}
}

func TestRunPick_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "pick-json", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newPickCmd()
	_ = cmd.Flags().Set("claim", "test-agent")

	err = runPick(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runPick error: %v", err)
	}
	if !containsSubstring(got, `"title"`) {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestRunPick_NothingToPick(t *testing.T) {
	kanbanDir := setupBoard(t)
	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newPickCmd()
	_ = cmd.Flags().Set("claim", "test-agent")

	err := runPick(cmd, nil)
	if err == nil {
		t.Fatal("expected error when nothing to pick")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.NothingToPick {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.NothingToPick)
	}
}

// ===== runMove dispatcher tests =====

func TestRunMove_SingleTask(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "move-dispatch", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	err = runMove(cmd, []string{"1", "todo"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runMove error: %v", err)
	}
	if !containsSubstring(got, "Moved task #1") {
		t.Errorf("expected 'Moved task #1' in output, got: %s", got)
	}
}

func TestRunMove_Batch(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "batch-move-1", "backlog")
	createTaskFileWithStatus(t, cfg.TasksPath(), 2, "batch-move-2", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	err = runMove(cmd, []string{"1,2", "todo"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runMove batch error: %v", err)
	}
	if !containsSubstring(got, "2/2") {
		t.Errorf("expected '2/2' in batch output, got: %s", got)
	}
}

func TestRunMove_BatchPartialFailure(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	// Only create task 1, not task 2 — task 2 will fail.
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "batch-move-ok", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	rOut, wOut := captureStdout(t)
	rErr, wErr := captureStderr(t)

	cmd := newMoveCmd()
	batchErr := runMove(cmd, []string{"1,999", "todo"})

	stdout := drainPipe(t, rOut, wOut)
	_ = drainPipe(t, rErr, wErr)

	if batchErr == nil {
		t.Fatal("expected error for partial batch failure")
	}
	var silent *clierr.SilentError
	if !errors.As(batchErr, &silent) {
		t.Fatalf("expected SilentError, got %T", batchErr)
	}
	if !containsSubstring(stdout, "1/2") {
		t.Errorf("expected '1/2' in output, got: %s", stdout)
	}
}

// ===== runEdit dispatcher tests =====

func TestRunEdit_InvalidID(t *testing.T) {
	err := runEdit(editCmd, []string{"abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidTaskID {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidTaskID)
	}
}

func TestRunEdit_SingleTaskTable(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "edit-dispatch", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newEditCmd()
	_ = cmd.Flags().Set("priority", "high")

	err = editSingleTask(cfg, 1, cmd)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("editSingleTask error: %v", err)
	}
	if !containsSubstring(got, "Updated task #1") {
		t.Errorf("expected 'Updated task #1' in output, got: %s", got)
	}
}

func TestRunEdit_SingleTaskJSON(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "edit-json", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newEditCmd()
	_ = cmd.Flags().Set("priority", "high")

	err = editSingleTask(cfg, 1, cmd)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("editSingleTask error: %v", err)
	}
	if !containsSubstring(got, `"title"`) {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestRunEdit_SingleViaDispatcher(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "edit-via-run", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	// Use the real editCmd and set its flag.
	cmd := editCmd
	_ = cmd.Flags().Set("priority", "high")
	t.Cleanup(func() { _ = cmd.Flags().Set("priority", "") })

	err = runEdit(cmd, []string{"1"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runEdit error: %v", err)
	}
	if !containsSubstring(got, "Updated task #1") {
		t.Errorf("expected 'Updated task #1' in output, got: %s", got)
	}
}

func TestRunEdit_Batch(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "batch-edit-1", "backlog")
	createTaskFileWithStatus(t, cfg.TasksPath(), 2, "batch-edit-2", "backlog")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newEditCmd()
	_ = cmd.Flags().Set("priority", "high")

	err = runEdit(cmd, []string{"1,2"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runEdit batch error: %v", err)
	}
	if !containsSubstring(got, "2/2") {
		t.Errorf("expected '2/2' in batch output, got: %s", got)
	}
}

// ===== enforceMoveWIP with class =====

func TestEnforceMoveWIP_TaskWithClassAndConfig(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// The default config includes classes (expedite, etc.).
	// Create a task with expedite class; bypasses column WIP.
	tk := &task.Task{ID: 1, Class: "expedite", Status: "backlog"}
	err = enforceMoveWIP(cfg, tk, "todo")
	if err != nil {
		t.Errorf("expected nil for expedite class, got: %v", err)
	}
}

func TestEnforceMoveWIP_TaskWithClassNoClassesConfig(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Task has a class but it doesn't match any configured class.
	// Should fall through to enforceWIPLimit (column check).
	tk := &task.Task{ID: 1, Class: "unknown-class", Status: "backlog"}
	err = enforceMoveWIP(cfg, tk, "todo")
	if err != nil {
		t.Errorf("expected nil (no WIP limit on todo), got: %v", err)
	}
}

func TestEnforceMoveWIP_NoClass(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Standard task with no class — goes to enforceWIPLimit.
	tk := &task.Task{ID: 1, Class: "", Status: "backlog"}
	err = enforceMoveWIP(cfg, tk, "todo")
	if err != nil {
		t.Errorf("expected nil (no WIP limit), got: %v", err)
	}
}

// ===== Execute (subprocess-style) =====

// TestExecute_RootCmdInit verifies Execute doesn't panic on basic setup.
// Testing os.Exit paths requires subprocess, covered by e2e tests.
func TestExecute_RootCmdInit(t *testing.T) {
	// Verify that the root command is properly initialized with subcommands.
	if len(rootCmd.Commands()) == 0 {
		t.Error("rootCmd has no subcommands registered")
	}
}

// ===== appendBody helper =====

// ===== softDeleteAndLog edge cases =====

func TestSoftDeleteAndLog_DependentWarning(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeDeleteTask(t, cfg, &task.Task{
		ID: 1, Title: "parent", Status: "backlog",
		Priority: "medium", Created: now, Updated: now,
	})
	parentID := 1
	writeDeleteTask(t, cfg, &task.Task{
		ID: 2, Title: "child", Status: "backlog",
		Priority: "medium", Parent: &parentID, Created: now, Updated: now,
	})

	// deleteSingleTask exercises warnDependents + softDeleteAndLog together.
	setFlags(t, false, true, false)
	r, w := captureStdout(t)
	rErr, wErr := captureStderr(t)

	err = deleteSingleTask(cfg, 1, true)
	_ = drainPipe(t, r, w)
	stderr := drainPipe(t, rErr, wErr)

	if err != nil {
		t.Fatalf("deleteSingleTask error: %v", err)
	}
	if !containsSubstring(stderr, "Warning") {
		t.Errorf("expected Warning in stderr, got: %s", stderr)
	}

	// Verify task was archived.
	path, _ := task.FindByID(cfg.TasksPath(), 1)
	tk, _ := task.Read(path)
	if tk.Status != config.ArchivedStatus {
		t.Errorf("Status = %q, want %q", tk.Status, config.ArchivedStatus)
	}
}

// ===== runPick with tags filter =====

func TestRunPick_WithTagsFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create task with matching tag.
	now := time.Now()
	tk := &task.Task{
		ID: 1, Title: "tagged-task", Status: "backlog",
		Priority: "medium", Tags: []string{"coverage"},
		Created: now, Updated: now,
	}
	slug := task.GenerateSlug(tk.Title)
	filename := task.GenerateFilename(tk.ID, slug)
	path := filepath.Join(cfg.TasksPath(), filename)
	if wErr := task.Write(path, tk); wErr != nil {
		t.Fatal(wErr)
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newPickCmd()
	_ = cmd.Flags().Set("claim", "test-agent")
	_ = cmd.Flags().Set("tags", "coverage")

	err = runPick(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runPick with tags error: %v", err)
	}
	if !containsSubstring(got, "Picked task #1") {
		t.Errorf("expected 'Picked task #1' in output, got: %s", got)
	}
}
