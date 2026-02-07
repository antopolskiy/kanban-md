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
