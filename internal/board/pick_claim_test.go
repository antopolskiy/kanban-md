package board_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestPickAndClaim_Success(t *testing.T) {
	cfg, kanbanDir := setupMutateBoard(t)
	now := time.Now()

	// Write a candidate task.
	tk := &task.Task{ID: 1, Title: "test", Status: "todo"}
	err := task.Write(filepath.Join(cfg.TasksPath(), "1.md"), tk)
	if err != nil {
		t.Fatal(err)
	}

	params := board.PickAndClaimParams{
		Claimant: "agent-test",
	}

	picked, oldStatus, _, err := board.PickAndClaim(cfg, params, now)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if picked == nil {
		t.Fatal("expected picked task, got nil")
	}

	// Verify claim was set
	if picked.ClaimedBy != "agent-test" {
		t.Errorf("expected ClaimedBy = agent-test, got %q", picked.ClaimedBy)
	}
	if picked.ClaimedAt == nil || !picked.ClaimedAt.Equal(now) {
		t.Errorf("expected ClaimedAt = %v, got %v", now, picked.ClaimedAt)
	}

	// Verify file was saved correctly
	saved, err := task.Read(filepath.Join(cfg.TasksPath(), "1.md"))
	if err != nil {
		t.Fatal(err)
	}
	if saved.ClaimedBy != "agent-test" {
		t.Errorf("saved task ClaimedBy = %q, want agent-test", saved.ClaimedBy)
	}

	// Verify no move occurred since we didn't ask for one
	if oldStatus != "" {
		t.Errorf("expected empty oldStatus, got %q", oldStatus)
	}

	// Verify activity log
	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, _ := os.ReadFile(logPath)
	if !containsSubstring(string(data), "claim") {
		t.Errorf("expected 'claim' log, got: %s", data)
	}
	if containsSubstring(string(data), "move") {
		t.Errorf("did not expect 'move' log, got: %s", data)
	}
}

func TestPickAndClaim_WithMove(t *testing.T) {
	cfg, kanbanDir := setupMutateBoard(t)
	now := time.Now()

	tk := &task.Task{ID: 2, Title: "test", Status: "todo"}
	err := task.Write(filepath.Join(cfg.TasksPath(), "2.md"), tk)
	if err != nil {
		t.Fatal(err)
	}

	params := board.PickAndClaimParams{
		Claimant:   "agent-move",
		MoveTarget: "in-progress",
	}

	picked, oldStatus, _, err := board.PickAndClaim(cfg, params, now)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if picked.Status != "in-progress" {
		t.Errorf("expected status 'in-progress', got %q", picked.Status)
	}
	if oldStatus != "todo" {
		t.Errorf("expected oldStatus 'todo', got %q", oldStatus)
	}

	// Verify activity log has BOTH claim and move
	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, _ := os.ReadFile(logPath)
	if !containsSubstring(string(data), "claim") {
		t.Errorf("expected 'claim' log, got: %s", data)
	}
	if !containsSubstring(string(data), "move") {
		t.Errorf("expected 'move' log, got: %s", data)
	}
}

func TestPickAndClaim_WIPExceeded(t *testing.T) {
	cfg, _ := setupMutateBoard(t)

	// Create an existing task in 'in-progress' to fill WIP limit
	cfg.WIPLimits = map[string]int{"in-progress": 1}
	_ = cfg.Save() // Save to disk in case needed, though config is in mem

	tk1 := &task.Task{ID: 1, Title: "wip-filler", Status: "in-progress", ClaimedBy: "other"}
	_ = task.Write(filepath.Join(cfg.TasksPath(), "1.md"), tk1)

	tk2 := &task.Task{ID: 2, Title: "test", Status: "todo"}
	_ = task.Write(filepath.Join(cfg.TasksPath(), "2.md"), tk2)

	params := board.PickAndClaimParams{
		Claimant:   "agent-wip",
		MoveTarget: "in-progress",
	}

	// Should fail because in-progress WIP is full
	picked, _, _, err := board.PickAndClaim(cfg, params, time.Now())
	if err == nil {
		t.Fatal("expected WIP exceeded error, got nil")
	}
	if picked != nil {
		t.Errorf("expected nil task, got %v", picked)
	}
	if !containsSubstring(err.Error(), "WIP limit reached") {
		t.Errorf("expected WIP error message, got: %v", err)
	}
}

func TestPickAndClaim_RequiresClaimant(t *testing.T) {
	cfg, _ := setupMutateBoard(t)

	tk := &task.Task{ID: 1, Title: "test", Status: "todo"}
	if err := task.Write(filepath.Join(cfg.TasksPath(), "1.md"), tk); err != nil {
		t.Fatal(err)
	}

	picked, _, _, err := board.PickAndClaim(cfg, board.PickAndClaimParams{}, time.Now())
	if err == nil {
		t.Fatal("expected claim required error, got nil")
	}
	if picked != nil {
		t.Errorf("expected nil task, got %v", picked)
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.InvalidInput {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidInput)
	}
}

func TestPickAndClaim_ValidatesStatusParams(t *testing.T) {
	tests := []struct {
		name   string
		params board.PickAndClaimParams
	}{
		{
			name: "status filter",
			params: board.PickAndClaimParams{
				Claimant:     "agent-status",
				StatusFilter: "not-a-status",
			},
		},
		{
			name: "move target",
			params: board.PickAndClaimParams{
				Claimant:   "agent-status",
				MoveTarget: "not-a-status",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := setupMutateBoard(t)
			tk := &task.Task{ID: 1, Title: "test", Status: "todo"}
			if err := task.Write(filepath.Join(cfg.TasksPath(), "1.md"), tk); err != nil {
				t.Fatal(err)
			}

			picked, _, _, err := board.PickAndClaim(cfg, tt.params, time.Now())
			if err == nil {
				t.Fatal("expected invalid status error, got nil")
			}
			if picked != nil {
				t.Errorf("expected nil task, got %v", picked)
			}
			var cliErr *clierr.Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("expected clierr.Error, got %T: %v", err, err)
			}
			if cliErr.Code != clierr.InvalidStatus {
				t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidStatus)
			}
		})
	}
}
