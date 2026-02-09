package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
)

// --- parseWIPLimits tests ---

func TestParseWIPLimits_Valid(t *testing.T) {
	limits, err := parseWIPLimits([]string{"in-progress:3", "review:5"})
	if err != nil {
		t.Fatal(err)
	}
	if limits["in-progress"] != 3 {
		t.Errorf("in-progress = %d, want 3", limits["in-progress"])
	}
	if limits["review"] != 5 {
		t.Errorf("review = %d, want 5", limits["review"])
	}
}

func TestParseWIPLimits_InvalidFormat(t *testing.T) {
	_, err := parseWIPLimits([]string{"no-colon"})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestParseWIPLimits_InvalidNumber(t *testing.T) {
	_, err := parseWIPLimits([]string{"in-progress:abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric value")
	}
}

func TestParseWIPLimits_Empty(t *testing.T) {
	limits, err := parseWIPLimits([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(limits) != 0 {
		t.Errorf("expected empty map, got %v", limits)
	}
}

// --- runInit tests ---

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("name", "", "")
	cmd.Flags().StringSlice("statuses", nil, "")
	cmd.Flags().StringSlice("wip-limit", nil, "")
	return cmd
}

func TestRunInit_DefaultName(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newInitCmd()
	err := runInit(cmd, nil)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runInit error: %v", err)
	}
	if !containsSubstring(got, "Initialized board") {
		t.Errorf("expected init message, got: %s", got)
	}

	// Verify config file exists.
	if _, err := os.Stat(filepath.Join(kanbanDir, config.ConfigFileName)); err != nil {
		t.Errorf("config file should exist: %v", err)
	}

	// Verify tasks directory exists.
	if _, err := os.Stat(filepath.Join(kanbanDir, "tasks")); err != nil {
		t.Errorf("tasks directory should exist: %v", err)
	}
}

func TestRunInit_WithName(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newInitCmd()
	_ = cmd.Flags().Set("name", "MyProject")
	err := runInit(cmd, nil)

	_ = drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runInit error: %v", err)
	}

	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Board.Name != "MyProject" {
		t.Errorf("board name = %q, want %q", cfg.Board.Name, "MyProject")
	}
}

func TestRunInit_CustomStatuses(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newInitCmd()
	_ = cmd.Flags().Set("statuses", "open,closed")
	err := runInit(cmd, nil)

	_ = drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runInit error: %v", err)
	}

	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Statuses) != 2 || cfg.Statuses[0] != "open" || cfg.Statuses[1] != "closed" {
		t.Errorf("statuses = %v, want [open, closed]", cfg.Statuses)
	}
	if cfg.Defaults.Status != "open" {
		t.Errorf("default status = %q, want %q", cfg.Defaults.Status, "open")
	}
}

func TestRunInit_WithWIPLimits(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	cmd := newInitCmd()
	_ = cmd.Flags().Set("wip-limit", "in-progress:3")
	err := runInit(cmd, nil)

	_ = drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runInit error: %v", err)
	}

	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WIPLimits["in-progress"] != 3 {
		t.Errorf("wip_limits[in-progress] = %d, want 3", cfg.WIPLimits["in-progress"])
	}
}

func TestRunInit_AlreadyInitialized(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cmd := newInitCmd()
	err := runInit(cmd, nil)
	if err == nil {
		t.Fatal("expected error for already initialized board")
	}
}

func TestRunInit_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	cmd := newInitCmd()
	_ = cmd.Flags().Set("name", "TestJSON")
	err := runInit(cmd, nil)

	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runInit error: %v", err)
	}
	if !containsSubstring(got, `"status": "initialized"`) {
		t.Errorf("expected JSON with status:initialized, got: %s", got)
	}
}
