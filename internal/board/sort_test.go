package board

import (
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func testConfig() *config.Config {
	return config.NewDefault("Test")
}

func taskIDs(tasks []*task.Task) [3]int {
	return [3]int{tasks[0].ID, tasks[1].ID, tasks[2].ID}
}

func TestSortByID(t *testing.T) {
	tasks := []*task.Task{
		{ID: 3}, {ID: 1}, {ID: 2},
	}
	Sort(tasks, "id", false, testConfig())
	if got := taskIDs(tasks); got != [3]int{1, 2, 3} {
		t.Errorf("sort by id = %v, want [1, 2, 3]", got)
	}
}

func TestSortByIDReverse(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1}, {ID: 3}, {ID: 2},
	}
	Sort(tasks, "id", true, testConfig())
	if got := taskIDs(tasks); got != [3]int{3, 2, 1} {
		t.Errorf("sort by id reverse = %v, want [3, 2, 1]", got)
	}
}

func TestSortByStatus(t *testing.T) {
	cfg := testConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "done"},
		{ID: 2, Status: "backlog"},
		{ID: 3, Status: "in-progress"},
	}
	Sort(tasks, "status", false, cfg)
	got := [3]string{tasks[0].Status, tasks[1].Status, tasks[2].Status}
	if got != [3]string{"backlog", "in-progress", "done"} {
		t.Errorf("sort by status = %v", got)
	}
}

func TestSortByPriority(t *testing.T) {
	cfg := testConfig()
	tasks := []*task.Task{
		{ID: 1, Priority: "high"},
		{ID: 2, Priority: "low"},
		{ID: 3, Priority: "critical"},
	}
	Sort(tasks, "priority", false, cfg)
	got := [3]string{tasks[0].Priority, tasks[1].Priority, tasks[2].Priority}
	if got != [3]string{"low", "high", "critical"} {
		t.Errorf("sort by priority = %v", got)
	}
}

func TestSortByDue(t *testing.T) {
	d1 := date.New(2026, time.February, 10)
	d2 := date.New(2026, time.February, 20)
	tasks := []*task.Task{
		{ID: 1, Due: &d2},
		{ID: 2, Due: nil},
		{ID: 3, Due: &d1},
	}
	Sort(tasks, "due", false, testConfig())
	// d1 (earliest) first, d2 second, nil last.
	if got := taskIDs(tasks); got != [3]int{3, 1, 2} {
		t.Errorf("sort by due = %v, want [3, 1, 2]", got)
	}
}

func TestSortByCreated(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Created: now.Add(2 * time.Hour)},
		{ID: 2, Created: now},
		{ID: 3, Created: now.Add(time.Hour)},
	}
	Sort(tasks, "created", false, testConfig())
	if got := taskIDs(tasks); got != [3]int{2, 3, 1} {
		t.Errorf("sort by created = %v, want [2, 3, 1]", got)
	}
}

func TestSortByUpdated(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Updated: now.Add(time.Hour)},
		{ID: 2, Updated: now.Add(2 * time.Hour)},
		{ID: 3, Updated: now},
	}
	Sort(tasks, "updated", false, testConfig())
	if got := taskIDs(tasks); got != [3]int{3, 1, 2} {
		t.Errorf("sort by updated = %v, want [3, 1, 2]", got)
	}
}

func TestSortByTitle(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Title: "banana"},
		{ID: 2, Title: "Apple"},
		{ID: 3, Title: "cherry"},
	}
	// Case-insensitive ascending: Apple, banana, cherry.
	Sort(tasks, "title", false, testConfig())
	if got := taskIDs(tasks); got != [3]int{2, 1, 3} {
		t.Errorf("sort by title = %v, want [2, 1, 3]", got)
	}
}

func TestSortByTitleReverse(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Title: "banana"},
		{ID: 2, Title: "Apple"},
		{ID: 3, Title: "cherry"},
	}
	Sort(tasks, "title", true, testConfig())
	if got := taskIDs(tasks); got != [3]int{3, 1, 2} {
		t.Errorf("sort by title reverse = %v, want [3, 1, 2]", got)
	}
}

func TestSortByUnknownFieldFallsBackToID(t *testing.T) {
	tasks := []*task.Task{
		{ID: 3}, {ID: 1}, {ID: 2},
	}
	Sort(tasks, "nonexistent", false, testConfig())
	if got := taskIDs(tasks); got != [3]int{1, 2, 3} {
		t.Errorf("sort by unknown field = %v, want [1, 2, 3] (fallback to ID)", got)
	}
}
