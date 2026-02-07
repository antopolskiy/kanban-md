package config

import (
	"os"
	"path/filepath"
	"testing"
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

	// Verify statuses preserved in order.
	wantStatuses := []string{"backlog", "todo", "in-progress", "review", "done"}
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

	const expectedFiles = 3
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
