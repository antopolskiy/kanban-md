package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
)

func TestRunCreate_WithAssigneeAndTags(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newCreateCmd()
	_ = cmd.Flags().Set("assignee", "alice")
	_ = cmd.Flags().Set("tags", "bug,urgent")

	err := runCreate(cmd, []string{"Task with extras"})

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runCreate error: %v", err)
	}
	if !containsSubstring(got, "Assignee: alice") {
		t.Errorf("expected assignee in output, got: %s", got)
	}
	if !containsSubstring(got, "Tags: bug, urgent") {
		t.Errorf("expected tags in output, got: %s", got)
	}
}

func TestRunCreate_WIPLimitExceeded(t *testing.T) {
	kanbanDir := setupBoard(t)

	// Set a WIP limit of 1 on backlog (the default status for new tasks).
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.WIPLimits = map[string]int{"backlog": 1}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}

	// Create one task to fill the WIP limit.
	createTaskFile(t, cfg.TasksPath(), 1, "first-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newCreateCmd()
	err = runCreate(cmd, []string{"Second task should fail"})

	_ = drainPipe(t, r, w)

	if err == nil {
		t.Fatal("expected WIP limit error")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.WIPLimitExceeded {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.WIPLimitExceeded)
	}
}

func TestRunCreate_ClassAwareWIPLimit(t *testing.T) {
	kanbanDir := setupBoard(t)

	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	// Add an "expedite" class with WIP limit of 1.
	cfg.Classes = []config.ClassConfig{
		{Name: "expedite", WIPLimit: 1},
		{Name: "standard"},
	}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	// First create with expedite class should succeed.
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("class", "expedite")
	err = runCreate(cmd, []string{"First expedite"})

	_ = drainPipe(t, r, w)
	if err != nil {
		t.Fatalf("first create error: %v", err)
	}

	// Second create with expedite should fail (class WIP limit=1).
	r2, w2 := captureStdout(t)
	cmd2 := newCreateCmd()
	_ = cmd2.Flags().Set("class", "expedite")
	err = runCreate(cmd2, []string{"Second expedite"})

	_ = drainPipe(t, r2, w2)
	if err == nil {
		t.Fatal("expected class WIP limit error on second expedite task")
	}
}

func TestRunCreate_NoConfigDir(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newCreateCmd()
	err := runCreate(cmd, []string{"No board"})
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestRunInit_InvalidWIPLimit(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newInitCmd()
	_ = cmd.Flags().Set("wip-limit", "bad-format")

	err := runInit(cmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid WIP limit format")
	}
}

func TestRunInit_AlreadyInitialized_ErrorCode(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newInitCmd()
	err := runInit(cmd, nil)
	if err == nil {
		t.Fatal("expected error for already initialized board")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.BoardAlreadyExists {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.BoardAlreadyExists)
	}
}

func TestRunInit_DefaultDir(t *testing.T) {
	dir := t.TempDir()

	// Use empty flagDir so runInit falls back to config.DefaultDir relative to CWD.
	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	t.Chdir(dir)

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newInitCmd()
	_ = cmd.Flags().Set("name", "AutoDir")
	err := runInit(cmd, nil)

	_ = drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runInit error: %v", err)
	}

	// Verify the default directory was created.
	expectedDir := filepath.Join(dir, config.DefaultDir)
	if _, err := os.Stat(filepath.Join(expectedDir, config.ConfigFileName)); err != nil {
		t.Errorf("config should exist in default dir %s: %v", expectedDir, err)
	}
}
