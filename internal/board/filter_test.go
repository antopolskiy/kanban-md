package board

import (
	"testing"
	"time"

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
	result := FilterUnblocked(tasks, testConfig())
	// Tasks 1, 2, 4, 5, 6 should pass:
	// 1/4/6 have no deps, 2 depends on done task 1, 5 has missing dep ID treated as satisfied.
	// Task 3 should not (dep 4 not done).
	if len(result) != 5 {
		t.Errorf("got %d tasks, want 5 unblocked", len(result))
		for _, tk := range result {
			t.Logf("  got task #%d", tk.ID)
		}
	}
}

func TestFilterUnblockedNoDeps(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Title: "No deps", Status: "todo"},
	}
	result := FilterUnblocked(tasks, testConfig())
	if len(result) != 1 {
		t.Errorf("got %d tasks, want 1 (no deps = unblocked)", len(result))
	}
}

func TestFilterUnblockedMissingDep(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Title: "Depends on ghost", Status: "todo", DependsOn: []int{99}},
	}
	result := FilterUnblocked(tasks, testConfig())
	if len(result) != 1 {
		t.Errorf("got %d tasks, want 1 (missing dep = satisfied)", len(result))
	}
}

func TestFilterBySearch(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Title: "Fix login bug", Status: "backlog", Priority: "high", Body: "Users cannot log in with SSO"},
		{ID: 2, Title: "Add dark mode", Status: "todo", Priority: "medium", Tags: []string{"ui", "theme"}},
		{ID: 3, Title: "Update README", Status: "done", Priority: "low", Body: "Add installation instructions"},
		{ID: 4, Title: "Refactor auth", Status: "in-progress", Priority: "high", Body: "Split login flow"},
	}

	tests := []struct {
		name   string
		search string
		want   int
	}{
		{"match title", "login", 2},       // "Fix login bug" and "Split login flow" in body
		{"match body", "SSO", 1},          // body of task 1
		{"match tag", "theme", 1},         // tag of task 2
		{"case insensitive", "readme", 1}, // title of task 3
		{"no match", "nonexistent", 0},
		{"empty search", "", 4}, // all tasks
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Filter(tasks, FilterOptions{Search: tt.search})
			if len(result) != tt.want {
				t.Errorf("got %d tasks, want %d", len(result), tt.want)
			}
		})
	}
}

func TestFilterSearchCombinedWithOtherFilters(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Title: "Fix login bug", Status: "backlog", Priority: "high"},
		{ID: 2, Title: "Fix signup bug", Status: "backlog", Priority: "medium"},
		{ID: 3, Title: "Fix login flow", Status: "done", Priority: "high"},
	}

	result := Filter(tasks, FilterOptions{
		Search:   "login",
		Statuses: []string{"backlog"},
	})
	if len(result) != 1 {
		t.Errorf("got %d tasks, want 1 (login + backlog)", len(result))
	}
	if len(result) > 0 && result[0].ID != 1 {
		t.Errorf("got task #%d, want #1", result[0].ID)
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

func TestIsUnclaimed(t *testing.T) {
	const defaultTimeout = time.Hour

	pastTime := time.Now().Add(-2 * time.Hour)      // expired
	recentTime := time.Now().Add(-10 * time.Minute) // active

	tests := []struct {
		name    string
		task    *task.Task
		timeout time.Duration
		want    bool
	}{
		{
			name:    "no claim is unclaimed",
			task:    &task.Task{ID: 1, ClaimedBy: ""},
			timeout: defaultTimeout,
			want:    true,
		},
		{
			name:    "active claim is not unclaimed",
			task:    &task.Task{ID: 2, ClaimedBy: "agent-1", ClaimedAt: &recentTime},
			timeout: defaultTimeout,
			want:    false,
		},
		{
			name:    "expired claim is unclaimed",
			task:    &task.Task{ID: 3, ClaimedBy: "agent-1", ClaimedAt: &pastTime},
			timeout: defaultTimeout,
			want:    true,
		},
		{
			name:    "timeout zero with claim means never expires",
			task:    &task.Task{ID: 4, ClaimedBy: "agent-1", ClaimedAt: &pastTime},
			timeout: 0,
			want:    false,
		},
		{
			name:    "ClaimedBy set but ClaimedAt nil with timeout is not unclaimed",
			task:    &task.Task{ID: 5, ClaimedBy: "agent-1", ClaimedAt: nil},
			timeout: defaultTimeout,
			want:    false,
		},
		{
			name:    "ClaimedBy set but ClaimedAt nil without timeout is not unclaimed",
			task:    &task.Task{ID: 6, ClaimedBy: "agent-1", ClaimedAt: nil},
			timeout: 0,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnclaimed(tt.task, tt.timeout)
			if got != tt.want {
				t.Errorf("IsUnclaimed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterUnclaimed(t *testing.T) {
	recentTime := time.Now().Add(-10 * time.Minute) // active claim
	pastTime := time.Now().Add(-2 * time.Hour)      // expired claim

	tasks := []*task.Task{
		{ID: 1, Title: "No claim", Status: "todo"},
		{ID: 2, Title: "Active claim", Status: "todo", ClaimedBy: "agent-1", ClaimedAt: &recentTime},
		{ID: 3, Title: "Expired claim", Status: "todo", ClaimedBy: "agent-2", ClaimedAt: &pastTime},
	}

	result := Filter(tasks, FilterOptions{Unclaimed: true, ClaimTimeout: time.Hour})
	if len(result) != 2 {
		t.Errorf("got %d tasks, want 2 (unclaimed + expired)", len(result))
		for _, tk := range result {
			t.Logf("  got task #%d %q", tk.ID, tk.Title)
		}
	}

	// Verify the right tasks are returned.
	ids := make(map[int]bool, len(result))
	for _, tk := range result {
		ids[tk.ID] = true
	}
	if !ids[1] {
		t.Error("expected task #1 (unclaimed) in results")
	}
	if ids[2] {
		t.Error("task #2 (active claim) should NOT be in results")
	}
	if !ids[3] {
		t.Error("expected task #3 (expired claim) in results")
	}
}

func TestFilterByClaimedBy(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Title: "Agent 1", Status: "todo", ClaimedBy: "agent-1", ClaimedAt: &now},
		{ID: 2, Title: "Agent 2", Status: "todo", ClaimedBy: "agent-2", ClaimedAt: &now},
		{ID: 3, Title: "No claim", Status: "todo"},
	}

	result := Filter(tasks, FilterOptions{ClaimedBy: "agent-1"})
	if len(result) != 1 {
		t.Errorf("got %d tasks, want 1", len(result))
	}
	if len(result) > 0 && result[0].ID != 1 {
		t.Errorf("got task #%d, want #1", result[0].ID)
	}
}
