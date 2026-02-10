package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// newContextCmd creates a cobra command with the flags runContext expects.
func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("write-to", "", "")
	cmd.Flags().StringSlice("sections", nil, "")
	cmd.Flags().Int("days", defaultContextDays, "")
	return cmd
}

// writeContextTask creates a task file for context tests.
func writeContextTask(t *testing.T, cfg *config.Config, tk *task.Task) {
	t.Helper()
	slug := task.GenerateSlug(tk.Title)
	filename := task.GenerateFilename(tk.ID, slug)
	path := filepath.Join(cfg.TasksPath(), filename)
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}
}

// setupContextBoard creates a board with sample tasks for context tests.
func setupContextBoard(t *testing.T) *config.Config {
	t.Helper()
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeContextTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "In progress task",
		Status:   "in-progress",
		Priority: "high",
		Created:  now,
		Updated:  now,
	})
	writeContextTask(t, cfg, &task.Task{
		ID:       2,
		Title:    "Done task",
		Status:   "done",
		Priority: "medium",
		Created:  now.Add(-24 * time.Hour),
		Updated:  now,
	})
	writeContextTask(t, cfg, &task.Task{
		ID:       3,
		Title:    "Backlog task",
		Status:   "backlog",
		Priority: "low",
		Created:  now,
		Updated:  now,
	})

	return cfg
}

// --- runContext tests ---

func TestRunContext_DefaultMarkdownOutput(t *testing.T) {
	cfg := setupContextBoard(t)

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newContextCmd()
	err := runContext(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runContext error: %v", err)
	}
	// Markdown output should contain section headings.
	if !containsSubstring(got, "#") {
		t.Errorf("expected markdown heading in output, got: %s", got)
	}
}

func TestRunContext_JSONOutput(t *testing.T) {
	cfg := setupContextBoard(t)

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newContextCmd()
	err := runContext(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runContext error: %v", err)
	}
	if !containsSubstring(got, "{") {
		t.Errorf("expected JSON object in output, got: %s", got)
	}
}

func TestRunContext_WriteToFile(t *testing.T) {
	cfg := setupContextBoard(t)

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	outFile := filepath.Join(t.TempDir(), "context.md")

	r, w := captureStdout(t)

	cmd := newContextCmd()
	_ = cmd.Flags().Set("write-to", outFile)

	err := runContext(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runContext --write-to error: %v", err)
	}
	if !containsSubstring(got, "Context written to") {
		t.Errorf("expected confirmation message, got: %s", got)
	}

	// Verify file was written.
	data, err := os.ReadFile(outFile) //nolint:gosec // test path
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty context file")
	}
}

func TestRunContext_WriteToExistingFile(t *testing.T) {
	cfg := setupContextBoard(t)

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	outFile := filepath.Join(t.TempDir(), "agents.md")
	// Pre-populate with existing content and sentinel markers.
	existing := "# My Project\n\nSome content.\n\n<!-- BEGIN kanban-md context -->\nold context\n<!-- END kanban-md context -->\n\nMore content.\n"
	if err := os.WriteFile(outFile, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	r, w := captureStdout(t)

	cmd := newContextCmd()
	_ = cmd.Flags().Set("write-to", outFile)

	err := runContext(cmd, nil)
	_ = drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runContext --write-to error: %v", err)
	}

	data, err := os.ReadFile(outFile) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Existing content outside markers should be preserved.
	if !containsSubstring(content, "# My Project") {
		t.Error("expected existing heading to be preserved")
	}
	if !containsSubstring(content, "More content.") {
		t.Error("expected trailing content to be preserved")
	}
	// Old context should be replaced.
	if containsSubstring(content, "old context") {
		t.Error("expected old context block to be replaced")
	}
}

func TestRunContext_SectionsFilter(t *testing.T) {
	cfg := setupContextBoard(t)

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newContextCmd()
	_ = cmd.Flags().Set("sections", "in-progress")

	err := runContext(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runContext --sections error: %v", err)
	}
	// Output should contain something (at least a heading for in-progress).
	if len(got) == 0 {
		t.Error("expected non-empty output with sections filter")
	}
}

func TestRunContext_ExcludesArchivedTasks(t *testing.T) {
	cfg := setupContextBoard(t)

	// Add an archived task.
	now := time.Now()
	writeContextTask(t, cfg, &task.Task{
		ID:       4,
		Title:    "Archived invisible",
		Status:   config.ArchivedStatus,
		Priority: "low",
		Created:  now,
		Updated:  now,
	})

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newContextCmd()
	err := runContext(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runContext error: %v", err)
	}
	if containsSubstring(got, "Archived invisible") {
		t.Error("expected archived task to be excluded from context")
	}
}

func TestRunContext_CustomDays(t *testing.T) {
	cfg := setupContextBoard(t)

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newContextCmd()
	_ = cmd.Flags().Set("days", "1")

	err := runContext(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runContext --days error: %v", err)
	}
	// Should produce valid JSON output.
	if !containsSubstring(got, "{") {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestRunContext_NoConfigFails(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newContextCmd()
	err := runContext(cmd, nil)
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestRunContext_WriteToInvalidPath(t *testing.T) {
	cfg := setupContextBoard(t)

	oldFlagDir := flagDir
	flagDir = cfg.Dir()
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newContextCmd()
	_ = cmd.Flags().Set("write-to", "/nonexistent/dir/file.md")

	err := runContext(cmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid write-to path")
	}
	if !containsSubstring(err.Error(), "writing context file") {
		t.Errorf("expected 'writing context file' error, got: %v", err)
	}
}
