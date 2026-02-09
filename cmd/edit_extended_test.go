package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// newEditCmd creates a fresh cobra command with edit flags for testing.
func newEditCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("title", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("priority", "", "")
	cmd.Flags().String("assignee", "", "")
	cmd.Flags().StringSlice("add-tag", nil, "")
	cmd.Flags().StringSlice("remove-tag", nil, "")
	cmd.Flags().String("due", "", "")
	cmd.Flags().Bool("clear-due", false, "")
	cmd.Flags().String("estimate", "", "")
	cmd.Flags().String("body", "", "")
	cmd.Flags().String("started", "", "")
	cmd.Flags().Bool("clear-started", false, "")
	cmd.Flags().String("completed", "", "")
	cmd.Flags().Bool("clear-completed", false, "")
	cmd.Flags().Int("parent", 0, "")
	cmd.Flags().Bool("clear-parent", false, "")
	cmd.Flags().IntSlice("add-dep", nil, "")
	cmd.Flags().IntSlice("remove-dep", nil, "")
	cmd.Flags().String("block", "", "")
	cmd.Flags().Bool("unblock", false, "")
	cmd.Flags().String("claim", "", "")
	cmd.Flags().Bool("release", false, "")
	cmd.Flags().String("class", "", "")
	cmd.Flags().BoolP("force", "f", false, "")
	return cmd
}

// --- applySimpleEditFlags tests ---

func TestApplySimpleEditFlags_Title(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "New Title")
	cfg := config.NewDefault("Test")
	tk := &task.Task{Title: "Old Title"}

	changed, err := applySimpleEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Title != "New Title" {
		t.Errorf("title = %q, want %q", tk.Title, "New Title")
	}
}

func TestApplySimpleEditFlags_Status(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("status", "done")
	cfg := config.NewDefault("Test")
	tk := &task.Task{Status: "backlog"}

	changed, err := applySimpleEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Status != "done" {
		t.Errorf("status = %q, want %q", tk.Status, "done")
	}
}

func TestApplySimpleEditFlags_InvalidStatus(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("status", "invalid")
	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	_, err := applySimpleEditFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestApplySimpleEditFlags_Priority(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("priority", "critical")
	cfg := config.NewDefault("Test")
	tk := &task.Task{Priority: "low"}

	changed, err := applySimpleEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Priority != "critical" {
		t.Errorf("priority = %q, want %q", tk.Priority, "critical")
	}
}

func TestApplySimpleEditFlags_Assignee(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("assignee", "bob")
	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	changed, err := applySimpleEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Assignee != "bob" {
		t.Errorf("assignee = %q, want %q", tk.Assignee, "bob")
	}
}

func TestApplySimpleEditFlags_Estimate(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("estimate", "2d")
	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	changed, err := applySimpleEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Estimate != "2d" {
		t.Errorf("estimate = %q, want %q", tk.Estimate, "2d")
	}
}

func TestApplySimpleEditFlags_Body(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("body", "new body text")
	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	changed, err := applySimpleEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Body != "new body text" {
		t.Errorf("body = %q, want %q", tk.Body, "new body text")
	}
}

func TestApplySimpleEditFlags_Class(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("class", "expedite")
	cfg := config.NewDefault("Test")
	tk := &task.Task{}

	changed, err := applySimpleEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Class != "expedite" {
		t.Errorf("class = %q, want %q", tk.Class, "expedite")
	}
}

func TestApplySimpleEditFlags_NoFlags(t *testing.T) {
	cmd := newEditCmd()
	cfg := config.NewDefault("Test")
	tk := &task.Task{Title: "Original"}

	changed, err := applySimpleEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false when no flags set")
	}
	if tk.Title != "Original" {
		t.Errorf("title should not change, got %q", tk.Title)
	}
}

// --- applyTimestampFlags tests ---

func TestApplyTimestampFlags_Started(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("started", "2025-03-15")
	tk := &task.Task{}

	changed, err := applyTimestampFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Started == nil {
		t.Fatal("started should be set")
	}
	if tk.Started.Day() != 15 {
		t.Errorf("started day = %d, want 15", tk.Started.Day())
	}
}

func TestApplyTimestampFlags_ClearStarted(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("clear-started", "true")
	now := time.Now()
	tk := &task.Task{Started: &now}

	changed, err := applyTimestampFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Started != nil {
		t.Error("started should be nil after clear")
	}
}

func TestApplyTimestampFlags_StartedConflict(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("started", "2025-03-15")
	_ = cmd.Flags().Set("clear-started", "true")
	tk := &task.Task{}

	_, err := applyTimestampFlags(cmd, tk)
	if err == nil {
		t.Fatal("expected error for conflicting flags")
	}
}

func TestApplyTimestampFlags_Completed(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("completed", "2025-04-01")
	tk := &task.Task{}

	changed, err := applyTimestampFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Completed == nil {
		t.Fatal("completed should be set")
	}
}

func TestApplyTimestampFlags_ClearCompleted(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("clear-completed", "true")
	now := time.Now()
	tk := &task.Task{Completed: &now}

	changed, err := applyTimestampFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Completed != nil {
		t.Error("completed should be nil after clear")
	}
}

func TestApplyTimestampFlags_CompletedConflict(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("completed", "2025-04-01")
	_ = cmd.Flags().Set("clear-completed", "true")
	tk := &task.Task{}

	_, err := applyTimestampFlags(cmd, tk)
	if err == nil {
		t.Fatal("expected error for conflicting flags")
	}
}

func TestApplyTimestampFlags_InvalidDate(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("started", "not-a-date")
	tk := &task.Task{}

	_, err := applyTimestampFlags(cmd, tk)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

// --- applyTagDueFlags tests ---

func TestApplyTagDueFlags_AddTag(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("add-tag", "new-tag")
	tk := &task.Task{Tags: []string{"existing"}}

	changed, err := applyTagDueFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if len(tk.Tags) != 2 || tk.Tags[1] != "new-tag" {
		t.Errorf("tags = %v, want [existing, new-tag]", tk.Tags)
	}
}

func TestApplyTagDueFlags_RemoveTag(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("remove-tag", "old")
	tk := &task.Task{Tags: []string{"old", "keep"}}

	changed, err := applyTagDueFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if len(tk.Tags) != 1 || tk.Tags[0] != "keep" {
		t.Errorf("tags = %v, want [keep]", tk.Tags)
	}
}

func TestApplyTagDueFlags_Due(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("due", "2025-12-25")
	tk := &task.Task{}

	changed, err := applyTagDueFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Due == nil {
		t.Fatal("due should be set")
	}
}

func TestApplyTagDueFlags_ClearDue(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("clear-due", "true")
	tk := &task.Task{}

	changed, err := applyTagDueFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Due != nil {
		t.Error("due should be nil after clear")
	}
}

func TestApplyTagDueFlags_InvalidDue(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("due", "bad-date")
	tk := &task.Task{}

	_, err := applyTagDueFlags(cmd, tk)
	if err == nil {
		t.Fatal("expected error for invalid due date")
	}
}

// --- applyDepFlags tests ---

func TestApplyDepFlags_SetParent(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("parent", "5")
	tk := &task.Task{}

	changed, err := applyDepFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Parent == nil || *tk.Parent != 5 {
		t.Errorf("parent = %v, want 5", tk.Parent)
	}
}

func TestApplyDepFlags_ClearParent(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("clear-parent", "true")
	parentID := 5
	tk := &task.Task{Parent: &parentID}

	changed, err := applyDepFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Parent != nil {
		t.Error("parent should be nil after clear")
	}
}

func TestApplyDepFlags_ParentConflict(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("parent", "5")
	_ = cmd.Flags().Set("clear-parent", "true")
	tk := &task.Task{}

	_, err := applyDepFlags(cmd, tk)
	if err == nil {
		t.Fatal("expected error for conflicting parent flags")
	}
}

func TestApplyDepFlags_AddDep(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("add-dep", "3,4")
	tk := &task.Task{DependsOn: []int{1}}

	changed, err := applyDepFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if len(tk.DependsOn) != 3 {
		t.Errorf("depends_on len = %d, want 3", len(tk.DependsOn))
	}
}

func TestApplyDepFlags_RemoveDep(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("remove-dep", "2")
	tk := &task.Task{DependsOn: []int{1, 2, 3}}

	changed, err := applyDepFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if len(tk.DependsOn) != 2 {
		t.Errorf("depends_on len = %d, want 2", len(tk.DependsOn))
	}
}

// --- applyBlockFlags tests ---

func TestApplyBlockFlags_Block(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("block", "waiting on review")
	tk := &task.Task{}

	changed, err := applyBlockFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if !tk.Blocked {
		t.Error("expected blocked=true")
	}
	if tk.BlockReason != "waiting on review" {
		t.Errorf("block_reason = %q, want %q", tk.BlockReason, "waiting on review")
	}
}

func TestApplyBlockFlags_Unblock(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("unblock", "true")
	tk := &task.Task{Blocked: true, BlockReason: "old reason"}

	changed, err := applyBlockFlags(cmd, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Blocked {
		t.Error("expected blocked=false")
	}
	if tk.BlockReason != "" {
		t.Errorf("block_reason should be empty, got %q", tk.BlockReason)
	}
}

func TestApplyBlockFlags_BlockUnblockConflict(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("block", "reason")
	_ = cmd.Flags().Set("unblock", "true")
	tk := &task.Task{}

	_, err := applyBlockFlags(cmd, tk)
	if err == nil {
		t.Fatal("expected error for conflicting block flags")
	}
}

func TestApplyBlockFlags_EmptyBlockReason(t *testing.T) {
	cmd := newEditCmd()
	// Set the flag with empty string — this triggers Changed("block")=true.
	_ = cmd.Flags().Set("block", "")
	tk := &task.Task{}

	_, err := applyBlockFlags(cmd, tk)
	if err == nil {
		t.Fatal("expected error for empty block reason")
	}
}

// --- applyClaimFlags tests ---

func TestApplyClaimFlags_Claim(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("claim", "agent-1")
	tk := &task.Task{ID: 1}

	changed, err := applyClaimFlags(cmd, tk, "agent-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.ClaimedBy != "agent-1" {
		t.Errorf("claimed_by = %q, want %q", tk.ClaimedBy, "agent-1")
	}
	if tk.ClaimedAt == nil {
		t.Error("claimed_at should be set")
	}
}

func TestApplyClaimFlags_Release(t *testing.T) {
	cmd := newEditCmd()
	now := time.Now()
	tk := &task.Task{ID: 1, ClaimedBy: "agent-1", ClaimedAt: &now}

	changed, err := applyClaimFlags(cmd, tk, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.ClaimedBy != "" {
		t.Errorf("claimed_by should be empty, got %q", tk.ClaimedBy)
	}
	if tk.ClaimedAt != nil {
		t.Error("claimed_at should be nil after release")
	}
}

func TestApplyClaimFlags_ClaimAndReleaseConflict(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("claim", "agent-1")
	tk := &task.Task{}

	_, err := applyClaimFlags(cmd, tk, "agent-1", true)
	if err == nil {
		t.Fatal("expected error for claim+release conflict")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.StatusConflict {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.StatusConflict)
	}
}

func TestApplyClaimFlags_EmptyClaimName(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("claim", "")
	tk := &task.Task{}

	_, err := applyClaimFlags(cmd, tk, "", false)
	if err == nil {
		t.Fatal("expected error for empty claim name")
	}
}

func TestApplyClaimFlags_NoClaimFlags(t *testing.T) {
	cmd := newEditCmd()
	tk := &task.Task{}

	changed, err := applyClaimFlags(cmd, tk, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false when no claim flags set")
	}
}

// --- writeAndRename tests ---

func TestWriteAndRename_SameTitle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001-test-task.md")
	tk := &task.Task{ID: 1, Title: "test task", Status: "backlog", Priority: "medium"}

	// Write initial file.
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}

	newPath, err := writeAndRename(path, tk, "test task")
	if err != nil {
		t.Fatal(err)
	}
	if newPath != path {
		t.Errorf("path should not change, got %q", newPath)
	}
}

func TestWriteAndRename_TitleChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001-old-title.md")
	tk := &task.Task{ID: 1, Title: "new title", Status: "backlog", Priority: "medium"}

	// Write initial file.
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}

	newPath, err := writeAndRename(path, tk, "old title")
	if err != nil {
		t.Fatal(err)
	}
	if newPath == path {
		t.Error("path should change when title changes")
	}

	// Old file should be removed.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("old file should be removed")
	}
	// New file should exist.
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new file should exist: %v", err)
	}
}

// --- logEditActivity tests ---

func TestLogEditActivity_BasicEdit(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test"}
	logEditActivity(cfg, tk, false, "")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "edit") {
		t.Errorf("expected 'edit' action in log, got: %s", got)
	}
}

func TestLogEditActivity_BlockTransition(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", Blocked: true, BlockReason: "dependency"}
	logEditActivity(cfg, tk, false, "")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "block") {
		t.Errorf("expected 'block' action in log, got: %s", got)
	}
}

func TestLogEditActivity_UnblockTransition(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", Blocked: false}
	logEditActivity(cfg, tk, true, "")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "unblock") {
		t.Errorf("expected 'unblock' action in log, got: %s", got)
	}
}

func TestLogEditActivity_ClaimTransition(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", ClaimedBy: "agent-1"}
	logEditActivity(cfg, tk, false, "")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "claim") {
		t.Errorf("expected 'claim' action in log, got: %s", got)
	}
}

func TestLogEditActivity_ReleaseTransition(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", ClaimedBy: ""}
	logEditActivity(cfg, tk, false, "agent-1")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "release") {
		t.Errorf("expected 'release' action in log, got: %s", got)
	}
}

// --- applyEditFlags integration tests ---

func TestApplyEditFlags_MultipleFlags(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "Updated")
	_ = cmd.Flags().Set("priority", "high")
	_ = cmd.Flags().Set("add-tag", "urgent")

	cfg := config.NewDefault("Test")
	tk := &task.Task{Title: "Old", Priority: "low"}

	changed, err := applyEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if tk.Title != "Updated" {
		t.Errorf("title = %q, want %q", tk.Title, "Updated")
	}
	if tk.Priority != "high" {
		t.Errorf("priority = %q, want %q", tk.Priority, "high")
	}
	if len(tk.Tags) != 1 || tk.Tags[0] != "urgent" {
		t.Errorf("tags = %v, want [urgent]", tk.Tags)
	}
}

func TestApplyEditFlags_NoFlags(t *testing.T) {
	cmd := newEditCmd()
	cfg := config.NewDefault("Test")
	tk := &task.Task{Title: "Original"}

	changed, err := applyEditFlags(cmd, tk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false when no flags set")
	}
}
