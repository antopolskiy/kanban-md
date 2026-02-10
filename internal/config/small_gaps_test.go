package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const osWindows = "windows"

// ---------------------------------------------------------------------------
// migrate — version < 1 (negative version)
// ---------------------------------------------------------------------------

func TestMigrate_NegativeVersion(t *testing.T) {
	cfg := &Config{Version: -1}
	err := migrate(cfg)
	if err == nil {
		t.Fatal("expected error for negative version")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

// ---------------------------------------------------------------------------
// migrate — version > CurrentVersion
// ---------------------------------------------------------------------------

func TestMigrate_FutureVersion(t *testing.T) {
	cfg := &Config{Version: CurrentVersion + 1}
	err := migrate(cfg)
	if err == nil {
		t.Fatal("expected error for future version")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
	if !containsStr(err.Error(), "newer than supported") {
		t.Errorf("error = %v, want to contain 'newer than supported'", err)
	}
}

// ---------------------------------------------------------------------------
// migrate — already at current version (no-op)
// ---------------------------------------------------------------------------

func TestMigrate_AlreadyAtCurrent(t *testing.T) {
	cfg := NewDefault("Test")
	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate(current version) error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Load — version > CurrentVersion edge case
// ---------------------------------------------------------------------------

func TestLoad_FutureVersion(t *testing.T) {
	dir := t.TempDir()
	cfg := NewDefault("Test")
	cfg.Version = CurrentVersion + 1
	cfg.SetDir(dir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for future config version")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

// ---------------------------------------------------------------------------
// IsTerminalStatus — config with no archived status
// ---------------------------------------------------------------------------

func TestIsTerminalStatus_NoArchivedStatus(t *testing.T) {
	cfg := &Config{
		Statuses: []StatusConfig{{Name: "backlog"}, {Name: "done"}},
	}
	// With no "archived" in the list, the last status ("done") should be terminal.
	if !cfg.IsTerminalStatus("done") {
		t.Error("last status should be terminal when no archived status exists")
	}
	if cfg.IsTerminalStatus("backlog") {
		t.Error("first status should not be terminal")
	}
}

func TestIsTerminalStatus_EmptyStatuses(t *testing.T) {
	cfg := &Config{}
	if cfg.IsTerminalStatus("anything") {
		t.Error("should return false for empty statuses list")
	}
}

func TestIsTerminalStatus_ArchivedNotInList(t *testing.T) {
	// "archived" is explicitly checked by name even if not in the list.
	cfg := &Config{
		Statuses: []StatusConfig{{Name: "backlog"}, {Name: "done"}},
	}
	if !cfg.IsTerminalStatus(ArchivedStatus) {
		t.Error("ArchivedStatus should always be terminal even if not in list")
	}
}

// ---------------------------------------------------------------------------
// FindDir — reaches root without finding a board
// ---------------------------------------------------------------------------

func TestFindDir_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindDir(dir)
	if err == nil {
		t.Fatal("expected error when no board found")
	}
}

// ---------------------------------------------------------------------------
// FindDir — symlink to a kanban directory
// ---------------------------------------------------------------------------

func TestFindDir_Symlink(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("symlinks require elevated privileges on Windows")
	}
	root := t.TempDir()
	kanbanDir := filepath.Join(root, "real-kanban")
	if _, err := Init(kanbanDir, "Symlink Test"); err != nil {
		t.Fatal(err)
	}

	// Create a symlink from root/kanban -> real-kanban.
	symlink := filepath.Join(root, DefaultDir)
	if err := os.Symlink(kanbanDir, symlink); err != nil {
		t.Fatal(err)
	}

	found, err := FindDir(root)
	if err != nil {
		t.Fatalf("FindDir(root with symlink) error: %v", err)
	}
	// Should resolve via the kanban/ default dir symlink.
	if found == "" {
		t.Error("FindDir should find board via symlink")
	}
}

// ---------------------------------------------------------------------------
// Init — Save error path (write to read-only directory)
// ---------------------------------------------------------------------------

func TestInit_SaveError(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("chmod does not restrict writes on Windows")
	}
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	// Initialize successfully first.
	if _, err := Init(kanbanDir, "Test"); err != nil {
		t.Fatal(err)
	}

	// Make the kanban dir read-only.
	if err := os.Chmod(kanbanDir, 0o444); err != nil { //nolint:gosec // intentionally restrict for test
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(kanbanDir, 0o750) }) //nolint:gosec // restore permissions in cleanup

	// A second init should fail when saving config.
	_, err := Init(kanbanDir, "Test2")
	if err == nil {
		t.Fatal("expected error when Save fails due to read-only directory")
	}
}

// ---------------------------------------------------------------------------
// Save — WriteFile error (nonexistent parent)
// ---------------------------------------------------------------------------

func TestSave_NonexistentParent(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.SetDir(filepath.Join(t.TempDir(), "does", "not", "exist"))

	err := cfg.Save()
	if err != nil {
		// Good — WriteFile returned an error for missing parent.
		return
	}
	t.Fatal("expected error when config directory does not exist")
}

// ---------------------------------------------------------------------------
// StatusShowDuration — explicit false, explicit true, unknown status
// ---------------------------------------------------------------------------

func TestStatusShowDuration_ExplicitFalse(t *testing.T) {
	hide := false
	cfg := &Config{
		Statuses: []StatusConfig{{Name: "backlog", ShowDuration: &hide}},
	}
	if cfg.StatusShowDuration("backlog") {
		t.Error("ShowDuration should be false when explicitly set to false")
	}
}

func TestStatusShowDuration_UnknownStatus(t *testing.T) {
	cfg := &Config{
		Statuses: []StatusConfig{{Name: "backlog"}},
	}
	// Unknown status should default to true.
	if !cfg.StatusShowDuration("nonexistent") {
		t.Error("ShowDuration should default to true for unknown status")
	}
}

// ---------------------------------------------------------------------------
// Validate edge cases — NextID < 1, empty TasksDir
// ---------------------------------------------------------------------------

func TestValidate_NextIDZero(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.NextID = 0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for NextID = 0")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

func TestValidate_EmptyTasksDir(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TasksDir = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty tasks_dir")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

// ---------------------------------------------------------------------------
// Validate — too few statuses
// ---------------------------------------------------------------------------

func TestValidate_TooFewStatuses(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Statuses = []StatusConfig{{Name: "only-one"}}
	cfg.Defaults.Status = "only-one"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for fewer than 2 statuses")
	}
}

// ---------------------------------------------------------------------------
// Validate — WIP limit references unknown status
// ---------------------------------------------------------------------------

func TestValidate_WIPLimitUnknownStatus(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.WIPLimits = map[string]int{"nonexistent": 5}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for WIP limit on unknown status")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

// ---------------------------------------------------------------------------
// TitleLines — zero returns default
// ---------------------------------------------------------------------------

func TestTitleLines_ZeroReturnsDefault(t *testing.T) {
	cfg := &Config{}
	got := cfg.TitleLines()
	if got != DefaultTitleLines {
		t.Errorf("TitleLines() = %d, want %d", got, DefaultTitleLines)
	}
}

// ---------------------------------------------------------------------------
// ClaimTimeoutDuration — unparseable string
// ---------------------------------------------------------------------------

func TestClaimTimeoutDuration_Unparseable(t *testing.T) {
	cfg := &Config{ClaimTimeout: "not-a-duration"}
	if d := cfg.ClaimTimeoutDuration(); d != 0 {
		t.Errorf("ClaimTimeoutDuration() = %v, want 0 for unparseable", d)
	}
}

// ---------------------------------------------------------------------------
// migrateV7ToV8 — edge case with single status
// ---------------------------------------------------------------------------

func TestMigrateV7ToV8_SingleStatus(t *testing.T) {
	const v7 = 7
	const v8 = 8
	cfg := &Config{
		Version:  v7,
		Statuses: []StatusConfig{{Name: "only"}},
	}
	// Should not panic.
	if err := migrateV7ToV8(cfg); err != nil {
		t.Fatalf("migrateV7ToV8 error: %v", err)
	}
	if cfg.Version != v8 {
		t.Errorf("Version = %d, want %d", cfg.Version, v8)
	}
}

func TestMigrateV7ToV8_EmptyStatuses(t *testing.T) {
	const v7 = 7
	const v8 = 8
	cfg := &Config{
		Version:  v7,
		Statuses: nil,
	}
	if err := migrateV7ToV8(cfg); err != nil {
		t.Fatalf("migrateV7ToV8 error: %v", err)
	}
	if cfg.Version != v8 {
		t.Errorf("Version = %d, want %d", cfg.Version, v8)
	}
}

// ---------------------------------------------------------------------------
// Load — migrated config save error
// ---------------------------------------------------------------------------

func TestLoad_MigrationSaveError(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("chmod does not restrict writes on Windows")
	}
	dir := t.TempDir()
	// Write a v7 config.
	const v7 = 7
	cfg := NewDefault("Test")
	cfg.Version = v7
	cfg.SetDir(dir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Make the config file read-only so Save after migration fails.
	configPath := filepath.Join(dir, ConfigFileName)
	if err := os.Chmod(configPath, 0o444); err != nil { //nolint:gosec // intentionally restrict for test
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(configPath, 0o600) })

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error when migrated config cannot be saved")
	}
	if !containsStr(err.Error(), "saving migrated config") {
		t.Errorf("error = %v, want to contain 'saving migrated config'", err)
	}
}
