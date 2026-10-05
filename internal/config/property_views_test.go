package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestPropertyViewConfigKindsAndValidation(t *testing.T) {
	for _, extra := range []string{
		"display: {compact_fields: []}", "display: {compact_fields: null}", "tui: {card_fields: []}", "tui: {card_fields: null}",
		"display: {compact_fields: [priority, priority]}", "display: {compact_fields: [property:status]}", "children: {detail_sort: priority}",
		"group_orders: {property:type: [1, 1.0]}", "group_orders: {priority: [high]}", "group_orders: {property:type: [[nested]]}",
	} {
		t.Run(extra, func(t *testing.T) {
			cfg := NewDefault("test")
			if err := yaml.Unmarshal([]byte(extra), cfg); err == nil {
				if err = cfg.Validate(); err == nil {
					t.Fatal("invalid setting accepted")
				}
			}
		})
	}
	cfg := NewDefault("test")
	if err := yaml.Unmarshal([]byte("group_orders:\n  property:type: [milestone, null, 0x10, \"16\"]\ndisplay: {compact_fields: [status, property:type]}\ntui: {card_fields: [property:type]}\nchildren: {detail_sort: property:reading_order}\n"), cfg); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var again Config
	if err := yaml.Unmarshal(data, &again); err != nil {
		t.Fatal(err)
	}
	for i, value := range cfg.GroupOrders["property:type"] {
		if !value.Equal(again.GroupOrders["property:type"][i]) {
			t.Fatalf("group scalar changed: %s", value.JSONLiteral())
		}
	}
}

func TestPropertyConfigMalformedRoot(t *testing.T) {
	for _, data := range []string{"- value\n", "- first\n- second\n", "value\n", "tui: *unknown\n"} {
		t.Run(data, func(t *testing.T) {
			var cfg Config
			if err := yaml.Unmarshal([]byte(data), &cfg); err == nil {
				t.Fatal("malformed config did not return a decode error")
			}
		})
	}
}

func TestCompatV11PropertyDefaultsAndHistoricalFixtures(t *testing.T) {
	for _, version := range []string{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9", "v10", "v11"} {
		t.Run(version, func(t *testing.T) {
			tmp := t.TempDir()
			copyDir(t, filepath.Join("testdata", "compat", version), tmp)
			cfg, err := Load(tmp)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Version != 12 || !slices.Equal(cfg.CompactFields(), []string{"status", "priority"}) || !slices.Equal(cfg.CardFields(), []string{"priority"}) || len(cfg.GroupOrders) != 0 || cfg.Children.DetailSort != "" {
				t.Fatalf("migration changed defaults: %#v", cfg)
			}
		})
	}
}

func TestCompatV11PropertyMigrationPreservesSettingsAndTaskBytes(t *testing.T) {
	tmp := t.TempDir()
	copyDir(t, filepath.Join("testdata", "compat", "v11"), tmp)
	path := filepath.Join(tmp, "tasks", "001-sample-task.md")
	before, err := os.ReadFile(path) //nolint:gosec // isolated compatibility fixture
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TUI.NarrowThreshold != 86 || cfg.TitleLines() != 3 || !cfg.TUI.HideEmptyColumns || cfg.ClaimTimeout != "2h" || len(cfg.TUI.AgeThresholds) != 2 {
		t.Fatal("old view settings lost")
	}
	after, err := os.ReadFile(path) //nolint:gosec // isolated compatibility fixture
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("config migration rewrote task")
	}
	if again, err := Load(tmp); err != nil || again.Version != 12 {
		t.Fatalf("reload: %v", err)
	}
}
