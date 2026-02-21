package cmd

import (
	"errors"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
)

// --- runConfigGet extended tests ---

func TestRunConfigGet_JSON(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err := runConfigGet(nil, []string{"board.name"})

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runConfigGet JSON error: %v", err)
	}
	if !containsSubstring(got, testBoardName) {
		t.Errorf("expected board name in JSON output, got: %s", got)
	}
}

func TestRunConfigGet_NoConfig(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigGet(nil, []string{"board.name"})
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestRunConfigGet_InvalidKey_ErrorCode(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigGet(nil, []string{"does.not.exist"})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidInput {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidInput)
	}
}

// --- runConfigSet extended tests ---

func TestRunConfigSet_JSON(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err := runConfigSet(nil, []string{"board.name", "JSONBoard"})

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runConfigSet JSON error: %v", err)
	}
	if !containsSubstring(got, `"key"`) {
		t.Errorf("expected JSON key field, got: %s", got)
	}
	if !containsSubstring(got, "JSONBoard") {
		t.Errorf("expected new value in JSON output, got: %s", got)
	}
}

func TestRunConfigSet_InvalidValue(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigSet(nil, []string{"defaults.status", "nonexistent-status"})
	if err == nil {
		t.Fatal("expected error for invalid status value")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidInput {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidInput)
	}
}

func TestRunConfigSet_NoConfig(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigSet(nil, []string{"board.name", "test"})
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestRunConfigSet_ReadOnlyKey_ErrorCode(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigSet(nil, []string{"statuses", "a,b,c"})
	if err == nil {
		t.Fatal("expected error for read-only key")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidInput {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidInput)
	}
}

// --- runConfigShow extended tests ---

func TestRunConfigShow_NoConfig(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runConfigShow(nil, nil)
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

// --- accessor edge cases ---

func TestConfigAccessors_SetDefaultsClass_Empty(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")
	cfg.Defaults.Class = classExpedite

	if err := accessors["defaults.class"].set(cfg, ""); err != nil {
		t.Fatal(err)
	}
	if cfg.Defaults.Class != "" {
		t.Errorf("defaults.class = %q, want empty after clear", cfg.Defaults.Class)
	}
}

func TestConfigAccessors_SetBoardDescription(t *testing.T) {
	accessors := configAccessors()
	cfg := config.NewDefault("Test")

	if err := accessors["board.description"].set(cfg, "A test board"); err != nil {
		t.Fatal(err)
	}
	if cfg.Board.Description != "A test board" {
		t.Errorf("board.description = %q, want %q", cfg.Board.Description, "A test board")
	}
}

func TestConfigAccessors_GetReadOnlyValues(t *testing.T) {
	cfg := config.NewDefault("Test")
	accessors := configAccessors()

	tests := []struct {
		key  string
		want bool // just test that get doesn't panic
	}{
		{"statuses", true},
		{"priorities", true},
		{"tasks_dir", true},
		{"next_id", true},
		{"version", true},
		{"wip_limits", true},
		{"classes", true},
		{"tui.title_lines", true},
		{"tui.hide_empty_columns", true},
		{"tui.age_thresholds", true},
		{"claim_timeout", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			acc, ok := accessors[tt.key]
			if !ok {
				t.Fatalf("accessor for %q not found", tt.key)
			}
			val := acc.get(cfg)
			if val == nil {
				t.Errorf("get(%q) returned nil", tt.key)
			}
		})
	}
}
