package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// newCreateCmd creates a fresh cobra command with create flags for testing.
func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("title", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("priority", "", "")
	cmd.Flags().String("assignee", "", "")
	cmd.Flags().StringSlice("tags", nil, "")
	cmd.Flags().String("due", "", "")
	cmd.Flags().String("estimate", "", "")
	cmd.Flags().Int("parent", 0, "")
	cmd.Flags().IntSlice("depends-on", nil, "")
	cmd.Flags().String("body", "", "")
	cmd.Flags().String("class", "", "")
	cmd.Flags().String("claim", "", "")
	return cmd
}

func TestApplyCreateFlags_Status(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("status", "in-progress")

	cfg := config.NewDefault("Test")
	tk := &task.Task{Status: "backlog"}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Status != "in-progress" {
		t.Errorf("status = %q, want %q", tk.Status, "in-progress")
	}
}

func TestApplyCreateFlags_InvalidStatus(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("status", "nonexistent")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	err := applyCreateFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestApplyCreateFlags_Priority(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("priority", priorityHigh)

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Priority != priorityHigh {
		t.Errorf("priority = %q, want %q", tk.Priority, priorityHigh)
	}
}

func TestApplyCreateFlags_InvalidPriority(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("priority", "ultra")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	err := applyCreateFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestApplyCreateFlags_Assignee(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("assignee", "alice")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Assignee != "alice" {
		t.Errorf("assignee = %q, want %q", tk.Assignee, "alice")
	}
}

func TestApplyCreateFlags_Tags(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("tags", "bug,urgent")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if len(tk.Tags) != 2 || tk.Tags[0] != "bug" || tk.Tags[1] != "urgent" {
		t.Errorf("tags = %v, want [bug, urgent]", tk.Tags)
	}
}

func TestApplyCreateFlags_Due(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("due", "2025-06-15")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Due == nil {
		t.Fatal("due should be set")
	}
	if tk.Due.Year() != 2025 || tk.Due.Month() != 6 || tk.Due.Day() != 15 {
		t.Errorf("due = %v, want 2025-06-15", tk.Due)
	}
}

func TestApplyCreateFlags_InvalidDue(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("due", "not-a-date")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	err := applyCreateFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error for invalid due date")
	}
}

func TestApplyCreateFlags_Estimate(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("estimate", "4h")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Estimate != "4h" {
		t.Errorf("estimate = %q, want %q", tk.Estimate, "4h")
	}
}

func TestApplyCreateFlags_Parent(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("parent", "5")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Parent == nil || *tk.Parent != 5 {
		t.Errorf("parent = %v, want 5", tk.Parent)
	}
}

func TestApplyCreateFlags_DependsOn(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("depends-on", "2,3")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if len(tk.DependsOn) != 2 || tk.DependsOn[0] != 2 || tk.DependsOn[1] != 3 {
		t.Errorf("depends_on = %v, want [2, 3]", tk.DependsOn)
	}
}

func TestApplyCreateFlags_Body(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("body", "task description")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Body != "task description" {
		t.Errorf("body = %q, want %q", tk.Body, "task description")
	}
}

func TestApplyCreateFlags_Class(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("class", "expedite")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Class != "expedite" {
		t.Errorf("class = %q, want %q", tk.Class, "expedite")
	}
}

func TestApplyCreateFlags_InvalidClass(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("class", "invalid-class")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	err := applyCreateFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error for invalid class")
	}
}

func TestApplyCreateFlags_Claim(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("claim", "agent-test")

	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.ClaimedBy != "agent-test" {
		t.Errorf("claimed_by = %q, want %q", tk.ClaimedBy, "agent-test")
	}
	if tk.ClaimedAt == nil {
		t.Error("claimed_at should be set")
	}
}

func TestApplyCreateFlags_NoFlags(t *testing.T) {
	cmd := newCreateCmd()
	cfg := config.NewDefault("Test")
	tk := &task.Task{Status: "backlog", Priority: "medium"}

	if err := applyCreateFlags(cmd, tk, cfg); err != nil {
		t.Fatal(err)
	}
	if tk.Status != "backlog" {
		t.Errorf("status should not change, got %q", tk.Status)
	}
	if tk.Priority != "medium" {
		t.Errorf("priority should not change, got %q", tk.Priority)
	}
}

func TestRunCreate_Integration(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newCreateCmd()
	_ = cmd.Flags().Set("priority", priorityHigh)
	_ = cmd.Flags().Set("tags", "test")

	err := runCreate(cmd, []string{"Test task title"})

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runCreate error: %v", err)
	}
	if !containsSubstring(got, "Created task #1") {
		t.Errorf("expected 'Created task #1' in output, got: %s", got)
	}
	if !containsSubstring(got, "Test task title") {
		t.Errorf("expected title in output, got: %s", got)
	}

	// Verify the file was created.
	entries, err := os.ReadDir(filepath.Join(kanbanDir, "tasks"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 task file, got %d", len(entries))
	}

	// Verify config was updated.
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NextID != 2 {
		t.Errorf("NextID = %d, want 2", cfg.NextID)
	}
}

func TestRunCreate_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newCreateCmd()
	err := runCreate(cmd, []string{"JSON test"})

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runCreate error: %v", err)
	}
	if !containsSubstring(got, `"title": "JSON test"`) {
		t.Errorf("expected JSON output with title, got: %s", got)
	}
}

func TestResolveCreateTitle_Positional(t *testing.T) {
	cmd := newCreateCmd()
	title, err := resolveCreateTitle(cmd, []string{"My task"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "My task" {
		t.Errorf("title = %q, want %q", title, "My task")
	}
}

func TestResolveCreateTitle_Flag(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("title", "Flag task")
	title, err := resolveCreateTitle(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Flag task" {
		t.Errorf("title = %q, want %q", title, "Flag task")
	}
}

func TestResolveCreateTitle_BothError(t *testing.T) {
	cmd := newCreateCmd()
	_ = cmd.Flags().Set("title", "Flag task")
	_, err := resolveCreateTitle(cmd, []string{"Positional task"})
	if err == nil {
		t.Fatal("expected error when both positional and --title provided")
	}
}

func TestResolveCreateTitle_NeitherError(t *testing.T) {
	cmd := newCreateCmd()
	_, err := resolveCreateTitle(cmd, nil)
	if err == nil {
		t.Fatal("expected error when no title provided")
	}
}
