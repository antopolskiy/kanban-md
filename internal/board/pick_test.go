package board

import (
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func newPickTestConfig() *config.Config {
	return &config.Config{
		Statuses: []config.StatusConfig{
			{Name: "backlog"}, {Name: "todo"}, {Name: "in-progress"}, {Name: "done"},
		},
		Priorities: []string{"low", "medium", "high", "critical"},
		Classes: []config.ClassConfig{
			{Name: "expedite", WIPLimit: 1, BypassColumnWIP: true},
			{Name: "fixed-date"},
			{Name: "standard"},
			{Name: "intangible"},
		},
	}
}

func TestPickHighestPriority(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "low"},
		{ID: 2, Status: "todo", Priority: "critical"},
		{ID: 3, Status: "todo", Priority: "high"},
	}

	picked := Pick(cfg, tasks, PickOptions{})
	if picked == nil {
		t.Fatal("Pick() returned nil, want task #2")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (critical priority)", picked.ID)
	}
}

func TestPickSkipsClaimed(t *testing.T) {
	cfg := newPickTestConfig()
	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", ClaimedBy: "agent-1", ClaimedAt: &now},
		{ID: 2, Status: "todo", Priority: "high"},
	}

	picked := Pick(cfg, tasks, PickOptions{ClaimTimeout: time.Hour})
	if picked == nil {
		t.Fatal("Pick() returned nil, want task #2")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (skip claimed)", picked.ID)
	}
}

func TestPickSkipsBlocked(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", Blocked: true, BlockReason: "waiting"},
		{ID: 2, Status: "todo", Priority: "high"},
	}

	picked := Pick(cfg, tasks, PickOptions{})
	if picked == nil {
		t.Fatal("Pick() returned nil, want task #2")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (skip blocked)", picked.ID)
	}
}

func TestPickSkipsUnmetDeps(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", DependsOn: []int{10}},
		{ID: 2, Status: "todo", Priority: "high"},
		{ID: 10, Status: "in-progress", Priority: "medium"}, // not done
	}

	picked := Pick(cfg, tasks, PickOptions{})
	if picked == nil {
		t.Fatal("Pick() returned nil, want task #2")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (skip unmet deps)", picked.ID)
	}
}

func TestPickSatisfiedDeps(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", DependsOn: []int{10}},
		{ID: 2, Status: "todo", Priority: "high"},
		{ID: 10, Status: "done", Priority: "medium"}, // done
	}

	picked := Pick(cfg, tasks, PickOptions{})
	if picked == nil {
		t.Fatal("Pick() returned nil")
	}
	if picked.ID != 1 {
		t.Errorf("Pick() ID = %d, want 1 (deps satisfied, critical priority)", picked.ID)
	}
}

func TestPickFromSpecificStatus(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical"},
		{ID: 2, Status: "backlog", Priority: "high"},
	}

	picked := Pick(cfg, tasks, PickOptions{Statuses: []string{"backlog"}})
	if picked == nil {
		t.Fatal("Pick() returned nil, want task #2")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (from backlog only)", picked.ID)
	}
}

func TestPickNoneAvailable(t *testing.T) {
	cfg := newPickTestConfig()
	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", ClaimedBy: "agent-1", ClaimedAt: &now},
		{ID: 2, Status: "done", Priority: "high"}, // terminal
	}

	picked := Pick(cfg, tasks, PickOptions{ClaimTimeout: time.Hour})
	if picked != nil {
		t.Errorf("Pick() = task #%d, want nil", picked.ID)
	}
}

func TestPickExpiredClaimIsAvailable(t *testing.T) {
	cfg := newPickTestConfig()
	expired := time.Now().Add(-2 * time.Hour)
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", ClaimedBy: "agent-1", ClaimedAt: &expired},
	}

	picked := Pick(cfg, tasks, PickOptions{ClaimTimeout: time.Hour})
	if picked == nil {
		t.Fatal("Pick() returned nil, want task #1 (expired claim)")
	}
	if picked.ID != 1 {
		t.Errorf("Pick() ID = %d, want 1", picked.ID)
	}
}

func TestPickWithTagFilter(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", Tags: []string{"backend"}},
		{ID: 2, Status: "todo", Priority: "high", Tags: []string{"frontend"}},
	}

	picked := Pick(cfg, tasks, PickOptions{Tags: []string{"frontend"}})
	if picked == nil {
		t.Fatal("Pick() returned nil, want task #2")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (tag filter)", picked.ID)
	}
}

func TestPickWithMultipleTagFilter(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", Tags: []string{"backend"}},
		{ID: 2, Status: "todo", Priority: "high", Tags: []string{"frontend"}},
		{ID: 3, Status: "todo", Priority: "medium", Tags: []string{"coverage"}},
	}

	// Filter by frontend OR coverage — should pick #2 (highest priority among matches).
	picked := Pick(cfg, tasks, PickOptions{Tags: []string{"frontend", "coverage"}})
	if picked == nil {
		t.Fatal("Pick() returned nil, want task #2")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (multi-tag filter, highest priority)", picked.ID)
	}
}

func TestPickExpediteBeforeStandard(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", Class: "standard"},
		{ID: 2, Status: "todo", Priority: "low", Class: "expedite"},
	}

	picked := Pick(cfg, tasks, PickOptions{})
	if picked == nil {
		t.Fatal("Pick() returned nil")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (expedite before standard)", picked.ID)
	}
}

func TestPickIntangibleLast(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "critical", Class: "intangible"},
		{ID: 2, Status: "todo", Priority: "low", Class: "standard"},
	}

	picked := Pick(cfg, tasks, PickOptions{})
	if picked == nil {
		t.Fatal("Pick() returned nil")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (standard before intangible)", picked.ID)
	}
}

func TestPickFixedDateByDueDate(t *testing.T) {
	cfg := newPickTestConfig()
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "high", Class: "fixed-date", Due: parseTestDate(t, "2026-03-15")},
		{ID: 2, Status: "todo", Priority: "low", Class: "fixed-date", Due: parseTestDate(t, "2026-02-15")},
	}

	picked := Pick(cfg, tasks, PickOptions{})
	if picked == nil {
		t.Fatal("Pick() returned nil")
	}
	if picked.ID != 2 {
		t.Errorf("Pick() ID = %d, want 2 (soonest due date first)", picked.ID)
	}
}

func parseTestDate(t *testing.T, s string) *date.Date {
	t.Helper()
	d, err := date.Parse(s)
	if err != nil {
		t.Fatalf("parseTestDate(%q): %v", s, err)
	}
	return &d
}
