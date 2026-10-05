package tui

import (
	"slices"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func parentID(id int) *int { return &id }

func TestHierarchyDepthAndAncestors(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1},
		{ID: 2, Parent: parentID(1), Status: "archived"},
		{ID: 3, Parent: parentID(2)},
		{ID: 4, Parent: parentID(40)},
		{ID: 5, Parent: parentID(4)},
		{ID: 6, Parent: parentID(6)},
		{ID: 7, Parent: parentID(6)},
		{ID: 8, Parent: parentID(9)},
		{ID: 9, Parent: parentID(8)},
		{ID: 10, Parent: parentID(8)},
		{ID: 11, Parent: parentID(10)},
	}
	want := map[int]int{1: 0, 2: 1, 3: 2, 4: 0, 5: 1, 6: 0, 7: 1, 8: unknownDepth, 9: unknownDepth, 10: unknownDepth, 11: unknownDepth}
	for _, reverse := range []bool{false, true} {
		input := slices.Clone(tasks)
		if reverse {
			slices.Reverse(input)
		}
		h := newTaskHierarchy(input)
		for id, depth := range want {
			if h.depths[id] != depth {
				t.Fatalf("reverse=%v depth #%d=%d, want %d", reverse, id, h.depths[id], depth)
			}
		}
		path := h.ancestors(tasks[2])
		if path.marker != "" || len(path.tasks) != 2 || path.tasks[0].ID != 1 || path.tasks[1].ID != 2 {
			t.Fatalf("archived ancestor path = %#v", path)
		}
		for _, tt := range []struct {
			id     int
			marker string
		}{
			{4, "missing parent #40"},
			{5, "missing parent #40"},
			{6, "self parent #6"},
			{7, "self parent #6"},
			{8, "cycle to #8"},
			{11, "cycle to #8"},
		} {
			if got := h.ancestors(h.byID[tt.id]); got.marker != tt.marker {
				t.Fatalf("#%d marker=%q, want %q", tt.id, got.marker, tt.marker)
			}
		}
	}
}

func TestHierarchyDeepChainIterative(t *testing.T) {
	const count = 10000
	tasks := make([]*task.Task, count)
	for i := range tasks {
		tasks[i] = &task.Task{ID: i + 1}
		if i > 0 {
			tasks[i].Parent = parentID(i)
		}
	}
	slices.Reverse(tasks)
	h := newTaskHierarchy(tasks)
	if h.depths[count] != count-1 || h.depths[1] != 0 {
		t.Fatalf("deep chain depths: last=%d root=%d", h.depths[count], h.depths[1])
	}
	path := h.ancestors(h.byID[count])
	if path.marker != "" || len(path.tasks) != count-1 || path.tasks[0].ID != 1 || path.tasks[count-2].ID != count-1 {
		t.Fatalf("deep path length=%d marker=%q", len(path.tasks), path.marker)
	}
}

func TestHierarchyLongCycleAndDescendant(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Parent: parentID(2)},
		{ID: 2, Parent: parentID(3)},
		{ID: 3, Parent: parentID(1)},
		{ID: 4, Parent: parentID(3)},
	}
	h := newTaskHierarchy(tasks)
	for id := range h.byID {
		if h.depths[id] != unknownDepth {
			t.Fatalf("cycle-connected #%d depth=%d", id, h.depths[id])
		}
	}
	if path := h.ancestors(tasks[3]); path.marker != "cycle to #3" || len(path.tasks) != 3 {
		t.Fatalf("cycle descendant path=%#v", path)
	}
}
