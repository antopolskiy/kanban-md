package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefault(t *testing.T) {
	cfg := NewDefault("Test Project")
	if cfg.Board.Name != "Test Project" {
		t.Errorf("Board.Name = %q, want %q", cfg.Board.Name, "Test Project")
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if cfg.NextID != 1 {
		t.Errorf("NextID = %d, want 1", cfg.NextID)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{"valid default", func(_ *Config) {}, false},
		{"bad version", func(c *Config) { c.Version = 99 }, true},
		{"empty name", func(c *Config) { c.Board.Name = "" }, true},
		{"empty tasks_dir", func(c *Config) { c.TasksDir = "" }, true},
		{"one status", func(c *Config) { c.Statuses = []string{"only"} }, true},
		{"duplicate statuses", func(c *Config) { c.Statuses = []string{"a", "a"} }, true},
		{"no priorities", func(c *Config) { c.Priorities = nil }, true},
		{"bad default status", func(c *Config) { c.Defaults.Status = "nonexistent" }, true},
		{"bad default priority", func(c *Config) { c.Defaults.Priority = "nonexistent" }, true},
		{"zero next_id", func(c *Config) { c.NextID = 0 }, true},
		{"valid wip_limits", func(c *Config) { c.WIPLimits = map[string]int{"in-progress": 3} }, false},
		{"wip unknown status", func(c *Config) { c.WIPLimits = map[string]int{"bogus": 3} }, true},
		{"wip negative limit", func(c *Config) { c.WIPLimits = map[string]int{"in-progress": -1} }, true},
		{"wip zero limit", func(c *Config) { c.WIPLimits = map[string]int{"in-progress": 0} }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefault("Test")
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrInvalid) {
				// version and other checks should wrap ErrInvalid
				if tt.name != "valid default" {
					t.Logf("error: %v", err)
				}
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := NewDefault("Test Project")
	cfg.SetDir(dir)
	cfg.NextID = 5

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Board.Name != "Test Project" {
		t.Errorf("loaded Board.Name = %q, want %q", loaded.Board.Name, "Test Project")
	}
	if loaded.NextID != 5 {
		t.Errorf("loaded NextID = %d, want 5", loaded.NextID)
	}
	if loaded.Dir() != dir {
		t.Errorf("loaded Dir() = %q, want %q", loaded.Dir(), dir)
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load() error = %v, want ErrNotFound", err)
	}
}

func TestFindDir(t *testing.T) {
	// Create: tmpdir/kanban/config.yml
	root := t.TempDir()
	kanbanDir := filepath.Join(root, DefaultDir)
	if err := os.MkdirAll(kanbanDir, 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := NewDefault("Test")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Find from root.
	found, err := FindDir(root)
	if err != nil {
		t.Fatalf("FindDir(%q) error: %v", root, err)
	}
	if found != kanbanDir {
		t.Errorf("FindDir() = %q, want %q", found, kanbanDir)
	}

	// Find from a subdirectory.
	subDir := filepath.Join(root, "some", "deep", "path")
	if mkErr := os.MkdirAll(subDir, 0o750); mkErr != nil {
		t.Fatal(mkErr)
	}
	found, err = FindDir(subDir)
	if err != nil {
		t.Fatalf("FindDir(%q) error: %v", subDir, err)
	}
	if found != kanbanDir {
		t.Errorf("FindDir() = %q, want %q", found, kanbanDir)
	}
}

func TestFindDirNotFound(t *testing.T) {
	_, err := FindDir(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("FindDir() error = %v, want ErrNotFound", err)
	}
}

func TestStatusIndex(t *testing.T) {
	cfg := NewDefault("Test")
	if idx := cfg.StatusIndex("in-progress"); idx != 2 {
		t.Errorf("StatusIndex('in-progress') = %d, want 2", idx)
	}
	if idx := cfg.StatusIndex("nonexistent"); idx != -1 {
		t.Errorf("StatusIndex('nonexistent') = %d, want -1", idx)
	}
}

func TestIsTerminalStatus(t *testing.T) {
	cfg := NewDefault("Test")
	// Default statuses: [backlog, todo, in-progress, review, done]
	if !cfg.IsTerminalStatus("done") {
		t.Error("IsTerminalStatus('done') = false, want true")
	}
	if cfg.IsTerminalStatus("backlog") {
		t.Error("IsTerminalStatus('backlog') = true, want false")
	}
	if cfg.IsTerminalStatus("nonexistent") {
		t.Error("IsTerminalStatus('nonexistent') = true, want false")
	}
}

func TestIsTerminalStatusEmptyStatuses(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Statuses = nil
	if cfg.IsTerminalStatus("done") {
		t.Error("IsTerminalStatus with empty statuses should return false")
	}
}

func TestPriorityIndex(t *testing.T) {
	cfg := NewDefault("Test")
	if idx := cfg.PriorityIndex("high"); idx != 2 {
		t.Errorf("PriorityIndex('high') = %d, want 2", idx)
	}
	if idx := cfg.PriorityIndex("nonexistent"); idx != -1 {
		t.Errorf("PriorityIndex('nonexistent') = %d, want -1", idx)
	}
}
