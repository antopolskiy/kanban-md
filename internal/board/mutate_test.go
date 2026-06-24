package board_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func setupMutateBoard(t *testing.T) (*config.Config, string) {
	dir := t.TempDir()
	cfg := config.NewDefault(dir)
	cfg.SetDir(dir)
	// Add "review" status so handoff doesn't error out
	cfg.Statuses = append(cfg.Statuses, config.StatusConfig{Name: "review"})
	if err := os.MkdirAll(cfg.TasksPath(), 0755); err != nil {
		t.Fatal(err)
	}
	return cfg, dir
}

func containsSubstring(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func TestHandoff_MoveAndBlock(t *testing.T) {
	cfg, kanbanDir := setupMutateBoard(t)

	tk := &task.Task{ID: 1, Title: "test", Status: "todo", ClaimedBy: "agent-a"}
	err := task.Write(filepath.Join(cfg.TasksPath(), "1.md"), tk)
	if err != nil {
		t.Fatal(err)
	}

	params := board.HandoffParams{
		ID:          1,
		Claimant:    "agent-a",
		Release:     true,
		BlockReason: "waiting",
	}

	_, err = board.Handoff(cfg, params, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath)
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
	if !containsSubstring(got, "release") {
		t.Errorf("expected 'release' in log, got: %s", got)
	}
}

func TestHandoff_ReleaseOnly(t *testing.T) {
	cfg, kanbanDir := setupMutateBoard(t)

	tk := &task.Task{ID: 1, Title: "test", Status: "review", ClaimedBy: "agent-a"}
	err := task.Write(filepath.Join(cfg.TasksPath(), "1.md"), tk)
	if err != nil {
		t.Fatal(err)
	}

	params := board.HandoffParams{
		ID:          1,
		Claimant:    "agent-a",
		Release:     true,
	}

	_, err = board.Handoff(cfg, params, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath)
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
	if containsSubstring(got, "block") {
		t.Errorf("should not log 'block', got: %s", got)
	}
}

func TestHandoff_NoMoveNoBlockNoClaim(t *testing.T) {
	cfg, kanbanDir := setupMutateBoard(t)

	tk := &task.Task{ID: 1, Title: "test", Status: "review", ClaimedBy: "agent-b"}
	err := task.Write(filepath.Join(cfg.TasksPath(), "1.md"), tk)
	if err != nil {
		t.Fatal(err)
	}

	params := board.HandoffParams{
		ID:          1,
		Claimant:    "agent-b", // keeping claim
		Release:     false,
	}

	_, err = board.Handoff(cfg, params, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath)
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

func TestPickAndClaim_NoCandidates(t *testing.T) {
	cfg, _ := setupMutateBoard(t)

	params := board.PickAndClaimParams{
		Claimant: "agent-a",
	}

	_, _, err := board.PickAndClaim(cfg, params, time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
