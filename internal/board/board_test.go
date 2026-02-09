package board

import (
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestSummaryEmptyBoard(t *testing.T) {
	cfg := config.NewDefault("Test Board")
	s := Summary(cfg, nil, time.Now())

	if s.BoardName != "Test Board" {
		t.Errorf("BoardName = %q, want %q", s.BoardName, "Test Board")
	}
	if s.TotalTasks != 0 {
		t.Errorf("TotalTasks = %d, want 0", s.TotalTasks)
	}
	wantStatuses := cfg.BoardStatuses()
	if len(s.Statuses) != len(wantStatuses) {
		t.Errorf("Statuses len = %d, want %d", len(s.Statuses), len(wantStatuses))
	}
	for _, ss := range s.Statuses {
		if ss.Count != 0 {
			t.Errorf("status %q count = %d, want 0", ss.Status, ss.Count)
		}
	}
}

func TestSummaryCountsByStatus(t *testing.T) {
	cfg := config.NewDefault("Test Board")
	tasks := []*task.Task{
		{ID: 1, Status: "backlog", Priority: "high"},
		{ID: 2, Status: "backlog", Priority: "medium"},
		{ID: 3, Status: "in-progress", Priority: "high"},
		{ID: 4, Status: "done", Priority: "low"},
	}

	s := Summary(cfg, tasks, time.Now())

	if s.TotalTasks != 4 {
		t.Errorf("TotalTasks = %d, want 4", s.TotalTasks)
	}

	statusCounts := make(map[string]int)
	for _, ss := range s.Statuses {
		statusCounts[ss.Status] = ss.Count
	}
	if statusCounts["backlog"] != 2 {
		t.Errorf("backlog count = %d, want 2", statusCounts["backlog"])
	}
	if statusCounts[sectionInProgress] != 1 {
		t.Errorf("in-progress count = %d, want 1", statusCounts[sectionInProgress])
	}
	if statusCounts["done"] != 1 {
		t.Errorf("done count = %d, want 1", statusCounts["done"])
	}
}

func TestSummaryWIPLimits(t *testing.T) {
	cfg := config.NewDefault("Test Board")
	cfg.WIPLimits = map[string]int{sectionInProgress: 3}

	s := Summary(cfg, nil, time.Now())

	for _, ss := range s.Statuses {
		if ss.Status == sectionInProgress && ss.WIPLimit != 3 {
			t.Errorf("in-progress WIPLimit = %d, want 3", ss.WIPLimit)
		}
	}
}

func TestSummaryBlockedCounts(t *testing.T) {
	cfg := config.NewDefault("Test Board")
	tasks := []*task.Task{
		{ID: 1, Status: "in-progress", Priority: "medium", Blocked: true, BlockReason: "waiting"},
		{ID: 2, Status: "in-progress", Priority: "medium"},
		{ID: 3, Status: "todo", Priority: "medium", Blocked: true, BlockReason: "deps"},
	}

	s := Summary(cfg, tasks, time.Now())

	statusBlocked := make(map[string]int)
	for _, ss := range s.Statuses {
		statusBlocked[ss.Status] = ss.Blocked
	}
	if statusBlocked[sectionInProgress] != 1 {
		t.Errorf("in-progress blocked = %d, want 1", statusBlocked[sectionInProgress])
	}
	if statusBlocked["todo"] != 1 {
		t.Errorf("todo blocked = %d, want 1", statusBlocked["todo"])
	}
}

func TestSummaryOverdueCounts(t *testing.T) {
	cfg := config.NewDefault("Test Board")
	pastDue := date.New(2020, 1, 1)
	futureDue := date.New(2030, 1, 1)

	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "medium", Due: &pastDue},        // overdue
		{ID: 2, Status: "todo", Priority: "medium", Due: &futureDue},      // not overdue
		{ID: 3, Status: "done", Priority: "medium", Due: &pastDue},        // terminal, not overdue
		{ID: 4, Status: "in-progress", Priority: "medium", Due: &pastDue}, // overdue
		{ID: 5, Status: "backlog", Priority: "medium"},                    // no due date
	}

	s := Summary(cfg, tasks, time.Now())

	totalOverdue := 0
	for _, ss := range s.Statuses {
		totalOverdue += ss.Overdue
	}
	if totalOverdue != 2 {
		t.Errorf("total overdue = %d, want 2", totalOverdue)
	}
}

func TestSummaryTerminalNotOverdue(t *testing.T) {
	cfg := config.NewDefault("Test Board")
	pastDue := date.New(2020, 1, 1)

	tasks := []*task.Task{
		{ID: 1, Status: "done", Priority: "medium", Due: &pastDue},
	}

	s := Summary(cfg, tasks, time.Now())

	for _, ss := range s.Statuses {
		if ss.Status == "done" && ss.Overdue != 0 {
			t.Errorf("done overdue = %d, want 0 (terminal tasks not overdue)", ss.Overdue)
		}
	}
}

func TestSummaryPriorityDistribution(t *testing.T) {
	cfg := config.NewDefault("Test Board")
	tasks := []*task.Task{
		{ID: 1, Status: "backlog", Priority: "high"},
		{ID: 2, Status: "backlog", Priority: "high"},
		{ID: 3, Status: "todo", Priority: "medium"},
		{ID: 4, Status: "done", Priority: "low"},
	}

	s := Summary(cfg, tasks, time.Now())

	prioMap := make(map[string]int)
	for _, pc := range s.Priorities {
		prioMap[pc.Priority] = pc.Count
	}
	if prioMap["high"] != 2 {
		t.Errorf("high count = %d, want 2", prioMap["high"])
	}
	if prioMap["medium"] != 1 {
		t.Errorf("medium count = %d, want 1", prioMap["medium"])
	}
	if prioMap["low"] != 1 {
		t.Errorf("low count = %d, want 1", prioMap["low"])
	}
}

func TestSummaryStatusOrder(t *testing.T) {
	cfg := config.NewDefault("Test Board")
	s := Summary(cfg, nil, time.Now())

	boardStatuses := cfg.BoardStatuses()
	for i, ss := range s.Statuses {
		if ss.Status != boardStatuses[i] {
			t.Errorf("Statuses[%d] = %q, want %q", i, ss.Status, boardStatuses[i])
		}
	}
}
