package output

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestTaskTableWritesToWriter(t *testing.T) {
	DisableColor()
	t.Cleanup(func() {
		headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	})

	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Title: "Test task", Status: "backlog", Priority: "medium", Created: now, Updated: now},
	}

	var buf strings.Builder
	TaskTable(&buf, tasks)

	output := buf.String()
	if !strings.Contains(output, "Test task") {
		t.Errorf("TaskTable output missing task title:\n%s", output)
	}
	if !strings.Contains(output, "ID") {
		t.Errorf("TaskTable output missing header:\n%s", output)
	}
}

func TestTaskTableEmptyWritesNothing(t *testing.T) {
	var buf strings.Builder
	TaskTable(&buf, nil)
	// "No tasks found." is written to stderr, not the writer.
	if buf.String() != "" {
		t.Errorf("TaskTable empty output to writer = %q, want empty", buf.String())
	}
}

func TestMessagefWritesToWriter(t *testing.T) {
	var buf strings.Builder
	Messagef(&buf, "hello %s", "world")
	if buf.String() != "hello world\n" {
		t.Errorf("Messagef output = %q, want %q", buf.String(), "hello world\n")
	}
}

func TestJSONWritesToWriter(t *testing.T) {
	var buf strings.Builder
	err := JSON(&buf, map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"key": "value"`) {
		t.Errorf("JSON output missing content:\n%s", buf.String())
	}
}

func TestJSONErrorWritesToWriter(t *testing.T) {
	var buf strings.Builder
	JSONError(&buf, "TEST_CODE", "test message", nil)
	if !strings.Contains(buf.String(), `"code": "TEST_CODE"`) {
		t.Errorf("JSONError output missing code:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), `"error": "test message"`) {
		t.Errorf("JSONError output missing error:\n%s", buf.String())
	}
}

func TestTaskTableColumnAlignment(t *testing.T) {
	// Force ANSI color output even in non-TTY (test) environments.
	// This is critical to catch the bug where %-*s counts ANSI bytes as width.
	oldHeader, oldDim := headerStyle, dimStyle
	t.Cleanup(func() {
		headerStyle = oldHeader
		dimStyle = oldDim
	})
	lipgloss.SetColorProfile(termenv.ANSI256)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	now := time.Now()
	due := date.New(2025, 6, 15)

	tasks := []*task.Task{
		{
			ID: 1, Title: "Task with all fields", Status: "in-progress",
			Priority: "high", Assignee: "alice", Tags: []string{"feature"},
			Due: &due, Created: now, Updated: now,
		},
		{
			ID: 2, Title: "Task with empty fields", Status: "backlog",
			Priority: "medium", Assignee: "", Tags: nil,
			Due: nil, Created: now, Updated: now,
		},
		{
			ID: 3, Title: "Another task", Status: "todo",
			Priority: "low", Assignee: "bob", Tags: []string{"bug", "urgent"},
			Due: &due, Created: now, Updated: now,
		},
	}

	var buf strings.Builder
	TaskTable(&buf, tasks)
	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	const expectedMinLines = 4 // header + 3 data rows
	if len(lines) < expectedMinLines {
		t.Fatalf("expected at least 4 lines, got %d:\n%s", len(lines), output)
	}

	row1Width := lipgloss.Width(lines[1])
	row2Width := lipgloss.Width(lines[2])
	row3Width := lipgloss.Width(lines[3])

	const maxDrift = 3 // allow tiny rounding differences
	if abs(row1Width-row2Width) > maxDrift {
		t.Errorf("column misalignment: row 1 visible width = %d, row 2 visible width = %d (drift > %d)\nrow1: %s\nrow2: %s",
			row1Width, row2Width, maxDrift, lines[1], lines[2])
	}
	if abs(row1Width-row3Width) > maxDrift {
		t.Errorf("column misalignment: row 1 visible width = %d, row 3 visible width = %d (drift > %d)\nrow1: %s\nrow3: %s",
			row1Width, row3Width, maxDrift, lines[1], lines[3])
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestFormatDuration_Days(t *testing.T) {
	d := 50 * time.Hour
	got := FormatDuration(d)
	if got != "2d 2h" {
		t.Errorf("FormatDuration(50h) = %q, want %q", got, "2d 2h")
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	d := 3*time.Hour + 30*time.Minute
	got := FormatDuration(d)
	if got != "3h 30m" {
		t.Errorf("FormatDuration(3h30m) = %q, want %q", got, "3h 30m")
	}
}

func TestFormatDuration_Zero(t *testing.T) {
	got := FormatDuration(0)
	if got != "0h 0m" {
		t.Errorf("FormatDuration(0) = %q, want %q", got, "0h 0m")
	}
}

func TestFormatDuration_ExactDays(t *testing.T) {
	d := 48 * time.Hour
	got := FormatDuration(d)
	if got != "2d 0h" {
		t.Errorf("FormatDuration(48h) = %q, want %q", got, "2d 0h")
	}
}

func TestTaskDetail(t *testing.T) {
	DisableColor()
	t.Cleanup(func() {
		headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	})

	now := time.Date(2026, 2, 8, 14, 30, 0, 0, time.UTC)
	started := time.Date(2026, 2, 7, 10, 0, 0, 0, time.UTC)
	due := date.New(2026, time.March, 1)
	tk := &task.Task{
		ID:       5,
		Title:    "Test task detail",
		Status:   "in-progress",
		Priority: "high",
		Assignee: "alice",
		Tags:     []string{"feature", "urgent"},
		Due:      &due,
		Estimate: "4h",
		Created:  now.Add(-24 * time.Hour),
		Updated:  now,
		Started:  &started,
		Body:     "This is the body.",
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	for _, want := range []string{
		"Task #5: Test task detail",
		"Status:      in-progress",
		"Priority:    high",
		"Assignee:    alice",
		"Tags:        feature, urgent",
		"Estimate:    4h",
		"This is the body.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("TaskDetail missing %q in output:\n%s", want, out)
		}
	}
}

func TestTaskDetailCompleted(t *testing.T) {
	DisableColor()
	t.Cleanup(func() {
		headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	})

	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	started := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)
	tk := &task.Task{
		ID: 1, Title: "Done task", Status: "done", Priority: "medium",
		Created: created, Updated: completed, Started: &started, Completed: &completed,
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "Lead time") {
		t.Error("completed task should show lead time")
	}
	if !strings.Contains(out, "Cycle time") {
		t.Error("completed task with started should show cycle time")
	}
}

func TestOverviewTable(t *testing.T) {
	DisableColor()
	t.Cleanup(func() {
		headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	})

	overview := board.Overview{
		BoardName:  "Test Board",
		TotalTasks: 10,
		Statuses: []board.StatusSummary{
			{Status: "backlog", Count: 5, Blocked: 1, Overdue: 0},
			{Status: "in-progress", Count: 3, WIPLimit: 4, Blocked: 0, Overdue: 1},
			{Status: "done", Count: 2, Blocked: 0, Overdue: 0},
		},
		Priorities: []board.PriorityCount{
			{Priority: "high", Count: 4},
			{Priority: "low", Count: 6},
		},
	}

	var buf strings.Builder
	OverviewTable(&buf, overview)
	out := buf.String()

	for _, want := range []string{
		"Test Board",
		"Total: 10 tasks",
		"STATUS",
		"backlog",
		"in-progress",
		"3/4", // WIP limit display
		"PRIORITY",
		"high",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("OverviewTable missing %q in output:\n%s", want, out)
		}
	}
}

func TestMetricsTable(t *testing.T) {
	DisableColor()
	t.Cleanup(func() {
		headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	})

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
	MetricsTable(&buf, metrics)
	out := buf.String()

	for _, want := range []string{
		"Flow Metrics",
		"3 tasks",
		"12 tasks",
		"50.0%",
		"Aging task",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("MetricsTable missing %q in output:\n%s", want, out)
		}
	}
}

func TestMetricsTableNoAging(t *testing.T) {
	DisableColor()
	t.Cleanup(func() {
		headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	})

	metrics := board.Metrics{
		Throughput7d:  0,
		Throughput30d: 0,
	}

	var buf strings.Builder
	MetricsTable(&buf, metrics)
	out := buf.String()

	if strings.Contains(out, "AGE") {
		t.Error("MetricsTable without aging items should not show aging header")
	}
}

func TestActivityLogTable(t *testing.T) {
	DisableColor()
	t.Cleanup(func() {
		headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	})

	entries := []board.LogEntry{
		{
			Timestamp: time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC),
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
	ActivityLogTable(&buf, entries)
	out := buf.String()

	for _, want := range []string{
		"TIMESTAMP",
		"ACTION",
		"create",
		"move",
		"New task",
		"backlog -> todo",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ActivityLogTable missing %q in output:\n%s", want, out)
		}
	}
}

func TestActivityLogTableEmpty(t *testing.T) {
	var buf strings.Builder
	ActivityLogTable(&buf, nil)
	// "No activity log entries found." is written to stderr, not the writer.
	if buf.String() != "" {
		t.Errorf("ActivityLogTable empty output to writer = %q, want empty", buf.String())
	}
}
