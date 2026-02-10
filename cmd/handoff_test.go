package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const (
	testHandoffAgent = "agent-1"
	testReviewStatus = "review"
)

// newHandoffCmd creates a fresh cobra command with handoff flags for testing.
func newHandoffCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("claim", "", "")
	cmd.Flags().String("note", "", "")
	cmd.Flags().BoolP("timestamp", "t", false, "")
	cmd.Flags().String("block", "", "")
	cmd.Flags().Bool("release", false, "")
	return cmd
}

// setupHandoffTask creates a board with a task in in-progress claimed by testHandoffAgent.
func setupHandoffTask(t *testing.T) *config.Config {
	t.Helper()
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeHandoffTask(t, cfg, &task.Task{
		ID:        1,
		Title:     "Handoff test task",
		Status:    "in-progress",
		Priority:  "medium",
		ClaimedBy: testHandoffAgent,
		ClaimedAt: &now,
		Created:   now,
		Updated:   now,
	})
	return cfg
}

// writeHandoffTask writes a task file to the given config's tasks directory.
func writeHandoffTask(t *testing.T, cfg *config.Config, tk *task.Task) {
	t.Helper()
	slug := task.GenerateSlug(tk.Title)
	filename := task.GenerateFilename(tk.ID, slug)
	path := filepath.Join(cfg.TasksPath(), filename)
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}
}

// --- executeHandoff tests ---

func TestExecuteHandoff_BasicMoveToReview(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	if got.Status != testReviewStatus {
		t.Errorf("Status = %q, want %q", got.Status, testReviewStatus)
	}
	if got.ClaimedBy != testHandoffAgent {
		t.Errorf("ClaimedBy = %q, want %q", got.ClaimedBy, testHandoffAgent)
	}
}

func TestExecuteHandoff_AlreadyAtReview(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeHandoffTask(t, cfg, &task.Task{
		ID:        1,
		Title:     "Already review",
		Status:    testReviewStatus,
		Priority:  "medium",
		ClaimedBy: testHandoffAgent,
		ClaimedAt: &now,
		Created:   now,
		Updated:   now,
	})

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	// Should stay at review without error.
	if got.Status != testReviewStatus {
		t.Errorf("Status = %q, want %q", got.Status, testReviewStatus)
	}
}

func TestExecuteHandoff_MissingClaimFlag(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	// No --claim flag set.

	_, err := executeHandoff(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected error for missing claim")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidInput {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidInput)
	}
}

func TestExecuteHandoff_DifferentClaimant(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", "agent-2")

	_, err := executeHandoff(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected error for different claimant")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.TaskClaimed {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.TaskClaimed)
	}
}

func TestExecuteHandoff_TaskNotFound(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)

	_, err = executeHandoff(cfg, 999, cmd)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestExecuteHandoff_WithBlock(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("block", "Waiting on creds")

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	if !got.Blocked {
		t.Error("expected task to be blocked")
	}
	if got.BlockReason != "Waiting on creds" {
		t.Errorf("BlockReason = %q, want %q", got.BlockReason, "Waiting on creds")
	}
}

func TestExecuteHandoff_EmptyBlockReason(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("block", "")

	_, err := executeHandoff(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected error for empty block reason")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidInput {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidInput)
	}
}

func TestExecuteHandoff_WithNote(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("note", "Branch: task/1-feature")

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	if !strings.Contains(got.Body, "Branch: task/1-feature") {
		t.Errorf("Body = %q, want note text", got.Body)
	}
}

func TestExecuteHandoff_WithNoteAndTimestamp(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("note", "progress update")
	_ = cmd.Flags().Set("timestamp", "true")

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	if !strings.Contains(got.Body, "[[") || !strings.Contains(got.Body, "]]") {
		t.Errorf("Body should contain timestamp markers, got %q", got.Body)
	}
	if !strings.Contains(got.Body, "progress update") {
		t.Errorf("Body should contain note text, got %q", got.Body)
	}
}

func TestExecuteHandoff_NoteAppendsToExistingBody(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeHandoffTask(t, cfg, &task.Task{
		ID:        1,
		Title:     "Has body",
		Status:    "in-progress",
		Priority:  "medium",
		Body:      "Existing context here.",
		ClaimedBy: testHandoffAgent,
		ClaimedAt: &now,
		Created:   now,
		Updated:   now,
	})

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("note", "new handoff note")

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	if !strings.Contains(got.Body, "Existing context here.") {
		t.Errorf("Body should preserve existing content, got %q", got.Body)
	}
	if !strings.Contains(got.Body, "new handoff note") {
		t.Errorf("Body should contain new note, got %q", got.Body)
	}
}

func TestExecuteHandoff_WithRelease(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("release", "true")

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	if got.ClaimedBy != "" {
		t.Errorf("ClaimedBy = %q, want empty (released)", got.ClaimedBy)
	}
	if got.ClaimedAt != nil {
		t.Error("ClaimedAt should be nil after release")
	}
}

func TestExecuteHandoff_AllFlagsCombined(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeHandoffTask(t, cfg, &task.Task{
		ID:        1,
		Title:     "Full handoff",
		Status:    "in-progress",
		Priority:  "medium",
		Body:      "existing",
		ClaimedBy: testHandoffAgent,
		ClaimedAt: &now,
		Created:   now,
		Updated:   now,
	})

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("block", "Need deploy creds")
	_ = cmd.Flags().Set("note", "Branch: task/1-full")
	_ = cmd.Flags().Set("timestamp", "true")
	_ = cmd.Flags().Set("release", "true")

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	if got.Status != testReviewStatus {
		t.Errorf("Status = %q, want %q", got.Status, testReviewStatus)
	}
	if !got.Blocked {
		t.Error("expected blocked")
	}
	if got.BlockReason != "Need deploy creds" {
		t.Errorf("BlockReason = %q, want %q", got.BlockReason, "Need deploy creds")
	}
	if got.ClaimedBy != "" {
		t.Errorf("ClaimedBy = %q, want empty (released)", got.ClaimedBy)
	}
	if !strings.Contains(got.Body, "existing") {
		t.Errorf("Body should preserve existing content, got %q", got.Body)
	}
	if !strings.Contains(got.Body, "Branch: task/1-full") {
		t.Errorf("Body should contain note, got %q", got.Body)
	}
	if !strings.Contains(got.Body, "[[") {
		t.Errorf("Body should contain timestamp, got %q", got.Body)
	}
}

// --- handoffSingleTask tests ---

func TestHandoffSingleTask_TableOutput(t *testing.T) {
	cfg := setupHandoffTask(t)
	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)

	err := handoffSingleTask(cfg, 1, cmd)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("handoffSingleTask error: %v", err)
	}
	if !containsSubstring(got, "Handed off task #1") {
		t.Errorf("expected 'Handed off task #1' in output, got: %s", got)
	}
	if !containsSubstring(got, testReviewStatus) {
		t.Errorf("expected %q in output, got: %s", testReviewStatus, got)
	}
}

func TestHandoffSingleTask_TableOutputBlocked(t *testing.T) {
	cfg := setupHandoffTask(t)
	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("block", "Waiting on user")

	err := handoffSingleTask(cfg, 1, cmd)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("handoffSingleTask error: %v", err)
	}
	if !containsSubstring(got, "blocked") {
		t.Errorf("expected 'blocked' in output, got: %s", got)
	}
	if !containsSubstring(got, "Waiting on user") {
		t.Errorf("expected block reason in output, got: %s", got)
	}
}

func TestHandoffSingleTask_TableOutputReleased(t *testing.T) {
	cfg := setupHandoffTask(t)
	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	_ = cmd.Flags().Set("release", "true")

	err := handoffSingleTask(cfg, 1, cmd)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("handoffSingleTask error: %v", err)
	}
	if !containsSubstring(got, "claim released") {
		t.Errorf("expected 'claim released' in output, got: %s", got)
	}
}

func TestHandoffSingleTask_JSONOutput(t *testing.T) {
	cfg := setupHandoffTask(t)
	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)

	err := handoffSingleTask(cfg, 1, cmd)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("handoffSingleTask error: %v", err)
	}
	if !containsSubstring(got, `"status"`) {
		t.Errorf("expected JSON with status field, got: %s", got)
	}
	if !containsSubstring(got, `"review"`) {
		t.Errorf("expected review status in JSON, got: %s", got)
	}
}

func TestHandoffSingleTask_PropagatesError(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	// No --claim → error.

	err := handoffSingleTask(cfg, 1, cmd)
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}

// --- logHandoffActivity tests ---

func TestLogHandoffActivity_MoveAndBlock(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", Status: testReviewStatus, Blocked: true, BlockReason: "waiting"}
	logHandoffActivity(cfg, tk, "in-progress")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "move") {
		t.Errorf("expected 'move' in log, got: %s", got)
	}
	if !containsSubstring(got, "handoff") {
		t.Errorf("expected 'handoff' in log, got: %s", got)
	}
	if !containsSubstring(got, "block") {
		t.Errorf("expected 'block' in log, got: %s", got)
	}
}

func TestLogHandoffActivity_ReleaseOnly(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// ClaimedBy="" means claim was released.
	tk := &task.Task{ID: 1, Title: "test", Status: testReviewStatus, ClaimedBy: ""}
	logHandoffActivity(cfg, tk, testReviewStatus)

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "release") {
		t.Errorf("expected 'release' in log, got: %s", got)
	}
	// Should not log move (same status).
	if containsSubstring(got, "move") {
		t.Errorf("should not log 'move' when status unchanged, got: %s", got)
	}
}

func TestLogHandoffActivity_NoMoveNoBlockNoClaim(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Same status, not blocked, still claimed → only handoff log entry.
	tk := &task.Task{ID: 1, Title: "test", Status: testReviewStatus, ClaimedBy: testHandoffAgent}
	logHandoffActivity(cfg, tk, testReviewStatus)

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "handoff") {
		t.Errorf("expected 'handoff' in log, got: %s", got)
	}
	if containsSubstring(got, "move") {
		t.Errorf("should not log 'move', got: %s", got)
	}
	if containsSubstring(got, "block") {
		t.Errorf("should not log 'block', got: %s", got)
	}
	if containsSubstring(got, "release") {
		t.Errorf("should not log 'release', got: %s", got)
	}
}

func TestExecuteHandoff_ExpiredClaimAllowsHandoff(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create task with an expired claim from agent-1.
	past := time.Now().Add(-48 * time.Hour)
	writeHandoffTask(t, cfg, &task.Task{
		ID:        1,
		Title:     "Expired claim",
		Status:    "in-progress",
		Priority:  "medium",
		ClaimedBy: testHandoffAgent,
		ClaimedAt: &past,
		Created:   past,
		Updated:   past,
	})

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", "agent-2")

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("expected expired claim to allow handoff, got: %v", err)
	}
	if got.ClaimedBy != "agent-2" {
		t.Errorf("ClaimedBy = %q, want %q", got.ClaimedBy, "agent-2")
	}
}

func TestExecuteHandoff_NoNoteNoChange(t *testing.T) {
	cfg := setupHandoffTask(t)

	cmd := newHandoffCmd()
	_ = cmd.Flags().Set("claim", testHandoffAgent)
	// No --note flag set.

	got, err := executeHandoff(cfg, 1, cmd)
	if err != nil {
		t.Fatalf("executeHandoff error: %v", err)
	}
	// Body should remain empty since no note was given and task had no body.
	if got.Body != "" {
		t.Errorf("Body = %q, want empty", got.Body)
	}
}
