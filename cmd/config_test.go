package cmd

import (
	"testing"

	"github.com/antopolskiy/kanban-md/internal/config"
)

const classExpedite = "expedite"

// --- configAccessors tests ---

func TestConfigAccessors_AllKeysHaveAccessors(t *testing.T) {
	accessors := configAccessors()
	keys := allConfigKeys()

	for _, key := range keys {
		if _, ok := accessors[key]; !ok {
			t.Errorf("allConfigKeys contains %q but no accessor exists", key)
		}
	}
}

func TestAllConfigKeys_ExpectedCoverage(t *testing.T) {
	keys := allConfigKeys()
	expected := []string{
		"version",
		"board.name",
		"board.description",
		"tasks_dir",
		"statuses",
		"priorities",
		"defaults.status",
		"defaults.priority",
		"defaults.class",
		"wip_limits",
		"claim_timeout",
		"classes",
		"tui.title_lines",
		"tui.hide_empty_columns",
		"tui.age_thresholds",
		"next_id",
	}

	if len(keys) != len(expected) {
		t.Fatalf("allConfigKeys len = %d, want %d", len(keys), len(expected))
	}
	for i := range expected {
		if keys[i] != expected[i] {
			t.Fatalf("allConfigKeys[%d] = %q, want %q", i, keys[i], expected[i])
		}
	}
}

func TestConfigAccessors_GetBoardName(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault(testBoardName)

	got := accessors["board.name"].get(cfg)
	if got != testBoardName {
		t.Errorf("board.name = %v, want TestBoard", got)
	}
}

func TestConfigAccessors_SetBoardName(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Old")

	if err := accessors["board.name"].set(cfg, "New"); err != nil {
		t.Fatal(err)
	}
	if cfg.Board.Name != "New" {
		t.Errorf("board.name = %q, want %q", cfg.Board.Name, "New")
	}
}

func TestConfigAccessors_SetDefaultsStatus_Valid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	if err := accessors["defaults.status"].set(cfg, statusInProgress); err != nil {
		t.Fatal(err)
	}
	if cfg.Defaults.Status != statusInProgress {
		t.Errorf("defaults.status = %q, want %q", cfg.Defaults.Status, statusInProgress)
	}
}

func TestConfigAccessors_SetDefaultsStatus_Invalid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	err := accessors["defaults.status"].set(cfg, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid default status")
	}
}

func TestConfigAccessors_SetDefaultsPriority_Valid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	if err := accessors["defaults.priority"].set(cfg, priorityHigh); err != nil {
		t.Fatal(err)
	}
	if cfg.Defaults.Priority != priorityHigh {
		t.Errorf("defaults.priority = %q, want %q", cfg.Defaults.Priority, priorityHigh)
	}
}

func TestConfigAccessors_SetDefaultsPriority_Invalid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	err := accessors["defaults.priority"].set(cfg, "ultra")
	if err == nil {
		t.Fatal("expected error for invalid default priority")
	}
}

func TestConfigAccessors_SetDefaultsClass_Valid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	if err := accessors["defaults.class"].set(cfg, classExpedite); err != nil {
		t.Fatal(err)
	}
	if cfg.Defaults.Class != classExpedite {
		t.Errorf("defaults.class = %q, want %q", cfg.Defaults.Class, classExpedite)
	}
}

func TestConfigAccessors_SetDefaultsClass_Invalid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	err := accessors["defaults.class"].set(cfg, "ultra")
	if err == nil {
		t.Fatal("expected error for invalid default class")
	}
}

func TestConfigAccessors_SetClaimTimeout(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	if err := accessors["claim_timeout"].set(cfg, "2h"); err != nil {
		t.Fatal(err)
	}
	if cfg.ClaimTimeout != "2h" {
		t.Errorf("claim_timeout = %q, want %q", cfg.ClaimTimeout, "2h")
	}
}

func TestConfigAccessors_SetClaimTimeout_Invalid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	err := accessors["claim_timeout"].set(cfg, "soon")
	if err == nil {
		t.Fatal("expected error for invalid claim_timeout")
	}
}

func TestConfigAccessors_SetTUITitleLines(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	if err := accessors["tui.title_lines"].set(cfg, "3"); err != nil {
		t.Fatal(err)
	}
	if cfg.TUI.TitleLines != 3 {
		t.Errorf("tui.title_lines = %d, want 3", cfg.TUI.TitleLines)
	}
}

func TestConfigAccessors_SetTUITitleLines_Invalid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	err := accessors["tui.title_lines"].set(cfg, "abc")
	if err == nil {
		t.Fatal("expected error for non-numeric title_lines")
	}
}

func TestConfigAccessors_SetTUIHideEmptyColumns(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	if err := accessors["tui.hide_empty_columns"].set(cfg, "true"); err != nil {
		t.Fatal(err)
	}
	if !cfg.TUI.HideEmptyColumns {
		t.Error("tui.hide_empty_columns = false, want true")
	}
}

func TestConfigAccessors_SetTUIHideEmptyColumns_Invalid(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	err := accessors["tui.hide_empty_columns"].set(cfg, "nope")
	if err == nil {
		t.Fatal("expected error for non-boolean hide_empty_columns")
	}
}

func TestConfigAccessors_ReadOnlyKeys(t *testing.T) {
	accessors := configAccessors()
	readOnlyKeys := []string{
		"statuses", "priorities", "tasks_dir", "next_id", "version",
		"wip_limits", "classes", "tui.age_thresholds",
	}

	for _, key := range readOnlyKeys {
		acc, ok := accessors[key]
		if !ok {
			t.Errorf("accessor for %q not found", key)
			continue
		}
		if acc.writable {
			t.Errorf("key %q should be read-only", key)
		}
	}
}

func TestConfigAccessors_WritableKeys(t *testing.T) {
	accessors := configAccessors()
	writableKeys := []string{
		"board.name", "board.description", "defaults.status", "defaults.priority",
		"defaults.class", "claim_timeout", "tui.title_lines", "tui.hide_empty_columns",
	}

	for _, key := range writableKeys {
		acc, ok := accessors[key]
		if !ok {
			t.Errorf("accessor for %q not found", key)
			continue
		}
		if !acc.writable {
			t.Errorf("key %q should be writable", key)
		}
	}
}

func TestConfigAccessors_WIPLimitsNil(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")
	cfg.WIPLimits = nil

	got := accessors["wip_limits"].get(cfg)
	m, ok := got.(map[string]int)
	if !ok {
		t.Fatalf("expected map[string]int, got %T", got)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map for nil WIP limits, got %v", m)
	}
}

// --- formatConfigValue tests ---

func TestFormatConfigValue_StringSlice(t *testing.T) {
	got := formatConfigValue([]string{"a", "b", "c"})
	want := "a, b, c"
	if got != want {
		t.Errorf("formatConfigValue = %q, want %q", got, want)
	}
}

func TestFormatConfigValue_EmptyMap(t *testing.T) {
	got := formatConfigValue(map[string]int{})
	if got != "--" {
		t.Errorf("formatConfigValue = %q, want %q", got, "--")
	}
}

func TestFormatConfigValue_MapWithValues(t *testing.T) {
	got := formatConfigValue(map[string]int{"review": 5})
	if !containsSubstring(got, "review=5") {
		t.Errorf("formatConfigValue = %q, want to contain review=5", got)
	}
}

func TestFormatConfigValue_String(t *testing.T) {
	got := formatConfigValue("hello")
	if got != "hello" {
		t.Errorf("formatConfigValue = %q, want %q", got, "hello")
	}
}

func TestFormatConfigValue_Int(t *testing.T) {
	got := formatConfigValue(42)
	if got != "42" {
		t.Errorf("formatConfigValue = %q, want %q", got, "42")
	}
}

// --- runConfigShow tests ---

func TestRunConfigShow_Table(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err := runConfigShow(nil, nil)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runConfigShow error: %v", err)
	}
	if !containsSubstring(got, "board.name") {
		t.Errorf("expected board.name in output, got: %s", got)
	}
	if !containsSubstring(got, testBoardName) {
		t.Errorf("expected board name value, got: %s", got)
	}
}

func TestRunConfigShow_JSON(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err := runConfigShow(nil, nil)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runConfigShow error: %v", err)
	}
	if !containsSubstring(got, `"board.name"`) {
		t.Errorf("expected JSON key board.name, got: %s", got)
	}
}

// --- runConfigGet tests ---

func TestRunConfigGet_ValidKey(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err := runConfigGet(nil, []string{"board.name"})

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runConfigGet error: %v", err)
	}
	if !containsSubstring(got, testBoardName) {
		t.Errorf("expected 'TestBoard', got: %s", got)
	}
}

func TestRunConfigGet_InvalidKey(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigGet(nil, []string{"nonexistent.key"})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

// --- runConfigSet tests ---

func TestRunConfigSet_WritableKey(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err := runConfigSet(nil, []string{"board.name", "NewName"})

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runConfigSet error: %v", err)
	}
	if !containsSubstring(got, "NewName") {
		t.Errorf("expected new name in output, got: %s", got)
	}

	// Verify persisted.
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Board.Name != "NewName" {
		t.Errorf("board.name = %q, want %q", cfg.Board.Name, "NewName")
	}
}

func TestRunConfigSet_ReadOnlyKey(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigSet(nil, []string{"version", "99"})
	if err == nil {
		t.Fatal("expected error for read-only key")
	}
}

func TestRunConfigSet_InvalidKey(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigSet(nil, []string{"nonexistent", "value"})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
