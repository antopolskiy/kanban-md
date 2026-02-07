package cmd

import (
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func testConfig() *config.Config {
	return config.NewDefault("Test")
}

func TestUpdateTimestamps_FirstMoveFromInitial(t *testing.T) {
	cfg := testConfig()
	// Default statuses: [backlog, todo, in-progress, review, done]
	tk := &task.Task{Status: "todo"}

	updateTimestamps(tk, "backlog", "todo", cfg)

	if tk.Started == nil {
		t.Fatal("Started should be set on first move out of initial status")
	}
	if tk.Completed != nil {
		t.Error("Completed should be nil")
	}
}

func TestUpdateTimestamps_SubsequentMovePreservesStarted(t *testing.T) {
	cfg := testConfig()
	originalStarted := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	tk := &task.Task{Status: "in-progress", Started: &originalStarted}

	updateTimestamps(tk, "todo", "in-progress", cfg)

	if tk.Started != &originalStarted {
		t.Error("Started should not be overwritten on subsequent moves")
	}
}

func TestUpdateTimestamps_MoveToTerminal(t *testing.T) {
	cfg := testConfig()
	started := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	tk := &task.Task{Status: "done", Started: &started}

	updateTimestamps(tk, "review", "done", cfg)

	if tk.Completed == nil {
		t.Fatal("Completed should be set on move to terminal status")
	}
	if tk.Started != &started {
		t.Error("Started should be preserved")
	}
}

func TestUpdateTimestamps_DirectToTerminal(t *testing.T) {
	cfg := testConfig()
	tk := &task.Task{Status: "done"}

	updateTimestamps(tk, "backlog", "done", cfg)

	if tk.Started == nil {
		t.Fatal("Started should be set on direct move to terminal")
	}
	if tk.Completed == nil {
		t.Fatal("Completed should be set on direct move to terminal")
	}
}

func TestUpdateTimestamps_ReopenClearsCompleted(t *testing.T) {
	cfg := testConfig()
	started := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	tk := &task.Task{Status: "review", Started: &started, Completed: &completed}

	updateTimestamps(tk, "done", "review", cfg)

	if tk.Completed != nil {
		t.Error("Completed should be cleared when moving back from terminal")
	}
	if tk.Started == nil {
		t.Error("Started should be preserved when reopening")
	}
}

func TestUpdateTimestamps_MiddleMoveNoChange(t *testing.T) {
	cfg := testConfig()
	started := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	tk := &task.Task{Status: "in-progress", Started: &started}

	updateTimestamps(tk, "todo", "in-progress", cfg)

	if tk.Started != &started {
		t.Error("Started should not change on middle status move")
	}
	if tk.Completed != nil {
		t.Error("Completed should remain nil on middle status move")
	}
}
