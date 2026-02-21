package e2e_test

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Config command tests
// ---------------------------------------------------------------------------

func TestConfigShowAll(t *testing.T) {
	kanbanDir := initBoard(t)

	var cfg map[string]any
	runKanbanJSON(t, kanbanDir, &cfg, "config")

	// Verify expected keys are present.
	expectedKeys := []string{
		"version", "board.name", "board.description", "tasks_dir",
		"statuses", "priorities", "defaults.status", "defaults.priority", "defaults.class",
		"wip_limits", "claim_timeout", "classes",
		"tui.title_lines", "tui.hide_empty_columns", "tui.age_thresholds", "next_id",
	}
	for _, key := range expectedKeys {
		if _, ok := cfg[key]; !ok {
			t.Errorf("missing key %q in config output", key)
		}
	}

	// board.name is derived from CWD during init; just verify it's non-empty.
	if cfg["board.name"] == "" {
		t.Error("board.name is empty")
	}
}

func TestConfigGetBoardName(t *testing.T) {
	kanbanDir := initBoard(t)

	var name string
	runKanbanJSON(t, kanbanDir, &name, "config", "get", "board.name")
	if name == "" {
		t.Error("board.name is empty")
	}
}

func TestConfigGetStatuses(t *testing.T) {
	kanbanDir := initBoard(t)

	var statuses []string
	runKanbanJSON(t, kanbanDir, &statuses, "config", "get", "statuses")
	const wantStatusCount = 6 // includes "archived"
	if len(statuses) != wantStatusCount {
		t.Fatalf("statuses = %v, want %d items", statuses, wantStatusCount)
	}
	if statuses[0] != statusBacklog {
		t.Errorf("statuses[0] = %q, want %q", statuses[0], statusBacklog)
	}
}

func TestConfigSetDefaultPriority(t *testing.T) {
	kanbanDir := initBoard(t)

	// Set default priority to high.
	runKanban(t, kanbanDir, "--json", "config", "set", "defaults.priority", "high")

	// Verify it persisted.
	var val string
	runKanbanJSON(t, kanbanDir, &val, "config", "get", "defaults.priority")
	if val != priorityHigh {
		t.Errorf("defaults.priority = %q, want %q", val, priorityHigh)
	}
}

func TestConfigSetBoardName(t *testing.T) {
	kanbanDir := initBoard(t)

	runKanban(t, kanbanDir, "--json", "config", "set", "board.name", "My New Board")

	var val string
	runKanbanJSON(t, kanbanDir, &val, "config", "get", "board.name")
	if val != "My New Board" {
		t.Errorf("board.name = %q, want %q", val, "My New Board")
	}
}

func TestConfigSetClaimTimeout(t *testing.T) {
	kanbanDir := initBoard(t)

	runKanban(t, kanbanDir, "--json", "config", "set", "claim_timeout", "2h")

	var val string
	runKanbanJSON(t, kanbanDir, &val, "config", "get", "claim_timeout")
	if val != "2h" {
		t.Errorf("claim_timeout = %q, want %q", val, "2h")
	}
}

func TestConfigSetDefaultsClass(t *testing.T) {
	kanbanDir := initBoard(t)

	runKanban(t, kanbanDir, "--json", "config", "set", "defaults.class", "expedite")

	var val string
	runKanbanJSON(t, kanbanDir, &val, "config", "get", "defaults.class")
	if val != "expedite" {
		t.Errorf("defaults.class = %q, want %q", val, "expedite")
	}
}

func TestConfigSetTUIHideEmptyColumns(t *testing.T) {
	kanbanDir := initBoard(t)

	runKanban(t, kanbanDir, "--json", "config", "set", "tui.hide_empty_columns", "true")

	var val bool
	runKanbanJSON(t, kanbanDir, &val, "config", "get", "tui.hide_empty_columns")
	if !val {
		t.Error("tui.hide_empty_columns = false, want true")
	}
}

func TestConfigSetReadOnlyKey(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "config", "set", "next_id", "99")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "read-only") {
		t.Errorf("error = %q, want 'read-only'", errResp.Error)
	}
}

func TestConfigGetInvalidKey(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "config", "get", "nonexistent.key")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "unknown config key") {
		t.Errorf("error = %q, want 'unknown config key'", errResp.Error)
	}
}

func TestConfigSetInvalidDefaultStatus(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "config", "set", "defaults.status", "nonexistent")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "invalid default status") {
		t.Errorf("error = %q, want 'invalid default status'", errResp.Error)
	}
}

func TestConfigSetInvalidClaimTimeout(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "config", "set", "claim_timeout", "soon")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "invalid claim_timeout") {
		t.Errorf("error = %q, want 'invalid claim_timeout'", errResp.Error)
	}
}

func TestConfigTableOutput(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--table", "config")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", r.exitCode)
	}
	if !strings.Contains(r.stdout, "board.name") {
		t.Errorf("table output missing board.name:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "defaults.status") {
		t.Errorf("table output missing defaults.status:\n%s", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Context command tests
// ---------------------------------------------------------------------------
