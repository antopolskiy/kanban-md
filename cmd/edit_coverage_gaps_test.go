package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// --- applyEditChanges coverage ---

func TestApplyEditChanges_ApplyEditFlagsError(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("status", "nonexistent-status")
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1, Status: "backlog"}

	_, err := applyEditChanges(cmd, tk, cfg, "", false)
	if err == nil {
		t.Fatal("expected error from invalid status")
	}
}

func TestApplyEditChanges_ClaimFlagsError(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("claim", "agent-1")
	_ = cmd.Flags().Set("release", "true")
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1}

	_, err := applyEditChanges(cmd, tk, cfg, "agent-1", true)
	if err == nil {
		t.Fatal("expected error from --claim + --release conflict")
	}
}

func TestApplyEditChanges_ClaimOnlyChange(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("claim", "agent-1")
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1}

	changed, err := applyEditChanges(cmd, tk, cfg, "agent-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for claim-only edit")
	}
	if tk.ClaimedBy != "agent-1" {
		t.Errorf("ClaimedBy = %q, want %q", tk.ClaimedBy, "agent-1")
	}
}

// --- executeEdit coverage ---

func TestExecuteEdit_TaskReadError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a task file with invalid content.
	tasksDir := cfg.TasksPath()
	badPath := filepath.Join(tasksDir, "1-bad-task.md")
	writeErr := os.WriteFile(badPath, []byte("not valid frontmatter"), 0o600)
	if writeErr != nil {
		t.Fatal(writeErr)
	}

	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "New Title")

	_, _, err = executeEdit(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected error from malformed task file")
	}
}

func TestExecuteEdit_ValidateEditClaimError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a task claimed by someone else.
	createClaimedTaskFile(t, cfg.TasksPath(), 1, "claimed-task", "other-agent")

	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "New Title")

	_, _, err = executeEdit(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected claim error")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.TaskClaimed {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.TaskClaimed)
	}
}

func TestExecuteEdit_ApplyEditChangesError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "test-task")

	cmd := newEditCmd()
	_ = cmd.Flags().Set("status", "nonexistent")

	_, _, err = executeEdit(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected error from invalid status")
	}
}

func TestExecuteEdit_NoChanges(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "test-task")

	cmd := newEditCmd()
	// No flags set → no changes.

	_, _, err = executeEdit(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected 'no changes' error")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.NoChanges {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.NoChanges)
	}
}

func TestExecuteEdit_ValidateEditPostError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "test-task")

	// Set self-referencing dependency to trigger validateEditPost error.
	cmd := newEditCmd()
	_ = cmd.Flags().Set("add-dep", "1")

	_, _, err = executeEdit(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected validation error from self-reference")
	}
}

// --- validateEditClaim coverage ---

func TestValidateEditClaim_RequireClaimNoClaimant(t *testing.T) {
	cfg := config.NewDefault("Test")
	// Enable require_claim for statusInProgress.
	for i := range cfg.Statuses {
		if cfg.Statuses[i].Name == statusInProgress {
			cfg.Statuses[i].RequireClaim = true
		}
	}

	tk := &task.Task{ID: 1, Status: statusInProgress}
	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "New Title")

	_, _, err := validateEditClaim(cfg, tk, cmd)
	if err == nil {
		t.Fatal("expected require_claim error")
	}
}

// --- validateEditPost coverage ---

func TestValidateEditPost_RequireClaimOnStatusChangeViaCfg(t *testing.T) {
	cfg := config.NewDefault("Test")
	for i := range cfg.Statuses {
		if cfg.Statuses[i].Name == statusInProgress {
			cfg.Statuses[i].RequireClaim = true
		}
	}
	cfg.SetDir(t.TempDir())

	tk := &task.Task{ID: 1, Status: statusInProgress}

	err := validateEditPost(cfg, tk, "backlog", "")
	if err == nil {
		t.Fatal("expected require_claim error on status change")
	}
}

func TestValidateEditPost_ClassAwareWIPLimit(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Disable require_claim so we reach the class WIP check.
	for i := range cfg.Statuses {
		cfg.Statuses[i].RequireClaim = false
	}

	// Configure a class with WIP limit.
	cfg.Classes = []config.ClassConfig{
		{Name: "expedite", WIPLimit: 1},
	}

	// Create an existing expedite task in in-progress.
	createTaskFileWithClassAndStatus(t, cfg.TasksPath(), 2, "existing-expedite", statusInProgress, "expedite")

	tk := &task.Task{ID: 1, Status: statusInProgress, Class: "expedite"}

	err = validateEditPost(cfg, tk, "backlog", "")
	if err == nil {
		t.Fatal("expected class WIP limit error")
	}
	if !containsSubstring(err.Error(), "expedite") {
		t.Errorf("expected error mentioning class name, got: %v", err)
	}
}

// --- applyEditFlags coverage ---

func TestApplyEditFlags_SimpleEditFlagsError(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("priority", "invalid-priority")
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1}

	_, err := applyEditFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error from invalid priority")
	}
}

func TestApplyEditFlags_HelperFunctionError(t *testing.T) {
	cmd := newEditCmd()
	_ = cmd.Flags().Set("started", "not-a-date")
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1}

	_, err := applyEditFlags(cmd, tk, cfg)
	if err == nil {
		t.Fatal("expected error from invalid started date")
	}
}

// --- deleteSingleTask coverage ---

func TestDeleteSingleTask_TaskReadError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a malformed task file.
	badPath := filepath.Join(cfg.TasksPath(), "1-bad-task.md")
	writeErr := os.WriteFile(badPath, []byte("not valid frontmatter"), 0o600)
	if writeErr != nil {
		t.Fatal(writeErr)
	}

	err = deleteSingleTask(cfg, 1, true)
	if err == nil {
		t.Fatal("expected error from malformed task file")
	}
}

func TestDeleteSingleTask_NonTTYWithoutYes(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "test-task")

	// In tests, stdin is a pipe (not a TTY), so this should return
	// ConfirmationReq without --yes.
	err = deleteSingleTask(cfg, 1, false)
	if err == nil {
		t.Fatal("expected confirmation error in non-TTY mode")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.ConfirmationReq {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.ConfirmationReq)
	}
}

func TestDeleteSingleTask_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "json-delete-task")

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err = deleteSingleTask(cfg, 1, true)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsSubstring(got, `"status": "deleted"`) {
		t.Errorf("expected JSON with status:deleted, got: %s", got)
	}
	if !containsSubstring(got, `"id": 1`) {
		t.Errorf("expected JSON with id:1, got: %s", got)
	}
}

// --- executeDelete coverage ---

func TestExecuteDelete_TaskReadError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(cfg.TasksPath(), "1-bad-task.md")
	writeErr := os.WriteFile(badPath, []byte("not valid frontmatter"), 0o600)
	if writeErr != nil {
		t.Fatal(writeErr)
	}

	err = executeDelete(cfg, 1)
	if err == nil {
		t.Fatal("expected error from malformed task file")
	}
}

// --- softDeleteAndLog coverage ---

func TestSoftDeleteAndLog_WriteError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	createTaskFile(t, cfg.TasksPath(), 1, "write-fail-task")
	path, findErr := task.FindByID(cfg.TasksPath(), 1)
	if findErr != nil {
		t.Fatal(findErr)
	}
	tk, readErr := task.Read(path)
	if readErr != nil {
		t.Fatal(readErr)
	}

	// Make the task file read-only to trigger a write error.
	chmodErr := os.Chmod(path, 0o400)
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })

	err = softDeleteAndLog(cfg, path, tk)
	if err == nil {
		t.Fatal("expected write error")
	}
}

// --- writeAndRename error path in executeEdit ---

func TestExecuteEdit_WriteAndRenameError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "test-task")

	if runtime.GOOS == "windows" { //nolint:goconst // idiomatic platform check
		t.Skip("chmod does not restrict directory writes on Windows")
	}

	// Make the tasks directory read-only so creating the renamed file fails.
	tasksDir := cfg.TasksPath()
	chmodErr := os.Chmod(tasksDir, 0o500) //nolint:gosec // intentionally restricting dir for test
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	t.Cleanup(func() { _ = os.Chmod(tasksDir, 0o750) }) //nolint:gosec // restoring dir perms

	cmd := newEditCmd()
	_ = cmd.Flags().Set("title", "New Title")

	_, _, err = executeEdit(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected write error")
	}
}

// --- deleteSingleTask softDeleteAndLog error ---

func TestDeleteSingleTask_SoftDeleteWriteError(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "fail-delete-task")

	if runtime.GOOS == "windows" {
		t.Skip("chmod does not restrict file writes on Windows")
	}

	// Make the task file read-only to trigger a write error.
	path, findErr := task.FindByID(cfg.TasksPath(), 1)
	if findErr != nil {
		t.Fatal(findErr)
	}
	chmodErr := os.Chmod(path, 0o400)
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })

	err = deleteSingleTask(cfg, 1, true)
	if err == nil {
		t.Fatal("expected write error from softDeleteAndLog")
	}
}

// --- helpers ---

func createClaimedTaskFile(t *testing.T, tasksDir string, id int, title, claimant string) {
	t.Helper()
	slug := task.GenerateSlug(title)
	filename := task.GenerateFilename(id, slug)
	now := time.Now().UTC().Format(time.RFC3339)
	content := "---\nid: " + idStr(id) + "\ntitle: " + title +
		"\nstatus: backlog\npriority: medium" +
		"\ncreated: 2025-01-01T00:00:00Z\nupdated: 2025-01-01T00:00:00Z" +
		"\nclaimed_by: " + claimant +
		"\nclaimed_at: " + now +
		"\n---\n"
	path := filepath.Join(tasksDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func createTaskFileWithClassAndStatus(t *testing.T, tasksDir string, id int, title, status, class string) {
	t.Helper()
	slug := task.GenerateSlug(title)
	filename := task.GenerateFilename(id, slug)
	content := "---\nid: " + idStr(id) + "\ntitle: " + title +
		"\nstatus: " + status + "\npriority: medium" +
		"\nclass: " + class +
		"\ncreated: 2025-01-01T00:00:00Z\nupdated: 2025-01-01T00:00:00Z\n---\n"
	path := filepath.Join(tasksDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
