package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Load error paths ---

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte("key: [invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if want := "parsing config"; !containsStr(err.Error(), want) {
		t.Errorf("error = %v, want to contain %q", err, want)
	}
}

func TestLoad_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte("version: 1"), 0o000); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
	if want := "reading config"; !containsStr(err.Error(), want) {
		t.Errorf("error = %v, want to contain %q", err, want)
	}
}

func TestLoad_ValidationErrorAfterMigration(t *testing.T) {
	dir := t.TempDir()
	// Write a config at current version with an invalid board name.
	cfg := NewDefault("")
	cfg.SetDir(dir)
	// Save bypasses Validate, so we can write an invalid config.
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

func TestLoad_MigrationSavesConfig(t *testing.T) {
	dir := t.TempDir()
	// Write a v7 config that will be migrated to v8.
	cfg := NewDefault("Migrate Test")
	cfg.Version = 7
	cfg.SetDir(dir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", loaded.Version, CurrentVersion)
	}

	// Reload to verify migration was persisted.
	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("second Load() error: %v", err)
	}
	if reloaded.Version != CurrentVersion {
		t.Errorf("reloaded Version = %d, want %d", reloaded.Version, CurrentVersion)
	}
}

func TestLoad_MigrationError(t *testing.T) {
	dir := t.TempDir()
	// Write a config with version 0 (invalid, triggers migrate error).
	if err := os.WriteFile(
		filepath.Join(dir, ConfigFileName),
		[]byte("version: 0\nboard:\n  name: Test\ntasks_dir: tasks\nstatuses:\n  - backlog\n  - done\npriorities:\n  - medium\ndefaults:\n  status: backlog\n  priority: medium\nnext_id: 1\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected migration error for version 0")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

// --- Init error paths ---

func TestInit_MkdirError(t *testing.T) {
	dir := t.TempDir()
	// Create a file at the path where Init needs to create a directory.
	blockingFile := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Try to init inside the file (mkdir will fail).
	badPath := filepath.Join(blockingFile, "kanban")

	_, err := Init(badPath, "Test")
	if err == nil {
		t.Fatal("expected error when mkdir fails")
	}
	if want := "creating tasks directory"; !containsStr(err.Error(), want) {
		t.Errorf("error = %v, want to contain %q", err, want)
	}
}

// --- Save error path ---

func TestSave_WriteError(t *testing.T) {
	cfg := NewDefault("Test")
	// Set dir to a path that doesn't exist.
	cfg.SetDir(filepath.Join(t.TempDir(), "nonexistent", "path"))

	err := cfg.Save()
	if err == nil {
		t.Fatal("expected error when write fails")
	}
}

// --- StatusRequiresClaim ---

func TestStatusRequiresClaim_UnknownStatus(t *testing.T) {
	cfg := NewDefault("Test")
	if cfg.StatusRequiresClaim("bogus-status") {
		t.Error("StatusRequiresClaim('bogus-status') = true, want false")
	}
}

func TestStatusRequiresClaim_RequiredStatus(t *testing.T) {
	cfg := NewDefault("Test")
	// Default config has require_claim: true on in-progress.
	if !cfg.StatusRequiresClaim("in-progress") {
		t.Error("StatusRequiresClaim('in-progress') = false, want true")
	}
}

func TestStatusRequiresClaim_OptionalStatus(t *testing.T) {
	cfg := NewDefault("Test")
	// Default config has require_claim: false on backlog.
	if cfg.StatusRequiresClaim("backlog") {
		t.Error("StatusRequiresClaim('backlog') = true, want false")
	}
}

// --- validateTUI error paths ---

func TestValidateTUI_InvalidDuration(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TUI.AgeThresholds = []AgeThreshold{
		{After: "not-a-duration", Color: "196"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid age threshold duration")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
	if want := "age_thresholds[0].after"; !containsStr(err.Error(), want) {
		t.Errorf("error = %v, want to contain %q", err, want)
	}
}

func TestValidateTUI_EmptyColor(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TUI.AgeThresholds = []AgeThreshold{
		{After: "1h", Color: ""},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty color")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
	if want := "color is required"; !containsStr(err.Error(), want) {
		t.Errorf("error = %v, want to contain %q", err, want)
	}
}

func TestValidateTUI_MultipleThresholdsSecondInvalid(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TUI.AgeThresholds = []AgeThreshold{
		{After: "1h", Color: "34"},
		{After: "bad", Color: "196"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid second threshold")
	}
	if want := "age_thresholds[1].after"; !containsStr(err.Error(), want) {
		t.Errorf("error = %v, want to contain %q", err, want)
	}
}

// --- AgeThresholdsDuration ---

func TestAgeThresholdsDuration_SkipsInvalidDuration(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TUI.AgeThresholds = []AgeThreshold{
		{After: "1h", Color: "34"},
		{After: "not-a-duration", Color: "196"},
		{After: "24h", Color: "226"},
	}

	result := cfg.AgeThresholdsDuration()
	if len(result) != 2 {
		t.Fatalf("got %d thresholds, want 2 (invalid should be skipped)", len(result))
	}
	if result[0].After != time.Hour {
		t.Errorf("result[0].After = %v, want 1h", result[0].After)
	}
	if result[1].After != 24*time.Hour {
		t.Errorf("result[1].After = %v, want 24h", result[1].After)
	}
}

func TestAgeThresholdsDuration_EmptyUsesDefaults(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TUI.AgeThresholds = nil

	result := cfg.AgeThresholdsDuration()
	if len(result) != len(DefaultAgeThresholds) {
		t.Fatalf("got %d thresholds, want %d (defaults)", len(result), len(DefaultAgeThresholds))
	}
}

// --- Validate additional edge cases ---

func TestValidate_DuplicatePriorities(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Priorities = []string{"high", "high"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate priorities")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

// --- FindDir from inside kanban directory ---

func TestFindDir_FromInsideKanbanDir(t *testing.T) {
	root := t.TempDir()
	kanbanDir := filepath.Join(root, DefaultDir)

	cfg, err := Init(kanbanDir, "Test")
	if err != nil {
		t.Fatal(err)
	}

	// FindDir from inside the kanban directory itself (where config.yml lives).
	found, err := FindDir(cfg.Dir())
	if err != nil {
		t.Fatalf("FindDir(kanban dir) error: %v", err)
	}
	if found != cfg.Dir() {
		t.Errorf("FindDir() = %q, want %q", found, cfg.Dir())
	}
}

func TestFindDir_FromTasksSubdir(t *testing.T) {
	root := t.TempDir()
	kanbanDir := filepath.Join(root, DefaultDir)

	cfg, err := Init(kanbanDir, "Test")
	if err != nil {
		t.Fatal(err)
	}

	// FindDir from the tasks subdirectory (walks up).
	found, err := FindDir(cfg.TasksPath())
	if err != nil {
		t.Fatalf("FindDir(tasks dir) error: %v", err)
	}
	if found != cfg.Dir() {
		t.Errorf("FindDir() = %q, want %q", found, cfg.Dir())
	}
}

// --- IsArchivedStatus ---

func TestIsArchivedStatus_True(t *testing.T) {
	cfg := NewDefault("Test")
	if !cfg.IsArchivedStatus(ArchivedStatus) {
		t.Error("IsArchivedStatus('archived') = false, want true")
	}
}

func TestIsArchivedStatus_False(t *testing.T) {
	cfg := NewDefault("Test")
	if cfg.IsArchivedStatus("done") {
		t.Error("IsArchivedStatus('done') = true, want false")
	}
}

func TestIsArchivedStatus_NoArchivedColumn(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Statuses = []StatusConfig{{Name: "backlog"}, {Name: "done"}}
	if cfg.IsArchivedStatus(ArchivedStatus) {
		t.Error("IsArchivedStatus with no archived column = true, want false")
	}
}

// --- BoardStatuses ---

func TestBoardStatuses_ExcludesArchived(t *testing.T) {
	cfg := NewDefault("Test")
	statuses := cfg.BoardStatuses()
	for _, s := range statuses {
		if s == ArchivedStatus {
			t.Error("BoardStatuses() should not contain 'archived'")
		}
	}
	// Should have all statuses except archived.
	if len(statuses) != len(cfg.Statuses)-1 {
		t.Errorf("BoardStatuses() len = %d, want %d", len(statuses), len(cfg.Statuses)-1)
	}
}

// --- ActiveStatuses ---

func TestActiveStatuses_ExcludesTerminal(t *testing.T) {
	cfg := NewDefault("Test")
	active := cfg.ActiveStatuses()
	for _, s := range active {
		if cfg.IsTerminalStatus(s) {
			t.Errorf("ActiveStatuses() contains terminal status %q", s)
		}
	}
}

// --- StatusConfig UnmarshalYAML ---

func TestStatusConfig_UnmarshalYAML_Scalar(t *testing.T) {
	dir := t.TempDir()
	// Write a config with old-style string statuses (v7 format uses StatusConfig
	// but old v6 YAML may have plain strings).
	yamlContent := `version: 8
board:
  name: Test
tasks_dir: tasks
statuses:
  - backlog
  - done
priorities:
  - medium
defaults:
  status: backlog
  priority: medium
tui:
  title_lines: 1
  age_thresholds:
    - after: "0s"
      color: "242"
next_id: 1
`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(yamlContent), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Statuses[0].Name != "backlog" {
		t.Errorf("Statuses[0].Name = %q, want 'backlog'", loaded.Statuses[0].Name)
	}
}

// containsStr is a simple substring check for error messages.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
