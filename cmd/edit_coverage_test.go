package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// --- runEdit error paths ---

func TestRunEdit_NoConfig(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "updated")
	err := runEdit(cmd, []string{"1"})
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestRunEdit_NoChanges(t *testing.T) {
	kanbanDir := setupBoard(t)
	createTaskFile(t, filepath.Join(kanbanDir, "tasks"), 1, "existing-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newEditCmd()
	// No flags set — should trigger "no changes specified".
	err := runEdit(cmd, []string{"1"})

	_ = drainPipe(t, r, w)

	if err == nil {
		t.Fatal("expected error for no changes")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.NoChanges {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.NoChanges)
	}
}

func TestRunEdit_TaskNotFound(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "updated")
	err := runEdit(cmd, []string{"999"})

	_ = drainPipe(t, r, w)

	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.TaskNotFound {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.TaskNotFound)
	}
}

func TestRunEdit_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	createTaskFile(t, filepath.Join(kanbanDir, "tasks"), 1, "task-to-edit")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false) // JSON output
	r, w := captureStdout(t)

	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "edited title")
	err := runEdit(cmd, []string{"1"})

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runEdit error: %v", err)
	}
	if !containsSubstring(got, "edited title") {
		t.Errorf("expected JSON with 'edited title', got: %s", got)
	}
}

// --- writeAndRename error paths ---

func TestWriteAndRename_WriteError(t *testing.T) {
	badPath := filepath.Join(t.TempDir(), "nonexistent", "dir", "task.md")
	tk := &task.Task{ID: 1, Title: "test", Status: "backlog", Priority: "medium"}

	_, err := task.WriteAndRename(badPath, tk, "test")
	if err == nil {
		t.Fatal("expected error when write fails")
	}
	if !containsSubstring(err.Error(), "writing task") {
		t.Errorf("error = %v, want to contain 'writing task'", err)
	}
}

func TestWriteAndRename_RemoveError(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "001-old-title.md")
	tk := &task.Task{ID: 1, Title: "new title", Status: "backlog", Priority: "medium"}

	// Write the initial file so writeAndRename can write the new path.
	if err := task.Write(oldPath, tk); err != nil {
		t.Fatal(err)
	}

	// Remove the old file before writeAndRename tries to remove it.
	if err := os.Remove(oldPath); err != nil {
		t.Fatal(err)
	}

	_, err := task.WriteAndRename(oldPath, tk, "old title")
	if err == nil {
		t.Fatal("expected error when remove fails")
	}
	if !containsSubstring(err.Error(), "removing old file") {
		t.Errorf("error = %v, want to contain 'removing old file'", err)
	}
}

// --- applySimpleEditFlags uncovered ---

func TestApplySimpleEditFlags_InvalidClass(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("class", "nonexistent-class")
	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	_, err := applySimpleEditFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error for invalid class")
	}
}

func TestApplySimpleEditFlags_InvalidPriority(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("priority", "bogus")
	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	_, err := applySimpleEditFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

// --- applyTimestampFlags uncovered ---

func TestApplyTimestampFlags_InvalidCompletedDate(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("completed", "not-a-date")
	tk := &task.Task{}

	_, err := applyTimestampFlags(cmd, tk)
	if err == nil {
		t.Fatal("expected error for invalid completed date")
	}
}
