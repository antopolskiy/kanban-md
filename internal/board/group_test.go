package board

import (
	"testing"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func newGroupTestConfig() *config.Config {
	return &config.Config{
		Statuses:   []string{"backlog", "todo", "in-progress", "done"},
		Priorities: []string{"low", "medium", "high", "critical"},
		Classes: []config.ClassConfig{
			{Name: "expedite"},
			{Name: classStandard},
			{Name: "intangible"},
		},
	}
}

func TestGroupByAssignee(t *testing.T) {
	cfg := newGroupTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Assignee: "alice"},
		{ID: 2, Status: "in-progress", Assignee: "bob"},
		{ID: 3, Status: "todo", Assignee: "alice"},
		{ID: 4, Status: "backlog"},
	}

	result := GroupBy(tasks, "assignee", cfg)
	if len(result.Groups) != 3 {
		t.Fatalf("Groups len = %d, want 3", len(result.Groups))
	}

	// Sorted alphabetically: (unassigned), alice, bob.
	if result.Groups[0].Key != "(unassigned)" {
		t.Errorf("Groups[0].Key = %q, want %q", result.Groups[0].Key, "(unassigned)")
	}
	if result.Groups[0].Total != 1 {
		t.Errorf("Groups[0].Total = %d, want 1", result.Groups[0].Total)
	}
	if result.Groups[1].Key != "alice" {
		t.Errorf("Groups[1].Key = %q, want %q", result.Groups[1].Key, "alice")
	}
	if result.Groups[1].Total != 2 {
		t.Errorf("Groups[1].Total = %d, want 2", result.Groups[1].Total)
	}
}

func TestGroupByTag(t *testing.T) {
	cfg := newGroupTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Tags: []string{"backend", "api"}},
		{ID: 2, Status: "todo", Tags: []string{"frontend"}},
		{ID: 3, Status: "todo"},
	}

	result := GroupBy(tasks, "tag", cfg)

	// Task #1 appears in both "backend" and "api" groups.
	if len(result.Groups) != 4 {
		t.Fatalf("Groups len = %d, want 4 (untagged, api, backend, frontend)", len(result.Groups))
	}

	// Sorted alphabetically: (untagged), api, backend, frontend.
	keys := make([]string, len(result.Groups))
	for i, g := range result.Groups {
		keys[i] = g.Key
	}
	// Check that multi-tag task appears in api group.
	found := false
	for _, g := range result.Groups {
		if g.Key == "api" && g.Total == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'api' group with 1 task")
	}
}

func TestGroupByClass(t *testing.T) {
	cfg := newGroupTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Class: "expedite"},
		{ID: 2, Status: "todo", Class: classStandard},
		{ID: 3, Status: "todo"}, // empty class = standard
	}

	result := GroupBy(tasks, "class", cfg)
	if len(result.Groups) != 2 {
		t.Fatalf("Groups len = %d, want 2", len(result.Groups))
	}
	// Sorted by config class order: expedite, standard.
	if result.Groups[0].Key != "expedite" {
		t.Errorf("Groups[0].Key = %q, want %q", result.Groups[0].Key, "expedite")
	}
	if result.Groups[1].Key != classStandard {
		t.Errorf("Groups[1].Key = %q, want %q", result.Groups[1].Key, classStandard)
	}
	if result.Groups[1].Total != 2 {
		t.Errorf("Groups[1].Total = %d, want 2 (task #2 + #3 with empty class)", result.Groups[1].Total)
	}
}

func TestGroupByStatus(t *testing.T) {
	cfg := newGroupTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo"},
		{ID: 2, Status: "todo"},
		{ID: 3, Status: "done"},
	}

	result := GroupBy(tasks, "status", cfg)

	// Groups sorted by config status order.
	if len(result.Groups) != 2 {
		t.Fatalf("Groups len = %d, want 2", len(result.Groups))
	}
	if result.Groups[0].Key != "todo" {
		t.Errorf("Groups[0].Key = %q, want %q", result.Groups[0].Key, "todo")
	}
	if result.Groups[1].Key != "done" {
		t.Errorf("Groups[1].Key = %q, want %q", result.Groups[1].Key, "done")
	}
}

func TestGroupByReturnsStatusSummaryPerGroup(t *testing.T) {
	cfg := newGroupTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Assignee: "alice"},
		{ID: 2, Status: "in-progress", Assignee: "alice"},
		{ID: 3, Status: "todo", Assignee: "alice"},
	}

	result := GroupBy(tasks, "assignee", cfg)
	if len(result.Groups) != 1 {
		t.Fatalf("Groups len = %d, want 1", len(result.Groups))
	}

	g := result.Groups[0]
	if len(g.Statuses) != len(cfg.Statuses) {
		t.Fatalf("Statuses len = %d, want %d", len(g.Statuses), len(cfg.Statuses))
	}
	// Check todo count.
	for _, ss := range g.Statuses {
		if ss.Status == "todo" && ss.Count != 2 {
			t.Errorf("todo count = %d, want 2", ss.Count)
		}
		if ss.Status == "in-progress" && ss.Count != 1 {
			t.Errorf("in-progress count = %d, want 1", ss.Count)
		}
	}
}
