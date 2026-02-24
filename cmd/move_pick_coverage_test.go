package cmd

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const testTodoStatus = "todo"

// ===== move.go coverage =====

// --- executeMove error paths ---

func TestExecuteMove_TaskNotFound(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	cmd := newMoveCmd()
	_, _, err = executeMove(cfg, 999, cmd, []string{"999", "todo"})
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

func TestExecuteMove_Idempotent(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "already-todo", testTodoStatus)

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	tk, oldStatus, moveErr := executeMove(cfg, 1, cmd, []string{"1", testTodoStatus})

	_ = drainPipe(t, r, w)

	if moveErr != nil {
		t.Fatalf("executeMove error: %v", moveErr)
	}
	if oldStatus != "" {
		t.Errorf("oldStatus = %q, want empty for idempotent move", oldStatus)
	}
	if tk.Status != testTodoStatus {
		t.Errorf("status = %q, want %q", tk.Status, testTodoStatus)
	}
}

func TestExecuteMove_RequireClaimForTarget(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "backlog-task", "backlog")

	cmd := newMoveCmd()
	// Move to in-progress (which requires claim by default) without --claim.
	_, _, err = executeMove(cfg, 1, cmd, []string{"1", "in-progress"})
	if err == nil {
		t.Fatal("expected error when moving to status that requires claim")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.ClaimRequired {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.ClaimRequired)
	}
}

func TestExecuteMove_BlockedTaskWarning(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a blocked task.
	tk := &task.Task{
		ID: 1, Title: "blocked-task", Status: "backlog", Priority: "medium",
		Blocked: true, BlockReason: "waiting on deps",
		Created: time.Now(), Updated: time.Now(),
	}
	slug := task.GenerateSlug(tk.Title)
	filename := task.GenerateFilename(tk.ID, slug)
	path := filepath.Join(cfg.TasksPath(), filename)
	if wErr := task.Write(path, tk); wErr != nil {
		t.Fatal(wErr)
	}

	rErr, wErr := captureStderr(t)

	cmd := newMoveCmd()
	_, _, moveErr := executeMove(cfg, 1, cmd, []string{"1", "todo"})

	stderr := drainPipe(t, rErr, wErr)

	if moveErr != nil {
		t.Fatalf("executeMove error: %v", moveErr)
	}
	if !containsSubstring(stderr, "blocked") {
		t.Errorf("expected blocked warning in stderr, got: %s", stderr)
	}
}

func TestExecuteMove_ApplyClaimFlag(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "claim-me", "backlog")

	cmd := newMoveCmd()
	_ = cmd.Flags().Set("claim", "test-agent")
	tk, _, moveErr := executeMove(cfg, 1, cmd, []string{"1", "todo"})
	if moveErr != nil {
		t.Fatalf("executeMove error: %v", moveErr)
	}
	if tk.ClaimedBy != "test-agent" {
		t.Errorf("claimed_by = %q, want %q", tk.ClaimedBy, "test-agent")
	}
	if tk.ClaimedAt == nil {
		t.Error("claimed_at should be set")
	}
}

// --- moveSingleTask output paths ---

func TestMoveSingleTask_IdempotentTable(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "idempotent", "todo")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	moveErr := moveSingleTask(cfg, 1, cmd, []string{"1", "todo"})

	got := drainPipe(t, r, w)

	if moveErr != nil {
		t.Fatalf("moveSingleTask error: %v", moveErr)
	}
	if !containsSubstring(got, "already at") {
		t.Errorf("expected 'already at' message, got: %s", got)
	}
}

func TestMoveSingleTask_IdempotentJSON(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "idempotent-json", "todo")

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	moveErr := moveSingleTask(cfg, 1, cmd, []string{"1", "todo"})

	got := drainPipe(t, r, w)

	if moveErr != nil {
		t.Fatalf("moveSingleTask error: %v", moveErr)
	}
	if !containsSubstring(got, `"changed": false`) {
		t.Errorf("expected changed:false in JSON, got: %s", got)
	}
}

func TestMoveSingleTask_SuccessJSON(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "move-json", "backlog")

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	moveErr := moveSingleTask(cfg, 1, cmd, []string{"1", "todo"})

	got := drainPipe(t, r, w)

	if moveErr != nil {
		t.Fatalf("moveSingleTask error: %v", moveErr)
	}
	if !containsSubstring(got, `"changed": true`) {
		t.Errorf("expected changed:true in JSON, got: %s", got)
	}
}

func TestMoveSingleTask_SuccessTable(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "move-table", "backlog")

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newMoveCmd()
	moveErr := moveSingleTask(cfg, 1, cmd, []string{"1", "todo"})

	got := drainPipe(t, r, w)

	if moveErr != nil {
		t.Fatalf("moveSingleTask error: %v", moveErr)
	}
	if !containsSubstring(got, "Moved task #1") {
		t.Errorf("expected 'Moved task #1' message, got: %s", got)
	}
}

// --- resolveTargetStatus ---

func TestResolveTargetStatus_InvalidStatusArg(t *testing.T) {
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1, Status: "backlog"}

	cmd := newMoveCmd()
	_, err := resolveTargetStatus(cmd, []string{"1", "nonexistent"}, tk, cfg)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestResolveTargetStatus_NextSuccess(t *testing.T) {
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1, Status: "backlog"}

	cmd := newMoveCmd()
	_ = cmd.Flags().Set("next", "true")
	got, err := resolveTargetStatus(cmd, []string{"1"}, tk, cfg)
	if err != nil {
		t.Fatalf("resolveTargetStatus error: %v", err)
	}
	if got != testTodoStatus {
		t.Errorf("status = %q, want %q", got, testTodoStatus)
	}
}

func TestResolveTargetStatus_NextAtLast(t *testing.T) {
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1, Status: config.ArchivedStatus}

	cmd := newMoveCmd()
	_ = cmd.Flags().Set("next", "true")
	_, err := resolveTargetStatus(cmd, []string{"1"}, tk, cfg)
	if err == nil {
		t.Fatal("expected boundary error at last status")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.BoundaryError {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.BoundaryError)
	}
}

func TestResolveTargetStatus_PrevSuccess(t *testing.T) {
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1, Status: testTodoStatus}

	cmd := newMoveCmd()
	_ = cmd.Flags().Set("prev", "true")
	got, err := resolveTargetStatus(cmd, []string{"1"}, tk, cfg)
	if err != nil {
		t.Fatalf("resolveTargetStatus error: %v", err)
	}
	if got != testBacklogStatus {
		t.Errorf("status = %q, want %q", got, testBacklogStatus)
	}
}

func TestResolveTargetStatus_PrevAtFirst(t *testing.T) {
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1, Status: "backlog"}

	cmd := newMoveCmd()
	_ = cmd.Flags().Set("prev", "true")
	_, err := resolveTargetStatus(cmd, []string{"1"}, tk, cfg)
	if err == nil {
		t.Fatal("expected boundary error at first status")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.BoundaryError {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.BoundaryError)
	}
}

func TestResolveTargetStatus_NoFlagOrArg(t *testing.T) {
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1, Status: "backlog"}

	cmd := newMoveCmd()
	_, err := resolveTargetStatus(cmd, []string{"1"}, tk, cfg)
	if err == nil {
		t.Fatal("expected error when no target specified")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidInput {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidInput)
	}
}

func TestResolveTargetStatus_NextUnknownStatus(t *testing.T) {
	cfg := config.NewDefault("Test")
	tk := &task.Task{ID: 1, Status: "nonexistent"}

	cmd := newMoveCmd()
	_ = cmd.Flags().Set("next", "true")
	_, err := resolveTargetStatus(cmd, []string{"1"}, tk, cfg)
	if err == nil {
		t.Fatal("expected boundary error for unknown status")
	}
}

// --- enforceWIPLimit ---

func TestEnforceWIPLimit_NoLimit(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	err = enforceWIPLimit(cfg, "backlog", "todo")
	if err != nil {
		t.Errorf("expected nil with no WIP limit, got: %v", err)
	}
}

func TestEnforceWIPLimit_Exceeded(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.WIPLimits = map[string]int{"todo": 1}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "in-todo", "todo")

	cfg, err = config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	err = enforceWIPLimit(cfg, "backlog", "todo")
	if err == nil {
		t.Fatal("expected WIP limit exceeded error")
	}
}

// --- enforceWIPLimitForClass ---

func TestEnforceWIPLimitForClass_NilClass(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Task with unknown class — ClassByName returns nil.
	tk := &task.Task{ID: 1, Class: "nonexistent", Status: "backlog"}
	err = enforceWIPLimitForClass(cfg, tk, "backlog", "todo")
	if err != nil {
		t.Errorf("expected nil when class not configured, got: %v", err)
	}
}

func TestEnforceWIPLimitForClass_BypassColumnWIP(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	// Set a tight column WIP limit that would normally fail.
	cfg.WIPLimits = map[string]int{"todo": 1}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 2, "existing-todo", "todo")

	cfg, err = config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Expedite class bypasses column WIP by default.
	tk := &task.Task{ID: 1, Class: "expedite", Status: "backlog"}
	err = enforceWIPLimitForClass(cfg, tk, "backlog", "todo")
	if err != nil {
		t.Errorf("expected nil when class bypasses column WIP, got: %v", err)
	}
}

func TestEnforceWIPLimitForClass_ClassWIPExceeded(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create an expedite task (default WIP limit = 1).
	writeTaskWithClass(t, cfg.TasksPath(), 2, "other-expedite", "todo", "expedite")

	tk := &task.Task{ID: 3, Class: "expedite", Status: "backlog"}
	err = enforceWIPLimitForClass(cfg, tk, "backlog", "todo")
	if err == nil {
		t.Fatal("expected class WIP exceeded error")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.ClassWIPExceeded {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.ClassWIPExceeded)
	}
}

// --- countByClass ---

func TestCountByClass_Counts(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Class: "expedite"},
		{ID: 2, Class: "expedite"},
		{ID: 3, Class: "standard"},
	}
	// Exclude ID 1 — should only count ID 2.
	got := countByClass(tasks, "expedite", 1)
	if got != 1 {
		t.Errorf("countByClass = %d, want 1", got)
	}
}

func TestCountByClass_NoneMatch(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Class: "standard"},
	}
	got := countByClass(tasks, "expedite", 0)
	if got != 0 {
		t.Errorf("countByClass = %d, want 0", got)
	}
}

// --- outputMoveResult ---

func TestOutputMoveResult_UnchangedTable(t *testing.T) {
	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	tk := &task.Task{ID: 1, Status: "todo"}
	err := outputMoveResult(tk, false)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("outputMoveResult error: %v", err)
	}
	if !containsSubstring(got, "already at") {
		t.Errorf("expected 'already at' message, got: %s", got)
	}
}

func TestOutputMoveResult_ChangedJSON(t *testing.T) {
	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	tk := &task.Task{ID: 1, Status: "todo"}
	err := outputMoveResult(tk, true)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("outputMoveResult error: %v", err)
	}
	if !containsSubstring(got, `"changed": true`) {
		t.Errorf("expected changed:true in JSON, got: %s", got)
	}
}

// --- runMove ---

func TestRunMove_NoConfig(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newMoveCmd()
	err := runMove(cmd, []string{"1", "todo"})
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestRunMove_InvalidID(t *testing.T) {
	cmd := newMoveCmd()
	err := runMove(cmd, []string{"abc", "todo"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
}

// ===== pick.go coverage =====

// --- validatePickFlags ---

func TestValidatePickFlags_InvalidStatus(t *testing.T) {
	cfg := config.NewDefault("Test")
	err := validatePickFlags(cfg, "nonexistent", "")
	if err == nil {
		t.Fatal("expected error for invalid status filter")
	}
}

func TestValidatePickFlags_InvalidMoveTarget(t *testing.T) {
	cfg := config.NewDefault("Test")
	err := validatePickFlags(cfg, "", "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid move target")
	}
}

func TestValidatePickFlags_BothValid(t *testing.T) {
	cfg := config.NewDefault("Test")
	err := validatePickFlags(cfg, "backlog", "todo")
	if err != nil {
		t.Errorf("expected nil for valid flags, got: %v", err)
	}
}

func TestValidatePickFlags_BothEmpty(t *testing.T) {
	cfg := config.NewDefault("Test")
	err := validatePickFlags(cfg, "", "")
	if err != nil {
		t.Errorf("expected nil for empty flags, got: %v", err)
	}
}

// --- executePick ---

func TestExecutePick_NoCandidates(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	// No tasks created — nothing to pick.
	_, _, err = executePick(cfg, "agent", "", "", nil)
	if err == nil {
		t.Fatal("expected error when nothing to pick")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.NothingToPick {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.NothingToPick)
	}
}

func TestExecutePick_Success(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "pickable-task", "backlog")

	picked, oldStatus, pickErr := executePick(cfg, "test-agent", "", "", nil)
	if pickErr != nil {
		t.Fatalf("executePick error: %v", pickErr)
	}
	if picked.ID != 1 {
		t.Errorf("picked ID = %d, want 1", picked.ID)
	}
	if picked.ClaimedBy != "test-agent" {
		t.Errorf("claimed_by = %q, want %q", picked.ClaimedBy, "test-agent")
	}
	if oldStatus != "" {
		t.Errorf("oldStatus = %q, want empty (no move)", oldStatus)
	}
}

func TestExecutePick_WithMove(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "pick-and-move", "backlog")

	picked, oldStatus, pickErr := executePick(cfg, "test-agent", "", "todo", nil)
	if pickErr != nil {
		t.Fatalf("executePick error: %v", pickErr)
	}
	if picked.Status != "todo" {
		t.Errorf("status = %q, want %q", picked.Status, "todo")
	}
	if oldStatus != "backlog" {
		t.Errorf("oldStatus = %q, want %q", oldStatus, "backlog")
	}
}

func TestExecutePick_MoveIdempotent(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "already-there", "todo")

	picked, oldStatus, pickErr := executePick(cfg, "test-agent", "", "todo", nil)
	if pickErr != nil {
		t.Fatalf("executePick error: %v", pickErr)
	}
	if picked.Status != "todo" {
		t.Errorf("status = %q, want %q", picked.Status, "todo")
	}
	// Move target == current status, so no move happened.
	if oldStatus != "" {
		t.Errorf("oldStatus = %q, want empty (idempotent)", oldStatus)
	}
}

func TestExecutePick_StatusFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "in-backlog", "backlog")
	createTaskFileWithStatus(t, cfg.TasksPath(), 2, "in-todo", "todo")

	// Pick only from "todo" — should pick task #2.
	picked, _, pickErr := executePick(cfg, "test-agent", "todo", "", nil)
	if pickErr != nil {
		t.Fatalf("executePick error: %v", pickErr)
	}
	if picked.ID != 2 {
		t.Errorf("picked ID = %d, want 2 (filtered to todo)", picked.ID)
	}
}

func TestExecutePick_WIPViolationOnMove(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.WIPLimits = map[string]int{"todo": 1}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}
	createTaskFileWithStatus(t, cfg.TasksPath(), 1, "existing-todo", "todo")
	createTaskFileWithStatus(t, cfg.TasksPath(), 2, "to-move", "backlog")

	cfg, err = config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = executePick(cfg, "test-agent", "backlog", "todo", nil)
	if err == nil {
		t.Fatal("expected WIP limit error on move")
	}
}

// --- outputPickResult ---

func TestOutputPickResult_JSON(t *testing.T) {
	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	picked := &task.Task{ID: 1, Title: "test-pick", Status: "backlog"}
	err := outputPickResult(picked, "", "agent", false)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("outputPickResult error: %v", err)
	}
	if !containsSubstring(got, "test-pick") {
		t.Errorf("expected task title in JSON output, got: %s", got)
	}
}

func TestOutputPickResult_TableNoMove(t *testing.T) {
	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	picked := &task.Task{ID: 1, Title: "test-pick", Status: "backlog", Body: "task body"}
	err := outputPickResult(picked, "", "agent", false)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("outputPickResult error: %v", err)
	}
	if !containsSubstring(got, "Picked task #1") {
		t.Errorf("expected 'Picked task #1' message, got: %s", got)
	}
	if containsSubstring(got, "->") {
		t.Errorf("expected no move arrow in output, got: %s", got)
	}
	if !containsSubstring(got, "Task #1: test-pick") {
		t.Errorf("expected task details after confirmation, got: %s", got)
	}
	if !containsSubstring(got, "task body") {
		t.Errorf("expected task body in detail output, got: %s", got)
	}
}

func TestOutputPickResult_TableWithMove(t *testing.T) {
	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	picked := &task.Task{ID: 1, Title: "test-pick", Status: "todo"}
	err := outputPickResult(picked, "backlog", "agent", false)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("outputPickResult error: %v", err)
	}
	if !containsSubstring(got, "Picked and moved") {
		t.Errorf("expected 'Picked and moved' message, got: %s", got)
	}
	if !containsSubstring(got, "backlog -> todo") {
		t.Errorf("expected 'backlog -> todo' in output, got: %s", got)
	}
	if !containsSubstring(got, "Task #1: test-pick") {
		t.Errorf("expected task details after confirmation, got: %s", got)
	}
}

func TestOutputPickResult_TableNoBody(t *testing.T) {
	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	picked := &task.Task{ID: 1, Title: "test-pick", Status: "backlog", Body: "task body"}
	err := outputPickResult(picked, "", "agent", true)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("outputPickResult error: %v", err)
	}
	if !containsSubstring(got, "Picked task #1") {
		t.Errorf("expected 'Picked task #1' message, got: %s", got)
	}
	if containsSubstring(got, "Task #1: test-pick") {
		t.Errorf("expected no task details when no-body is true, got: %s", got)
	}
	if containsSubstring(got, "task body") {
		t.Errorf("expected no task body when no-body is true, got: %s", got)
	}
}

// --- runPick ---

func TestRunPick_NoConfig(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newPickCmd()
	_ = cmd.Flags().Set("claim", "agent")
	err := runPick(cmd, nil)
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

// ===== helpers =====

// newMoveCmd creates a cobra.Command with move's flags for unit testing.
func newMoveCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("next", false, "")
	cmd.Flags().Bool("prev", false, "")
	cmd.Flags().String("claim", "", "")
	return cmd
}

// newPickCmd creates a cobra.Command with pick's flags for unit testing.
func newPickCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("claim", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("move", "", "")
	cmd.Flags().StringSlice("tags", nil, "")
	cmd.Flags().Bool("no-body", false, "")
	return cmd
}

// writeTaskWithClass creates a task file with a specific class of service.
func writeTaskWithClass(t *testing.T, tasksDir string, id int, title, status, class string) {
	t.Helper()
	now := time.Now()
	tk := &task.Task{
		ID: id, Title: title, Status: status, Priority: "medium",
		Class: class, Created: now, Updated: now,
	}
	slug := task.GenerateSlug(title)
	filename := task.GenerateFilename(id, slug)
	path := filepath.Join(tasksDir, filename)
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}
}
