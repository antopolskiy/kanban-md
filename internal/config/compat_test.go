package config

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

// copyDir recursively copies src to dst for test isolation.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("reading dir %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o750); err != nil {
		t.Fatalf("creating dir %s: %v", dst, err)
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		data, readErr := os.ReadFile(srcPath) //nolint:gosec // test fixture path
		if readErr != nil {
			t.Fatalf("reading %s: %v", srcPath, readErr)
		}
		if writeErr := os.WriteFile(dstPath, data, fileMode); writeErr != nil {
			t.Fatalf("writing %s: %v", dstPath, writeErr)
		}
	}
}

func TestCompatV1Config(t *testing.T) {
	const wantName = "Test Project"
	const wantNextID = 4

	// Copy fixtures to temp dir so tests don't modify originals.
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v1")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v1 fixture: %v", err)
	}

	// Verify all config fields parsed correctly.
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.Board.Name != wantName {
		t.Errorf("Board.Name = %q, want %q", cfg.Board.Name, wantName)
	}
	if cfg.Board.Description != "A project for testing compatibility" {
		t.Errorf("Board.Description = %q, want %q", cfg.Board.Description, "A project for testing compatibility")
	}
	if cfg.TasksDir != "tasks" {
		t.Errorf("TasksDir = %q, want %q", cfg.TasksDir, "tasks")
	}
	if cfg.NextID != wantNextID {
		t.Errorf("NextID = %d, want %d", cfg.NextID, wantNextID)
	}

	// Verify statuses preserved in order (archived added by migration).
	wantStatuses := []string{"backlog", "todo", "in-progress", "review", "done", ArchivedStatus}
	if len(cfg.Statuses) != len(wantStatuses) {
		t.Fatalf("Statuses len = %d, want %d", len(cfg.Statuses), len(wantStatuses))
	}
	for i, s := range wantStatuses {
		if cfg.Statuses[i] != s {
			t.Errorf("Statuses[%d] = %q, want %q", i, cfg.Statuses[i], s)
		}
	}

	// Verify priorities preserved in order.
	wantPriorities := []string{"low", "medium", "high", "critical"}
	if len(cfg.Priorities) != len(wantPriorities) {
		t.Fatalf("Priorities len = %d, want %d", len(cfg.Priorities), len(wantPriorities))
	}
	for i, p := range wantPriorities {
		if cfg.Priorities[i] != p {
			t.Errorf("Priorities[%d] = %q, want %q", i, cfg.Priorities[i], p)
		}
	}

	// Verify defaults.
	if cfg.Defaults.Status != "backlog" {
		t.Errorf("Defaults.Status = %q, want %q", cfg.Defaults.Status, "backlog")
	}
	if cfg.Defaults.Priority != "medium" {
		t.Errorf("Defaults.Priority = %q, want %q", cfg.Defaults.Priority, "medium")
	}
}

func TestCompatV1ConfigMigratesToCurrentVersion(t *testing.T) {
	// v1 config should migrate through v1→v2→v3.
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v1")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v1 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d (after migration)", cfg.Version, CurrentVersion)
	}
	if cfg.WIPLimits != nil {
		t.Errorf("WIPLimits = %v, want nil (v1 has no WIP limits)", cfg.WIPLimits)
	}
	// WIPLimit helper should return 0 for any status.
	if limit := cfg.WIPLimit("in-progress"); limit != 0 {
		t.Errorf("WIPLimit(in-progress) = %d, want 0", limit)
	}
	// v2→v3 migration should also have run.
	if cfg.ClaimTimeout != DefaultClaimTimeout {
		t.Errorf("ClaimTimeout = %q, want %q (from v2→v3 migration)", cfg.ClaimTimeout, DefaultClaimTimeout)
	}
	if cfg.Defaults.Class != DefaultClass {
		t.Errorf("Defaults.Class = %q, want %q (from v2→v3 migration)", cfg.Defaults.Class, DefaultClass)
	}
	// v3→v4 migration should also have run.
	if cfg.TUI.TitleLines != DefaultTitleLines {
		t.Errorf("TUI.TitleLines = %d, want %d (from v3→v4 migration)", cfg.TUI.TitleLines, DefaultTitleLines)
	}
	// v4→v5 migration should also have run.
	if len(cfg.TUI.AgeThresholds) != len(DefaultAgeThresholds) {
		t.Errorf("AgeThresholds len = %d, want %d (from v4→v5 migration)", len(cfg.TUI.AgeThresholds), len(DefaultAgeThresholds))
	}
	// v5→v6 migration should also have run.
	if !contains(cfg.Statuses, ArchivedStatus) {
		t.Error("Statuses should contain 'archived' after v5→v6 migration")
	}
}

func TestCompatV2Config(t *testing.T) {
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v2")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v2 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.Board.Name != "Test Project v2" {
		t.Errorf("Board.Name = %q, want %q", cfg.Board.Name, "Test Project v2")
	}

	// Verify WIP limits.
	if cfg.WIPLimit("in-progress") != 3 {
		t.Errorf("WIPLimit(in-progress) = %d, want 3", cfg.WIPLimit("in-progress"))
	}
	if cfg.WIPLimit("review") != 2 {
		t.Errorf("WIPLimit(review) = %d, want 2", cfg.WIPLimit("review"))
	}
	if cfg.WIPLimit("backlog") != 0 {
		t.Errorf("WIPLimit(backlog) = %d, want 0 (unlimited)", cfg.WIPLimit("backlog"))
	}
}

func TestCompatV2ConfigMigratesToV3(t *testing.T) {
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v2")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v2 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d (after migration)", cfg.Version, CurrentVersion)
	}

	// Migration should set claim_timeout default.
	if cfg.ClaimTimeout != DefaultClaimTimeout {
		t.Errorf("ClaimTimeout = %q, want %q", cfg.ClaimTimeout, DefaultClaimTimeout)
	}

	// Migration should set default classes.
	const expectedClasses = 4
	if len(cfg.Classes) != expectedClasses {
		t.Fatalf("Classes len = %d, want %d", len(cfg.Classes), expectedClasses)
	}
	if cfg.Classes[0].Name != "expedite" {
		t.Errorf("Classes[0].Name = %q, want %q", cfg.Classes[0].Name, "expedite")
	}
	if cfg.Classes[0].WIPLimit != 1 {
		t.Errorf("Classes[0].WIPLimit = %d, want 1", cfg.Classes[0].WIPLimit)
	}
	if !cfg.Classes[0].BypassColumnWIP {
		t.Error("Classes[0].BypassColumnWIP = false, want true")
	}

	// Migration should set defaults.class.
	if cfg.Defaults.Class != DefaultClass {
		t.Errorf("Defaults.Class = %q, want %q", cfg.Defaults.Class, DefaultClass)
	}
}

func TestCompatV3Config(t *testing.T) {
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v3")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v3 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.Board.Name != "Test Project v3" {
		t.Errorf("Board.Name = %q, want %q", cfg.Board.Name, "Test Project v3")
	}
	// v3 config has no tui section; migration should set default.
	if cfg.TUI.TitleLines != DefaultTitleLines {
		t.Errorf("TUI.TitleLines = %d, want %d", cfg.TUI.TitleLines, DefaultTitleLines)
	}
}

func TestCompatV3ConfigMigratesToV4(t *testing.T) {
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v3")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v3 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d (after migration)", cfg.Version, CurrentVersion)
	}
	if cfg.TUI.TitleLines != DefaultTitleLines {
		t.Errorf("TUI.TitleLines = %d, want %d", cfg.TUI.TitleLines, DefaultTitleLines)
	}
	// Existing v3 fields should be preserved.
	if cfg.ClaimTimeout != "1h" {
		t.Errorf("ClaimTimeout = %q, want %q", cfg.ClaimTimeout, "1h")
	}
	const expectedClasses = 4
	if len(cfg.Classes) != expectedClasses {
		t.Errorf("Classes len = %d, want %d", len(cfg.Classes), expectedClasses)
	}
}

func TestMigrationPersistsToDisk(t *testing.T) {
	// Verify that after Load() migrates a v1 config, the on-disk file
	// is updated to the current version so re-migration is avoided.
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v1")
	copyDir(t, fixture, tmp)

	// First Load triggers migration from v1→v3.
	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v1 fixture: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d after migration", cfg.Version, CurrentVersion)
	}

	// Read the raw config file from disk and parse without migration
	// to verify that the persisted version is already current.
	data, err := os.ReadFile(filepath.Join(tmp, ConfigFileName)) //nolint:gosec // test temp path
	if err != nil {
		t.Fatalf("reading config file: %v", err)
	}

	var raw Config
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parsing raw config: %v", err)
	}

	if raw.Version != CurrentVersion {
		t.Errorf("On-disk config version = %d, want %d (migration should persist)", raw.Version, CurrentVersion)
	}
	if raw.ClaimTimeout != DefaultClaimTimeout {
		t.Errorf("Persisted ClaimTimeout = %q, want %q", raw.ClaimTimeout, DefaultClaimTimeout)
	}
	if raw.Defaults.Class != DefaultClass {
		t.Errorf("Persisted Defaults.Class = %q, want %q", raw.Defaults.Class, DefaultClass)
	}
	if raw.TUI.TitleLines != DefaultTitleLines {
		t.Errorf("Persisted TUI.TitleLines = %d, want %d", raw.TUI.TitleLines, DefaultTitleLines)
	}
	if len(raw.TUI.AgeThresholds) != len(DefaultAgeThresholds) {
		t.Errorf("Persisted AgeThresholds len = %d, want %d", len(raw.TUI.AgeThresholds), len(DefaultAgeThresholds))
	}
	if !contains(raw.Statuses, ArchivedStatus) {
		t.Error("Persisted Statuses should contain 'archived' after v5→v6 migration")
	}
	const wantName = "Test Project"
	if raw.Board.Name != wantName {
		t.Errorf("Persisted Board.Name = %q, want %q", raw.Board.Name, wantName)
	}
	const wantNextID = 4
	if raw.NextID != wantNextID {
		t.Errorf("Persisted NextID = %d, want %d", raw.NextID, wantNextID)
	}
}

func TestCompatV4Config(t *testing.T) {
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v4")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v4 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.Board.Name != "Test Project v4" {
		t.Errorf("Board.Name = %q, want %q", cfg.Board.Name, "Test Project v4")
	}
	if cfg.TUI.TitleLines != DefaultTitleLines {
		t.Errorf("TUI.TitleLines = %d, want %d", cfg.TUI.TitleLines, DefaultTitleLines)
	}
}

func TestCompatV4ConfigMigratesToV5(t *testing.T) {
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v4")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v4 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d (after migration)", cfg.Version, CurrentVersion)
	}
	// v4→v5 migration should set default age thresholds.
	if len(cfg.TUI.AgeThresholds) != len(DefaultAgeThresholds) {
		t.Fatalf("AgeThresholds len = %d, want %d", len(cfg.TUI.AgeThresholds), len(DefaultAgeThresholds))
	}
	if cfg.TUI.AgeThresholds[0].After != "0s" {
		t.Errorf("AgeThresholds[0].After = %q, want %q", cfg.TUI.AgeThresholds[0].After, "0s")
	}
	// Existing v4 fields should be preserved.
	if cfg.TUI.TitleLines != DefaultTitleLines {
		t.Errorf("TUI.TitleLines = %d, want %d", cfg.TUI.TitleLines, DefaultTitleLines)
	}
	if cfg.ClaimTimeout != "1h" {
		t.Errorf("ClaimTimeout = %q, want %q", cfg.ClaimTimeout, "1h")
	}
}

func TestCompatV5Config(t *testing.T) {
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v5")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v5 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.Board.Name != "Test Project v5" {
		t.Errorf("Board.Name = %q, want %q", cfg.Board.Name, "Test Project v5")
	}
	if len(cfg.TUI.AgeThresholds) != len(DefaultAgeThresholds) {
		t.Errorf("AgeThresholds len = %d, want %d", len(cfg.TUI.AgeThresholds), len(DefaultAgeThresholds))
	}
}

func TestCompatV5ConfigMigratesToV6(t *testing.T) {
	tmp := t.TempDir()
	fixture := filepath.Join("testdata", "compat", "v5")
	copyDir(t, fixture, tmp)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() v5 fixture: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d (after migration)", cfg.Version, CurrentVersion)
	}
	// v5→v6 migration should add "archived" to statuses.
	if !contains(cfg.Statuses, ArchivedStatus) {
		t.Fatal("Statuses should contain 'archived' after v5→v6 migration")
	}
	wantStatuses := []string{"backlog", "todo", "in-progress", "review", "done", ArchivedStatus}
	if len(cfg.Statuses) != len(wantStatuses) {
		t.Fatalf("Statuses len = %d, want %d", len(cfg.Statuses), len(wantStatuses))
	}
	for i, s := range wantStatuses {
		if cfg.Statuses[i] != s {
			t.Errorf("Statuses[%d] = %q, want %q", i, cfg.Statuses[i], s)
		}
	}
	// Existing v5 fields should be preserved.
	if cfg.TUI.TitleLines != DefaultTitleLines {
		t.Errorf("TUI.TitleLines = %d, want %d", cfg.TUI.TitleLines, DefaultTitleLines)
	}
	if cfg.ClaimTimeout != "1h" {
		t.Errorf("ClaimTimeout = %q, want %q", cfg.ClaimTimeout, "1h")
	}
}

func TestCompatV1TasksReadable(t *testing.T) {
	// This test verifies that the current task reader can parse v1 task files.
	// We only check that files exist and are well-formed here; detailed task
	// parsing is tested in internal/task. The goal is to catch regressions
	// if task file format changes break backward compatibility.
	fixture := filepath.Join("testdata", "compat", "v1", "tasks")
	entries, err := os.ReadDir(fixture)
	if err != nil {
		t.Fatalf("reading fixture dir: %v", err)
	}

	const expectedFiles = 6
	if len(entries) != expectedFiles {
		t.Fatalf("expected %d fixture task files, got %d", expectedFiles, len(entries))
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(fixture, e.Name())
		data, readErr := os.ReadFile(path) //nolint:gosec // test fixture path
		if readErr != nil {
			t.Errorf("reading %s: %v", e.Name(), readErr)
			continue
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", e.Name())
		}
	}
}
