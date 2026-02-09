package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const nonexistentName = "nonexistent"

// --- TasksPath tests ---

func TestTasksPath(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.SetDir("/tmp/kanban")

	got := cfg.TasksPath()
	want := filepath.Join("/tmp/kanban", DefaultTasksDir)
	if got != want {
		t.Errorf("TasksPath() = %q, want %q", got, want)
	}
}

func TestTasksPath_CustomDir(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.SetDir("/tmp/kanban")
	cfg.TasksDir = "cards"

	got := cfg.TasksPath()
	want := "/tmp/kanban/cards"
	if got != want {
		t.Errorf("TasksPath() = %q, want %q", got, want)
	}
}

// --- ClaimTimeoutDuration tests ---

func TestClaimTimeoutDuration_Default(t *testing.T) {
	cfg := NewDefault("Test")

	got := cfg.ClaimTimeoutDuration()
	want := time.Hour
	if got != want {
		t.Errorf("ClaimTimeoutDuration() = %v, want %v", got, want)
	}
}

func TestClaimTimeoutDuration_Custom(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.ClaimTimeout = "30m"

	got := cfg.ClaimTimeoutDuration()
	want := 30 * time.Minute
	if got != want {
		t.Errorf("ClaimTimeoutDuration() = %v, want %v", got, want)
	}
}

func TestClaimTimeoutDuration_Empty(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.ClaimTimeout = ""

	got := cfg.ClaimTimeoutDuration()
	if got != 0 {
		t.Errorf("ClaimTimeoutDuration() = %v, want 0", got)
	}
}

func TestClaimTimeoutDuration_Invalid(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.ClaimTimeout = "not-a-duration"

	got := cfg.ClaimTimeoutDuration()
	if got != 0 {
		t.Errorf("ClaimTimeoutDuration() = %v, want 0 for invalid duration", got)
	}
}

// --- ClassByName tests ---

func TestClassByName_Found(t *testing.T) {
	cfg := NewDefault("Test")

	cl := cfg.ClassByName("expedite")
	if cl == nil {
		t.Fatal("ClassByName('expedite') = nil, want non-nil")
	}
	if cl.Name != "expedite" {
		t.Errorf("class.Name = %q, want %q", cl.Name, "expedite")
	}
	if cl.WIPLimit != 1 {
		t.Errorf("class.WIPLimit = %d, want 1", cl.WIPLimit)
	}
	if !cl.BypassColumnWIP {
		t.Error("class.BypassColumnWIP = false, want true")
	}
}

func TestClassByName_Standard(t *testing.T) {
	cfg := NewDefault("Test")

	cl := cfg.ClassByName("standard")
	if cl == nil {
		t.Fatal("ClassByName('standard') = nil, want non-nil")
	}
	if cl.WIPLimit != 0 {
		t.Errorf("class.WIPLimit = %d, want 0", cl.WIPLimit)
	}
	if cl.BypassColumnWIP {
		t.Error("class.BypassColumnWIP = true, want false")
	}
}

func TestClassByName_NotFound(t *testing.T) {
	cfg := NewDefault("Test")

	cl := cfg.ClassByName(nonexistentName)
	if cl != nil {
		t.Errorf("ClassByName('nonexistent') = %v, want nil", cl)
	}
}

func TestClassByName_EmptyClasses(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Classes = nil

	cl := cfg.ClassByName("standard")
	if cl != nil {
		t.Errorf("ClassByName with nil classes = %v, want nil", cl)
	}
}

// --- ClassNames tests ---

func TestClassNames_Default(t *testing.T) {
	cfg := NewDefault("Test")

	names := cfg.ClassNames()
	want := [4]string{"expedite", "fixed-date", "standard", "intangible"}
	if len(names) != len(want) {
		t.Fatalf("ClassNames() len = %d, want %d", len(names), len(want))
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("ClassNames()[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestClassNames_Empty(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Classes = nil

	names := cfg.ClassNames()
	if len(names) != 0 {
		t.Errorf("ClassNames() = %v, want empty", names)
	}
}

// --- ClassIndex tests ---

func TestClassIndex_Found(t *testing.T) {
	cfg := NewDefault("Test")

	tests := [4][2]string{
		{"expedite", "0"},
		{"fixed-date", "1"},
		{"standard", "2"},
		{"intangible", "3"},
	}
	for _, tt := range tests {
		idx := cfg.ClassIndex(tt[0])
		wantStr := tt[1]
		want := int(wantStr[0] - '0')
		if idx != want {
			t.Errorf("ClassIndex(%q) = %d, want %d", tt[0], idx, want)
		}
	}
}

func TestClassIndex_NotFound(t *testing.T) {
	cfg := NewDefault("Test")

	idx := cfg.ClassIndex(nonexistentName)
	if idx != -1 {
		t.Errorf("ClassIndex('nonexistent') = %d, want -1", idx)
	}
}

func TestClassIndex_EmptyClasses(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Classes = nil

	idx := cfg.ClassIndex("standard")
	if idx != -1 {
		t.Errorf("ClassIndex with nil classes = %d, want -1", idx)
	}
}

// --- Init tests ---

func TestInit_CreatesStructure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kanban")

	cfg, err := Init(dir, "My Board")
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Check config values.
	if cfg.Board.Name != "My Board" {
		t.Errorf("Board.Name = %q, want %q", cfg.Board.Name, "My Board")
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.NextID != 1 {
		t.Errorf("NextID = %d, want 1", cfg.NextID)
	}

	// Check tasks directory was created.
	tasksPath := cfg.TasksPath()
	info, err := os.Stat(tasksPath)
	if err != nil {
		t.Fatalf("tasks directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("tasks path is not a directory")
	}

	// Check config file was written.
	configPath := cfg.ConfigPath()
	if _, statErr := os.Stat(configPath); statErr != nil {
		t.Fatalf("config file not created: %v", statErr)
	}

	// Load and verify it round-trips.
	loaded, err := Load(cfg.Dir())
	if err != nil {
		t.Fatalf("Load() after Init error: %v", err)
	}
	if loaded.Board.Name != "My Board" {
		t.Errorf("loaded Board.Name = %q, want %q", loaded.Board.Name, "My Board")
	}
}

func TestInit_SetsAbsoluteDir(t *testing.T) {
	tmpDir := t.TempDir()
	relPath := filepath.Join(tmpDir, "rel-board")

	cfg, err := Init(relPath, "Test")
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	if !filepath.IsAbs(cfg.Dir()) {
		t.Errorf("Dir() = %q, want absolute path", cfg.Dir())
	}
}

// --- Validate class-related tests ---

func TestValidateClasses_EmptyClassName(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Classes = []ClassConfig{{Name: ""}}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty class name")
	}
}

func TestValidateClasses_DuplicateClassName(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Classes = []ClassConfig{
		{Name: "standard"},
		{Name: "standard"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate class name")
	}
}

func TestValidateClasses_NegativeWIPLimit(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Classes = []ClassConfig{
		{Name: "standard", WIPLimit: -1},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative class WIP limit")
	}
}

func TestValidateClasses_BadDefaultClass(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Defaults.Class = nonexistentName

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for default class not in classes list")
	}
}

func TestValidateClasses_NoClasses(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Classes = nil
	cfg.Defaults.Class = ""

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() with no classes should pass, got: %v", err)
	}
}

// --- Validate claim timeout tests ---

func TestValidateClaimTimeout_Valid(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.ClaimTimeout = "2h30m"

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error: %v", err)
	}
}

func TestValidateClaimTimeout_Invalid(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.ClaimTimeout = "not-a-duration"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid claim timeout")
	}
}

func TestValidateClaimTimeout_Empty(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.ClaimTimeout = ""

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() with empty claim timeout should pass, got: %v", err)
	}
}

// --- WIPLimit tests ---

func TestWIPLimit_NilMap(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.WIPLimits = nil

	got := cfg.WIPLimit("in-progress")
	if got != 0 {
		t.Errorf("WIPLimit() = %d, want 0 for nil map", got)
	}
}

func TestWIPLimit_ExistingKey(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.WIPLimits = map[string]int{"in-progress": 3}

	got := cfg.WIPLimit("in-progress")
	if got != 3 {
		t.Errorf("WIPLimit('in-progress') = %d, want 3", got)
	}
}

func TestWIPLimit_MissingKey(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.WIPLimits = map[string]int{"in-progress": 3}

	got := cfg.WIPLimit("backlog")
	if got != 0 {
		t.Errorf("WIPLimit('backlog') = %d, want 0", got)
	}
}

// --- TitleLines tests ---

func TestTitleLines_Zero(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TUI.TitleLines = 0

	got := cfg.TitleLines()
	if got != DefaultTitleLines {
		t.Errorf("TitleLines() = %d, want %d for zero value", got, DefaultTitleLines)
	}
}

func TestTitleLines_Set(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TUI.TitleLines = 3

	got := cfg.TitleLines()
	if got != 3 {
		t.Errorf("TitleLines() = %d, want 3", got)
	}
}

// --- IndexOf tests ---

func TestIndexOf_Found(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if idx := IndexOf(slice, "b"); idx != 1 {
		t.Errorf("IndexOf = %d, want 1", idx)
	}
}

func TestIndexOf_NotFound(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if idx := IndexOf(slice, "d"); idx != -1 {
		t.Errorf("IndexOf = %d, want -1", idx)
	}
}

func TestIndexOf_Empty(t *testing.T) {
	if idx := IndexOf(nil, "a"); idx != -1 {
		t.Errorf("IndexOf(nil) = %d, want -1", idx)
	}
}
