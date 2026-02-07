package board

import (
	"testing"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func makeTasks() []*task.Task {
	return []*task.Task{
		{ID: 1, Title: "Task 1", Status: "backlog", Priority: "high", Assignee: "alice", Tags: []string{"backend"}},
		{ID: 2, Title: "Task 2", Status: "in-progress", Priority: "medium", Assignee: "bob", Tags: []string{"frontend"}},
		{ID: 3, Title: "Task 3", Status: "done", Priority: "low", Assignee: "alice", Tags: []string{"backend", "api"}},
		{ID: 4, Title: "Task 4", Status: "backlog", Priority: "high", Tags: []string{"frontend"}},
	}
}

func TestFilterByStatus(t *testing.T) {
	result := Filter(makeTasks(), FilterOptions{Statuses: []string{"backlog"}})
	if len(result) != 2 {
		t.Errorf("got %d tasks, want 2", len(result))
	}
}

func TestFilterByMultipleStatuses(t *testing.T) {
	result := Filter(makeTasks(), FilterOptions{Statuses: []string{"backlog", "done"}})
	if len(result) != 3 {
		t.Errorf("got %d tasks, want 3", len(result))
	}
}

func TestFilterByAssignee(t *testing.T) {
	result := Filter(makeTasks(), FilterOptions{Assignee: "alice"})
	if len(result) != 2 {
		t.Errorf("got %d tasks, want 2", len(result))
	}
}

func TestFilterByTag(t *testing.T) {
	result := Filter(makeTasks(), FilterOptions{Tag: "api"})
	if len(result) != 1 {
		t.Errorf("got %d tasks, want 1", len(result))
	}
}

func TestFilterCombined(t *testing.T) {
	result := Filter(makeTasks(), FilterOptions{
		Statuses: []string{"backlog"},
		Assignee: "alice",
	})
	if len(result) != 1 {
		t.Errorf("got %d tasks, want 1", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("got task #%d, want #1", result[0].ID)
	}
}

func TestFilterNoMatch(t *testing.T) {
	result := Filter(makeTasks(), FilterOptions{Assignee: "nobody"})
	if len(result) != 0 {
		t.Errorf("got %d tasks, want 0", len(result))
	}
}

func TestFilterEmpty(t *testing.T) {
	result := Filter(makeTasks(), FilterOptions{})
	if len(result) != 4 {
		t.Errorf("got %d tasks, want 4 (no filter)", len(result))
	}
}

func makeTasksWithBlocked() []*task.Task {
	return []*task.Task{
		{ID: 1, Title: "Normal", Status: "backlog"},
		{ID: 2, Title: "Blocked", Status: "in-progress", Blocked: true, BlockReason: "waiting"},
		{ID: 3, Title: "Also blocked", Status: "todo", Blocked: true, BlockReason: "dependency"},
		{ID: 4, Title: "Not blocked", Status: "in-progress"},
	}
}

func TestFilterBlocked(t *testing.T) {
	blocked := true
	result := Filter(makeTasksWithBlocked(), FilterOptions{Blocked: &blocked})
	if len(result) != 2 {
		t.Errorf("got %d tasks, want 2 blocked", len(result))
	}
	for _, tk := range result {
		if !tk.Blocked {
			t.Errorf("task #%d should be blocked", tk.ID)
		}
	}
}

func TestFilterNotBlocked(t *testing.T) {
	notBlocked := false
	result := Filter(makeTasksWithBlocked(), FilterOptions{Blocked: &notBlocked})
	if len(result) != 2 {
		t.Errorf("got %d tasks, want 2 not-blocked", len(result))
	}
	for _, tk := range result {
		if tk.Blocked {
			t.Errorf("task #%d should not be blocked", tk.ID)
		}
	}
}

func TestFilterBlockedNil(t *testing.T) {
	result := Filter(makeTasksWithBlocked(), FilterOptions{})
	if len(result) != 4 {
		t.Errorf("got %d tasks, want 4 (no blocked filter)", len(result))
	}
}

func TestFilterByParentID(t *testing.T) {
	parent1 := 10
	parent2 := 20
	tasks := []*task.Task{
		{ID: 1, Title: "Child of 10", Parent: &parent1},
		{ID: 2, Title: "Child of 20", Parent: &parent2},
		{ID: 3, Title: "No parent"},
		{ID: 4, Title: "Also child of 10", Parent: &parent1},
	}

	result := Filter(tasks, FilterOptions{ParentID: &parent1})
	if len(result) != 2 {
		t.Errorf("got %d tasks, want 2 with parent 10", len(result))
	}
	for _, tk := range result {
		if tk.Parent == nil || *tk.Parent != parent1 {
			t.Errorf("task #%d has wrong parent", tk.ID)
		}
	}
}

func TestFilterByParentIDNoMatch(t *testing.T) {
	noParent := 99
	tasks := []*task.Task{
		{ID: 1, Title: "No parent"},
	}
	result := Filter(tasks, FilterOptions{ParentID: &noParent})
	if len(result) != 0 {
		t.Errorf("got %d tasks, want 0", len(result))
	}
}

func makeTasksWithDeps() []*task.Task {
	return []*task.Task{
		{ID: 1, Title: "Task 1", Status: "done"},
		{ID: 2, Title: "Task 2", Status: "todo", DependsOn: []int{1}},    // dep satisfied
		{ID: 3, Title: "Task 3", Status: "todo", DependsOn: []int{1, 4}}, // dep 4 not done
		{ID: 4, Title: "Task 4", Status: "in-progress"},                  // no deps
		{ID: 5, Title: "Task 5", Status: "todo", DependsOn: []int{99}},   // dep missing
		{ID: 6, Title: "Task 6", Status: "backlog"},                      // no deps
	}
}

func TestFilterUnblockedAllDepsSatisfied(t *testing.T) {
	tasks := makeTasksWithDeps()
	result := FilterUnblocked(tasks, "done")
	// Tasks 1, 2, 4, 6 should pass (1: no deps, 2: dep 1 is done, 4: no deps, 6: no deps)
	// Tasks 3, 5 should not (3: dep 4 not done, 5: dep 99 missing)
	if len(result) != 4 {
		t.Errorf("got %d tasks, want 4 unblocked", len(result))
		for _, tk := range result {
			t.Logf("  got task #%d", tk.ID)
		}
	}
}

func TestFilterUnblockedNoDeps(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Title: "No deps", Status: "todo"},
	}
	result := FilterUnblocked(tasks, "done")
	if len(result) != 1 {
		t.Errorf("got %d tasks, want 1 (no deps = unblocked)", len(result))
	}
}

func TestFilterUnblockedMissingDep(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Title: "Depends on ghost", Status: "todo", DependsOn: []int{99}},
	}
	result := FilterUnblocked(tasks, "done")
	if len(result) != 0 {
		t.Errorf("got %d tasks, want 0 (missing dep = blocked)", len(result))
	}
}

func TestCountByStatus(t *testing.T) {
	tasks := makeTasks() // 2 backlog, 1 in-progress, 1 done
	counts := CountByStatus(tasks)

	if counts["backlog"] != 2 {
		t.Errorf("backlog count = %d, want 2", counts["backlog"])
	}
	if counts["in-progress"] != 1 {
		t.Errorf("in-progress count = %d, want 1", counts["in-progress"])
	}
	if counts["done"] != 1 {
		t.Errorf("done count = %d, want 1", counts["done"])
	}
	if counts["todo"] != 0 {
		t.Errorf("todo count = %d, want 0", counts["todo"])
	}
}
