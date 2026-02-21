package config

import (
	"errors"
	"testing"
)

func TestMigrateCurrentVersionNoop(t *testing.T) {
	cfg := NewDefault("Test")
	if err := migrate(cfg); err != nil {
		t.Errorf("migrate() current version: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
}

func TestMigrateNewerVersionErrors(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = CurrentVersion + 1

	err := migrate(cfg)
	if err == nil {
		t.Fatal("migrate() newer version: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("migrate() error = %v, want ErrInvalid", err)
	}
}

func TestMigrateZeroVersionErrors(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 0

	err := migrate(cfg)
	if err == nil {
		t.Fatal("migrate() version 0: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("migrate() error = %v, want ErrInvalid", err)
	}
}

func TestMigrateNegativeVersionErrors(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = -1

	err := migrate(cfg)
	if err == nil {
		t.Fatal("migrate() negative version: expected error, got nil")
	}
}

func TestMigrateV1ToCurrentVersion(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 1
	// Clear fields that v2→v3 migration should set.
	cfg.ClaimTimeout = ""
	cfg.Classes = nil
	cfg.Defaults.Class = ""
	cfg.TUI = TUIConfig{} // Clear all TUI fields including AgeThresholds.

	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate() v1→current: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
}

func TestMigrateV2ToV3(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 2
	// Clear fields that v2→v3 migration should set.
	cfg.ClaimTimeout = ""
	cfg.Classes = nil
	cfg.Defaults.Class = ""

	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate() v2→v3: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.ClaimTimeout != DefaultClaimTimeout {
		t.Errorf("ClaimTimeout = %q, want %q", cfg.ClaimTimeout, DefaultClaimTimeout)
	}
	const expectedClasses = 4
	if len(cfg.Classes) != expectedClasses {
		t.Errorf("Classes len = %d, want %d", len(cfg.Classes), expectedClasses)
	}
	if cfg.Defaults.Class != DefaultClass {
		t.Errorf("Defaults.Class = %q, want %q", cfg.Defaults.Class, DefaultClass)
	}
}

func TestMigrateV3ToV4(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 3
	cfg.TUI = TUIConfig{} // Clear to test migration sets default.

	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate() v3→v4: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.TUI.TitleLines != DefaultTitleLines {
		t.Errorf("TUI.TitleLines = %d, want %d", cfg.TUI.TitleLines, DefaultTitleLines)
	}
}

func TestMigrateV4ToV5(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 4
	cfg.TUI.AgeThresholds = nil // Clear to test migration sets default.

	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate() v4→v5: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if len(cfg.TUI.AgeThresholds) != len(DefaultAgeThresholds) {
		t.Fatalf("AgeThresholds len = %d, want %d", len(cfg.TUI.AgeThresholds), len(DefaultAgeThresholds))
	}
	if cfg.TUI.AgeThresholds[0].After != "0s" {
		t.Errorf("AgeThresholds[0].After = %q, want %q", cfg.TUI.AgeThresholds[0].After, "0s")
	}
}

func TestMigrateV5ToV6(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 5
	// Remove archived status to simulate a v5 config.
	cfg.Statuses = []StatusConfig{
		{Name: "backlog"},
		{Name: "todo"},
		{Name: "in-progress"},
		{Name: "review"},
		{Name: "done"},
	}

	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate() v5→v6: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	names := cfg.StatusNames()
	if !contains(names, ArchivedStatus) {
		t.Fatal("Statuses should contain 'archived' after v5→v6 migration")
	}
	// Verify archived is at the end.
	if names[len(names)-1] != ArchivedStatus {
		t.Errorf("last status = %q, want %q", names[len(names)-1], ArchivedStatus)
	}
}

func TestMigrateV5ToV6Idempotent(t *testing.T) {
	// If archived already exists, migration should not add a duplicate.
	cfg := NewDefault("Test")
	cfg.Version = 5
	cfg.Statuses = []StatusConfig{
		{Name: "backlog"},
		{Name: "todo"},
		{Name: "in-progress"},
		{Name: "review"},
		{Name: "done"},
		{Name: ArchivedStatus},
	}

	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate() v5→v6 idempotent: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	// Count occurrences of "archived".
	count := 0
	for _, s := range cfg.StatusNames() {
		if s == ArchivedStatus {
			count++
		}
	}
	if count != 1 {
		t.Errorf("archived appears %d times, want 1", count)
	}
}

func TestMigrateV6ToV7(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 6

	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate() v6→v7: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	// Statuses should be preserved.
	names := cfg.StatusNames()
	if len(names) != len(DefaultStatuses) {
		t.Errorf("Statuses len = %d, want %d", len(names), len(DefaultStatuses))
	}
}

func TestMigrateV9ToV10(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 9
	cfg.TUI.HideEmptyColumns = false

	if err := migrate(cfg); err != nil {
		t.Fatalf("migrate() v9→v10: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.TUI.HideEmptyColumns {
		t.Error("HideEmptyColumns should remain false by default after migration")
	}
}
