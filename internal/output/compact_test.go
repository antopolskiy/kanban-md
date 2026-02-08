package output

import (
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestTaskCompactAllFields(t *testing.T) {
	now := time.Now()
	due := date.New(2026, time.March, 1)
	tasks := []*task.Task{
		{
			ID: 3, Title: "Add WIP limits", Status: "backlog", Priority: "high",
			Assignee: "alice", Tags: []string{"layer-4"}, Due: &due,
			Created: now, Updated: now,
		},
	}

	var buf strings.Builder
	TaskCompact(&buf, tasks)

	want := "#3 [backlog/high] Add WIP limits @alice (layer-4) due:2026-03-01\n"
	if buf.String() != want {
		t.Errorf("TaskCompact =\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestTaskCompactNoOptionalFields(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: 7, Title: "Simple task", Status: "todo", Priority: "medium", Created: now, Updated: now},
	}

	var buf strings.Builder
	TaskCompact(&buf, tasks)

	want := "#7 [todo/medium] Simple task\n"
	if buf.String() != want {
		t.Errorf("TaskCompact =\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestTaskCompactNoTags(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{
			ID: 1, Title: "No tags task", Status: "backlog", Priority: "low",
			Assignee: "bob", Created: now, Updated: now,
		},
	}

	var buf strings.Builder
	TaskCompact(&buf, tasks)

	out := buf.String()
	if strings.Contains(out, "(") {
		t.Errorf("TaskCompact with no tags should not contain parentheses: %s", out)
	}
	if !strings.Contains(out, "@bob") {
		t.Errorf("TaskCompact missing assignee: %s", out)
	}
}

func TestTaskCompactNoDue(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{
			ID: 1, Title: "No due", Status: "todo", Priority: "medium",
			Tags: []string{"bug"}, Created: now, Updated: now,
		},
	}

	var buf strings.Builder
	TaskCompact(&buf, tasks)

	out := buf.String()
	if strings.Contains(out, "due:") {
		t.Errorf("TaskCompact with no due should not contain due: %s", out)
	}
	if !strings.Contains(out, "(bug)") {
		t.Errorf("TaskCompact missing tags: %s", out)
	}
}

func TestTaskCompactEmpty(t *testing.T) {
	var buf strings.Builder
	TaskCompact(&buf, nil)

	if buf.String() != "" {
		t.Errorf("TaskCompact empty output to writer = %q, want empty", buf.String())
	}
}

func TestTaskCompactMultiple(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Title: "First", Status: "backlog", Priority: "high", Created: now, Updated: now},
		{ID: 2, Title: "Second", Status: "todo", Priority: "low", Created: now, Updated: now},
		{ID: 3, Title: "Third", Status: "done", Priority: "medium", Created: now, Updated: now},
	}

	var buf strings.Builder
	TaskCompact(&buf, tasks)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	const expectedLines = 3
	if len(lines) != expectedLines {
		t.Fatalf("expected 3 lines, got %d: %s", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "#1 ") {
		t.Errorf("first line should start with #1: %s", lines[0])
	}
	if !strings.HasPrefix(lines[2], "#3 ") {
		t.Errorf("third line should start with #3: %s", lines[2])
	}
}

func TestTaskDetailCompact(t *testing.T) {
	now := time.Date(2026, 2, 8, 14, 30, 0, 0, time.UTC)
	started := time.Date(2026, 2, 7, 10, 0, 0, 0, time.UTC)
	due := date.New(2026, time.March, 1)
	tk := &task.Task{
		ID: 5, Title: "Test detail", Status: "in-progress", Priority: "high",
		Assignee: "alice", Tags: []string{"feature"}, Due: &due,
		Estimate: "4h", Created: now.Add(-24 * time.Hour), Updated: now,
		Started: &started, Body: "Body text here.",
	}

	var buf strings.Builder
	TaskDetailCompact(&buf, tk)
	out := buf.String()

	for _, want := range []string{
		"#5 [in-progress/high] Test detail @alice (feature) due:2026-03-01 est:4h",
		"created:2026-02-07",
		"updated:2026-02-08",
		"started:2026-02-07",
		"  Body text here.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("TaskDetailCompact missing %q in:\n%s", want, out)
		}
	}
}

func TestTaskDetailCompactMinimal(t *testing.T) {
	now := time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)
	tk := &task.Task{
		ID: 1, Title: "Minimal", Status: "backlog", Priority: "low",
		Created: now, Updated: now,
	}

	var buf strings.Builder
	TaskDetailCompact(&buf, tk)
	out := buf.String()

	if strings.Contains(out, "est:") {
		t.Errorf("minimal task should not contain est: %s", out)
	}
	if strings.Contains(out, "started:") {
		t.Errorf("minimal task should not contain started: %s", out)
	}
	const expectedNewlines = 2 // header + timestamps
	if strings.Count(out, "\n") != expectedNewlines {
		t.Errorf("minimal task should have 2 lines, got:\n%s", out)
	}
}

func TestTaskDetailCompactWithBody(t *testing.T) {
	now := time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)
	tk := &task.Task{
		ID: 1, Title: "With body", Status: "todo", Priority: "medium",
		Created: now, Updated: now, Body: "Line one.\nLine two.",
	}

	var buf strings.Builder
	TaskDetailCompact(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "  Line one.") {
		t.Errorf("body line 1 should be indented: %s", out)
	}
	if !strings.Contains(out, "  Line two.") {
		t.Errorf("body line 2 should be indented: %s", out)
	}
}

func TestOverviewCompact(t *testing.T) {
	overview := board.Overview{
		BoardName:  "Test Board",
		TotalTasks: 10,
		Statuses: []board.StatusSummary{
			{Status: "backlog", Count: 5, Blocked: 1},
			{Status: "in-progress", Count: 3, WIPLimit: 4, Overdue: 1},
			{Status: "done", Count: 2},
		},
		Priorities: []board.PriorityCount{
			{Priority: "high", Count: 4},
			{Priority: "low", Count: 6},
		},
	}

	var buf strings.Builder
	OverviewCompact(&buf, overview)
	out := buf.String()

	for _, want := range []string{
		"Test Board (10 tasks)",
		"  backlog: 5 (1 blocked)",
		"  in-progress: 3/4 (1 overdue)",
		"  done: 2",
		"Priority: high=4 low=6",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("OverviewCompact missing %q in:\n%s", want, out)
		}
	}
}

func TestOverviewCompactNoPriorities(t *testing.T) {
	overview := board.Overview{
		BoardName:  "Empty Board",
		TotalTasks: 0,
	}

	var buf strings.Builder
	OverviewCompact(&buf, overview)
	out := buf.String()

	if !strings.Contains(out, "Empty Board (0 tasks)") {
		t.Errorf("OverviewCompact missing header: %s", out)
	}
	if strings.Contains(out, "Priority:") {
		t.Errorf("OverviewCompact should omit Priority line when empty: %s", out)
	}
}

func TestMetricsCompact(t *testing.T) {
	leadTime := float64(48)
	cycleTime := float64(24)
	efficiency := 0.5
	metrics := board.Metrics{
		Throughput7d:      3,
		Throughput30d:     12,
		AvgLeadTimeHours:  &leadTime,
		AvgCycleTimeHours: &cycleTime,
		FlowEfficiency:    &efficiency,
		AgingItems: []board.AgingItem{
			{ID: 1, Title: "Aging task", Status: "in-progress", AgeHours: 72},
		},
	}

	var buf strings.Builder
	MetricsCompact(&buf, metrics)
	out := buf.String()

	for _, want := range []string{
		"Throughput: 3/7d 12/30d",
		"Lead: 2d 0h",
		"Cycle: 1d 0h",
		"50.0%",
		"Aging: #1 [in-progress] Aging task (3d 0h)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("MetricsCompact missing %q in:\n%s", want, out)
		}
	}
}

func TestMetricsCompactNoAging(t *testing.T) {
	metrics := board.Metrics{
		Throughput7d:  0,
		Throughput30d: 0,
	}

	var buf strings.Builder
	MetricsCompact(&buf, metrics)
	out := buf.String()

	if strings.Contains(out, "Aging:") {
		t.Errorf("MetricsCompact without aging should not show Aging line: %s", out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("MetricsCompact no aging should be 1 line, got:\n%s", out)
	}
}

func TestMetricsCompactNilValues(t *testing.T) {
	metrics := board.Metrics{
		Throughput7d:  0,
		Throughput30d: 0,
	}

	var buf strings.Builder
	MetricsCompact(&buf, metrics)
	out := buf.String()

	if !strings.Contains(out, "Lead: --") {
		t.Errorf("nil lead time should show '--': %s", out)
	}
	if !strings.Contains(out, "Cycle: --") {
		t.Errorf("nil cycle time should show '--': %s", out)
	}
	if !strings.Contains(out, "Efficiency: --") {
		t.Errorf("nil efficiency should show '--': %s", out)
	}
}

func TestActivityLogCompact(t *testing.T) {
	entries := []board.LogEntry{
		{
			Timestamp: time.Date(2026, 2, 8, 12, 0, 5, 0, time.UTC),
			Action:    "create",
			TaskID:    1,
			Detail:    "New task",
		},
		{
			Timestamp: time.Date(2026, 2, 8, 13, 0, 0, 0, time.UTC),
			Action:    "move",
			TaskID:    1,
			Detail:    "backlog -> todo",
		},
	}

	var buf strings.Builder
	ActivityLogCompact(&buf, entries)
	out := buf.String()

	for _, want := range []string{
		"2026-02-08 12:00:05 create #1 New task",
		"2026-02-08 13:00:00 move #1 backlog -> todo",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ActivityLogCompact missing %q in:\n%s", want, out)
		}
	}
}

func TestActivityLogCompactEmpty(t *testing.T) {
	var buf strings.Builder
	ActivityLogCompact(&buf, nil)

	if buf.String() != "" {
		t.Errorf("ActivityLogCompact empty output to writer = %q, want empty", buf.String())
	}
}
