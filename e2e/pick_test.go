package e2e_test

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Pick command tests
// ---------------------------------------------------------------------------

func TestPickBasic(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Pick me", "--priority", "high")
	mustCreateTask(t, kanbanDir, "Lower priority")

	// Pick should select the highest-priority unclaimed task.
	var picked taskJSON
	r := runKanbanJSON(t, kanbanDir, &picked, "pick", "--claim", claimAgent1)
	if r.exitCode != 0 {
		t.Fatalf("pick failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if picked.Title != "Pick me" {
		t.Errorf("picked title = %q, want %q", picked.Title, "Pick me")
	}
	if picked.ClaimedBy != claimAgent1 {
		t.Errorf("claimed_by = %q, want %q", picked.ClaimedBy, claimAgent1)
	}
}

func TestPickWithMove(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Move me")

	var picked taskJSON
	r := runKanbanJSON(t, kanbanDir, &picked, "pick", "--claim", claimAgent1, "--move", "in-progress")
	if r.exitCode != 0 {
		t.Fatalf("pick --move failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if picked.Status != statusInProgress {
		t.Errorf("status = %q, want %q", picked.Status, "in-progress")
	}
}

func TestPickWithStatusFilter(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Backlog task")
	mustCreateTask(t, kanbanDir, "Todo task", "--status", statusTodo)

	var picked taskJSON
	r := runKanbanJSON(t, kanbanDir, &picked, "pick", "--claim", claimAgent1, "--status", statusTodo)
	if r.exitCode != 0 {
		t.Fatalf("pick --status failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if picked.Title != "Todo task" {
		t.Errorf("picked title = %q, want %q", picked.Title, "Todo task")
	}
}

func TestPickWithTagFilter(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "No tag task")
	mustCreateTask(t, kanbanDir, "Tagged task", "--tags", "urgent")

	var picked taskJSON
	r := runKanbanJSON(t, kanbanDir, &picked, "pick", "--claim", claimAgent1, "--tags", "urgent")
	if r.exitCode != 0 {
		t.Fatalf("pick --tags failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if picked.Title != "Tagged task" {
		t.Errorf("picked title = %q, want %q", picked.Title, "Tagged task")
	}
}

func TestPickNothingAvailable(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "pick", "--claim", claimAgent1)
	if errResp.Code != "NOTHING_TO_PICK" {
		t.Errorf("error code = %q, want NOTHING_TO_PICK", errResp.Code)
	}
}

func TestPickTableOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Table pick task", "--body", "task body")

	r := runKanban(t, kanbanDir, "pick", "--claim", claimAgent1)
	if r.exitCode != 0 {
		t.Fatalf("pick table output failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Picked task") {
		t.Error("table output should contain 'Picked task'")
	}
	if !strings.Contains(r.stdout, claimAgent1) {
		t.Error("table output should contain claimant name")
	}
	if !strings.Contains(r.stdout, "Task #1: Table pick task") {
		t.Errorf("table output should include task details, got:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "task body") {
		t.Errorf("table output should include task body, got:\n%s", r.stdout)
	}
}

func TestPickTableOutputWithMove(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Table move task")

	r := runKanban(t, kanbanDir, "pick", "--claim", claimAgent1, "--move", "in-progress")
	if r.exitCode != 0 {
		t.Fatalf("pick --move table output failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Picked and moved") {
		t.Error("table output should contain 'Picked and moved'")
	}
	if !strings.Contains(r.stdout, "Task #1: Table move task") {
		t.Errorf("table output should include task details, got:\n%s", r.stdout)
	}
}

func TestPickTableOutputNoBody(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "No body pick task", "--body", "do not show")

	r := runKanban(t, kanbanDir, "pick", "--claim", claimAgent1, "--no-body")
	if r.exitCode != 0 {
		t.Fatalf("pick --no-body failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Picked task") {
		t.Error("table output should contain 'Picked task'")
	}
	if strings.Contains(r.stdout, "Task #1: No body pick task") {
		t.Errorf("table output should not include task details with --no-body, got:\n%s", r.stdout)
	}
	if strings.Contains(r.stdout, "do not show") {
		t.Errorf("table output should not include task body with --no-body, got:\n%s", r.stdout)
	}
}

func TestPickCompactOutputIncludesDetails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Compact pick task", "--body", "compact body")

	r := runKanban(t, kanbanDir, "--compact", "pick", "--claim", claimAgent1)
	if r.exitCode != 0 {
		t.Fatalf("compact pick failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Picked task #1: Compact pick task") {
		t.Errorf("compact output missing pick confirmation, got:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "#1 [backlog/medium] Compact pick task") {
		t.Errorf("compact output missing task details, got:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "compact body") {
		t.Errorf("compact output missing task body, got:\n%s", r.stdout)
	}
}

func TestPickInvalidStatus(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "pick", "--claim", claimAgent1, "--status", "nonexistent")
	if errResp.Code != codeInvalidStatus {
		t.Errorf("error code = %q, want INVALID_STATUS", errResp.Code)
	}
}

func TestPickInvalidMoveTarget(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "pick", "--claim", claimAgent1, "--move", "nonexistent")
	if errResp.Code != codeInvalidStatus {
		t.Errorf("error code = %q, want INVALID_STATUS", errResp.Code)
	}
}

// ---------------------------------------------------------------------------
// Class-aware WIP limit tests
// ---------------------------------------------------------------------------
