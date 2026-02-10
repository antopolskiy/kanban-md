package output

import (
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// --- JSON marshal error ---

func TestJSON_MarshalError(t *testing.T) {
	var buf strings.Builder
	// Channels are not JSON-serializable.
	err := JSON(&buf, make(chan int))
	if err == nil {
		t.Fatal("expected error for non-serializable value")
	}
	if !strings.Contains(err.Error(), "encoding JSON") {
		t.Errorf("error = %v, want to contain 'encoding JSON'", err)
	}
}

// --- padRight edge cases ---

func TestPadRight_StringAtWidth(t *testing.T) {
	got := padRight("hello", 5)
	if got != "hello" {
		t.Errorf("padRight at width = %q, want %q", got, "hello")
	}
}

func TestPadRight_StringLongerThanWidth(t *testing.T) {
	got := padRight("long string here", 5)
	if got != "long string here" {
		t.Errorf("padRight longer = %q, want unchanged", got)
	}
}

// --- TaskDetail optional fields ---

func TestTaskDetail_WithClass(t *testing.T) {
	disableColorForTest(t)

	now := time.Now()
	tk := &task.Task{
		ID: 1, Title: "Class task", Status: "backlog", Priority: "medium",
		Class: "expedite", Created: now, Updated: now,
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "Class:") {
		t.Error("TaskDetail should show Class field")
	}
	if !strings.Contains(out, "expedite") {
		t.Errorf("TaskDetail missing class value:\n%s", out)
	}
}

func TestTaskDetail_WithClaimedBy(t *testing.T) {
	disableColorForTest(t)

	now := time.Now()
	claimedAt := time.Date(2026, 2, 8, 14, 0, 0, 0, time.UTC)
	tk := &task.Task{
		ID: 1, Title: "Claimed task", Status: "in-progress", Priority: "high",
		ClaimedBy: "test-agent", ClaimedAt: &claimedAt,
		Created: now, Updated: now,
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "Claimed by") {
		t.Error("TaskDetail should show Claimed by field")
	}
	if !strings.Contains(out, "test-agent") {
		t.Errorf("TaskDetail missing claimed_by value:\n%s", out)
	}
	if !strings.Contains(out, "since 2026-02-08") {
		t.Errorf("TaskDetail missing claimed_at date:\n%s", out)
	}
}

func TestTaskDetail_ClaimedWithoutClaimedAt(t *testing.T) {
	disableColorForTest(t)

	now := time.Now()
	tk := &task.Task{
		ID: 1, Title: "Claimed no date", Status: "backlog", Priority: "medium",
		ClaimedBy: "agent", Created: now, Updated: now,
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "Claimed by") {
		t.Error("TaskDetail should show Claimed by field")
	}
	if strings.Contains(out, "since") {
		t.Error("should not show 'since' without ClaimedAt")
	}
}

func TestTaskDetail_CompletedWithoutStarted(t *testing.T) {
	disableColorForTest(t)

	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)
	tk := &task.Task{
		ID: 1, Title: "Completed no started", Status: "done", Priority: "medium",
		Created: created, Updated: completed, Completed: &completed,
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "Lead time") {
		t.Error("completed task should show lead time")
	}
	if strings.Contains(out, "Cycle time") {
		t.Error("completed task without started should not show cycle time")
	}
}

func TestTaskDetail_NoBody(t *testing.T) {
	disableColorForTest(t)

	now := time.Now()
	tk := &task.Task{
		ID: 1, Title: "No body", Status: "backlog", Priority: "medium",
		Created: now, Updated: now, Body: "",
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	// No blank line + body after the metadata.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	lastLine := lines[len(lines)-1]
	if strings.TrimSpace(lastLine) == "" {
		t.Error("no trailing blank line for task without body")
	}
}

func TestTaskDetail_WithDue(t *testing.T) {
	disableColorForTest(t)

	now := time.Now()
	due := date.New(2026, time.March, 15)
	tk := &task.Task{
		ID: 1, Title: "Due task", Status: "backlog", Priority: "medium",
		Due: &due, Created: now, Updated: now,
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "Due:") {
		t.Error("TaskDetail should show Due field")
	}
	if !strings.Contains(out, "2026-03-15") {
		t.Errorf("TaskDetail missing due date:\n%s", out)
	}
}

func TestTaskDetail_NoTags(t *testing.T) {
	disableColorForTest(t)

	now := time.Now()
	tk := &task.Task{
		ID: 1, Title: "No tags", Status: "backlog", Priority: "medium",
		Created: now, Updated: now,
	}

	var buf strings.Builder
	TaskDetail(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "Tags:") {
		t.Error("TaskDetail should show Tags field even when empty")
	}
}

// --- TaskTable edge cases ---

func TestTaskTable_TitleTruncation(t *testing.T) {
	disableColorForTest(t)

	now := time.Now()
	longTitle := strings.Repeat("A", 60)
	tasks := []*task.Task{
		{ID: 1, Title: longTitle, Status: "backlog", Priority: "medium", Created: now, Updated: now},
	}

	var buf strings.Builder
	TaskTable(&buf, tasks)
	out := buf.String()

	if strings.Contains(out, longTitle) {
		t.Error("long title should be truncated")
	}
	if !strings.Contains(out, "...") {
		t.Error("truncated title should end with ...")
	}
}

func TestTaskTable_WithDueAndClaim(t *testing.T) {
	disableColorForTest(t)

	now := time.Now()
	due := date.New(2026, time.June, 1)
	tasks := []*task.Task{
		{
			ID: 1, Title: "Full task", Status: "in-progress", Priority: "high",
			ClaimedBy: "agent", Tags: []string{"feature"}, Due: &due,
			Created: now, Updated: now,
		},
	}

	var buf strings.Builder
	TaskTable(&buf, tasks)
	out := buf.String()

	if !strings.Contains(out, "@agent") {
		t.Errorf("TaskTable missing claimed_by:\n%s", out)
	}
	if !strings.Contains(out, "2026-06-01") {
		t.Errorf("TaskTable missing due date:\n%s", out)
	}
}

// --- MetricsTable aging title truncation ---

func TestMetricsTable_AgingTitleTruncation(t *testing.T) {
	disableColorForTest(t)

	longTitle := strings.Repeat("B", 50)
	metrics := board.Metrics{
		Throughput7d:  1,
		Throughput30d: 5,
		AgingItems: []board.AgingItem{
			{ID: 1, Title: longTitle, Status: "in-progress", AgeHours: 48},
		},
	}

	var buf strings.Builder
	MetricsTable(&buf, metrics)
	out := buf.String()

	if strings.Contains(out, longTitle) {
		t.Error("aging title should be truncated")
	}
	if !strings.Contains(out, "...") {
		t.Error("truncated aging title should end with ...")
	}
}

// --- MetricsCompact aging title truncation ---

func TestMetricsCompact_AgingTitleTruncation(t *testing.T) {
	longTitle := strings.Repeat("C", 70)
	metrics := board.Metrics{
		Throughput7d:  0,
		Throughput30d: 0,
		AgingItems: []board.AgingItem{
			{ID: 1, Title: longTitle, Status: "backlog", AgeHours: 96},
		},
	}

	var buf strings.Builder
	MetricsCompact(&buf, metrics)
	out := buf.String()

	if strings.Contains(out, longTitle) {
		t.Error("aging title should be truncated")
	}
	if !strings.Contains(out, "...") {
		t.Error("truncated aging title should end with ...")
	}
}

// --- TaskDetailCompact with completed ---

func TestTaskDetailCompact_WithCompleted(t *testing.T) {
	now := time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	tk := &task.Task{
		ID: 1, Title: "Done task", Status: "done", Priority: "medium",
		Created: now, Updated: completed, Completed: &completed,
	}

	var buf strings.Builder
	TaskDetailCompact(&buf, tk)
	out := buf.String()

	if !strings.Contains(out, "completed:2026-02-10") {
		t.Errorf("TaskDetailCompact missing completed timestamp:\n%s", out)
	}
}

// --- OverviewTable with classes ---

func TestOverviewTable_WithClasses(t *testing.T) {
	disableColorForTest(t)

	overview := board.Overview{
		BoardName:  "Board",
		TotalTasks: 5,
		Statuses: []board.StatusSummary{
			{Status: "backlog", Count: 5},
		},
		Classes: []board.ClassCount{
			{Class: "expedite", Count: 1},
			{Class: "standard", Count: 4},
		},
	}

	var buf strings.Builder
	OverviewTable(&buf, overview)
	out := buf.String()

	if !strings.Contains(out, "CLASS") {
		t.Error("OverviewTable with classes should show CLASS header")
	}
	if !strings.Contains(out, "expedite") {
		t.Errorf("OverviewTable missing class:\n%s", out)
	}
}

// --- formatOptionalHours / formatOptionalPercent ---

func TestFormatOptionalHours_Nil(t *testing.T) {
	got := formatOptionalHours(nil)
	if got == "" {
		t.Error("expected non-empty for nil (should show dash)")
	}
}

func TestFormatOptionalHours_Value(t *testing.T) {
	hours := float64(48)
	got := formatOptionalHours(&hours)
	if !strings.Contains(got, "2d") {
		t.Errorf("formatOptionalHours(48h) = %q, want to contain '2d'", got)
	}
}

func TestFormatOptionalPercent_Nil(t *testing.T) {
	got := formatOptionalPercent(nil)
	if got == "" {
		t.Error("expected non-empty for nil (should show dash)")
	}
}

func TestFormatOptionalPercent_Value(t *testing.T) {
	pct := 0.75
	got := formatOptionalPercent(&pct)
	if got != "75.0%" {
		t.Errorf("formatOptionalPercent(0.75) = %q, want %q", got, "75.0%")
	}
}

// --- stringOrDash ---

func TestStringOrDash_Empty(t *testing.T) {
	got := stringOrDash("")
	if got == "" {
		t.Error("expected non-empty for empty string (should show dash)")
	}
}

func TestStringOrDash_NonEmpty(t *testing.T) {
	got := stringOrDash("hello")
	if got != "hello" {
		t.Errorf("stringOrDash('hello') = %q, want %q", got, "hello")
	}
}

// --- claimDisplay ---

func TestClaimDisplay_Empty(t *testing.T) {
	tk := &task.Task{ID: 1}
	got := claimDisplay(tk)
	if got != "" {
		t.Errorf("claimDisplay(unclaimed) = %q, want empty", got)
	}
}

func TestClaimDisplay_WithClaim(t *testing.T) {
	tk := &task.Task{ID: 1, ClaimedBy: "agent"}
	got := claimDisplay(tk)
	if got != "@agent" {
		t.Errorf("claimDisplay = %q, want %q", got, "@agent")
	}
}

// --- compactDuration ---

func TestCompactDuration_Nil(t *testing.T) {
	got := compactDuration(nil)
	if got != "--" {
		t.Errorf("compactDuration(nil) = %q, want %q", got, "--")
	}
}

func TestCompactDuration_Value(t *testing.T) {
	h := float64(48)
	got := compactDuration(&h)
	if !strings.Contains(got, "2d") {
		t.Errorf("compactDuration(48) = %q, want to contain '2d'", got)
	}
}
