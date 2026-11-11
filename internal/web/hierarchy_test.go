package web

import (
	"testing"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestHierarchyDepthAndAncestors(t *testing.T) {
	parent := 1
	child := 2
	tasks := []*task.Task{
		{ID: 1, Title: "Root"},
		{ID: 2, Title: "Child", Parent: &parent},
		{ID: 3, Title: "Grand", Parent: &child},
	}
	h := newHierarchy(tasks)
	if h.depths[1] != 0 || h.depths[2] != 1 || h.depths[3] != 2 {
		t.Fatalf("depths = %#v", h.depths)
	}
	path := h.ancestors(tasks[2])
	if path.marker != "" || len(path.tasks) != 2 || path.tasks[0].ID != 1 || path.tasks[1].ID != 2 {
		t.Fatalf("ancestors = %+v marker %q", path.tasks, path.marker)
	}
}

func TestHierarchyCycleIsUnknown(t *testing.T) {
	first := 2
	second := 1
	tasks := []*task.Task{
		{ID: 1, Parent: &first},
		{ID: 2, Parent: &second},
	}
	h := newHierarchy(tasks)
	if h.depths[1] != unknownDepth || h.depths[2] != unknownDepth {
		t.Fatalf("depths = %#v", h.depths)
	}
	if !levelMatch(unknownDepth, levelUnknown) || levelMatch(0, levelUnknown) {
		t.Fatal("unknown level match")
	}
	if !levelMatch(1, "1") || levelMatch(1, "0") {
		t.Fatal("numeric level match")
	}
}
