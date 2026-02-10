package board

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ---------------------------------------------------------------------------
// AppendLog — OpenFile error (nonexistent nested path)
// ---------------------------------------------------------------------------

func TestAppendLog_OpenFileError(t *testing.T) {
	// Use a path inside a file to trigger OpenFile failure.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	entry := LogEntry{Timestamp: time.Now(), Action: "create", TaskID: 1, Detail: "test"}
	err := AppendLog(filepath.Join(blocker, "subdir"), entry)
	if err == nil {
		t.Error("expected error when log directory path is invalid")
	}
	if !strings.Contains(err.Error(), "opening log file") {
		t.Errorf("error = %v, want to contain 'opening log file'", err)
	}
}

// ---------------------------------------------------------------------------
// truncateLogIfNeeded — file open error
// ---------------------------------------------------------------------------

func TestTruncateLogIfNeeded_FileNotFound(t *testing.T) {
	err := truncateLogIfNeeded("/nonexistent/path/activity.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// ---------------------------------------------------------------------------
// truncateLogIfNeeded — scanner.Err() path (simulate with very long line)
// ---------------------------------------------------------------------------

func TestTruncateLogIfNeeded_ScannerError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)

	// Write a line that exceeds the default bufio.Scanner buffer (64K).
	const lineLen = 128 * 1024
	longLine := strings.Repeat("x", lineLen)
	if err := os.WriteFile(logPath, []byte(longLine+"\n"), logFileMode); err != nil {
		t.Fatal(err)
	}

	err := truncateLogIfNeeded(logPath)
	if err == nil {
		t.Error("expected scanner error for oversized line")
	}
}

// ---------------------------------------------------------------------------
// ReadLog — limit edge cases (0, 1, empty log)
// ---------------------------------------------------------------------------

func TestReadLog_LimitZero(t *testing.T) {
	dir := t.TempDir()
	const numEntries = 5
	for i := 1; i <= numEntries; i++ {
		mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "create", TaskID: i, Detail: "x"})
	}

	// Limit 0 means no limit — should return all entries.
	entries, err := ReadLog(dir, LogFilterOptions{Limit: 0})
	if err != nil {
		t.Fatalf("ReadLog error: %v", err)
	}
	if len(entries) != numEntries {
		t.Errorf("got %d entries, want %d (limit 0 = no limit)", len(entries), numEntries)
	}
}

func TestReadLog_LimitOne(t *testing.T) {
	dir := t.TempDir()
	const numEntries = 3
	for i := 1; i <= numEntries; i++ {
		mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "create", TaskID: i, Detail: "x"})
	}

	entries, err := ReadLog(dir, LogFilterOptions{Limit: 1})
	if err != nil {
		t.Fatalf("ReadLog error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("got %d entries, want 1", len(entries))
	}
	// Should return the last entry.
	if entries[0].TaskID != numEntries {
		t.Errorf("TaskID = %d, want %d (last entry)", entries[0].TaskID, numEntries)
	}
}

func TestReadLog_EmptyLogFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)
	if err := os.WriteFile(logPath, []byte(""), logFileMode); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadLog(dir, LogFilterOptions{})
	if err != nil {
		t.Fatalf("ReadLog error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0 for empty log", len(entries))
	}
}

func TestReadLog_LimitGreaterThanEntries(t *testing.T) {
	dir := t.TempDir()
	mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "create", TaskID: 1, Detail: "x"})

	const largeLimit = 100
	// Limit larger than entry count should return all.
	entries, err := ReadLog(dir, LogFilterOptions{Limit: largeLimit})
	if err != nil {
		t.Fatalf("ReadLog error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("got %d entries, want 1", len(entries))
	}
}

// ---------------------------------------------------------------------------
// ReadLog — unreadable log file
// ---------------------------------------------------------------------------

func TestReadLog_UnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not prevent reads on Windows")
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)
	if err := os.WriteFile(logPath, []byte(`{"action":"create"}`+"\n"), logFileMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(logPath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(logPath, 0o600) })

	_, err := ReadLog(dir, LogFilterOptions{})
	if err == nil {
		t.Error("expected error for unreadable log file")
	}
}

// ---------------------------------------------------------------------------
// sortPickCandidates — fixed-date class without due date
// ---------------------------------------------------------------------------

func TestSortPickCandidates_FixedDateNoDue(t *testing.T) {
	cfg := newPickTestConfig()
	candidates := [2]*task.Task{
		{ID: 1, Status: "todo", Priority: "low", Class: "fixed-date"},
		{ID: 2, Status: "todo", Priority: "high", Class: "fixed-date"},
	}
	slice := candidates[:]

	sortPickCandidates(slice, cfg)
	// Both fixed-date with no due -> fallback to priority. High priority comes first.
	if len(slice) < 1 {
		t.Fatal("no candidates after sort")
	}
	if slice[0].ID != 2 {
		t.Errorf("first candidate ID = %d, want 2 (higher priority)", slice[0].ID)
	}
}

func TestSortPickCandidates_FixedDateOneDueOneNot(t *testing.T) {
	cfg := newPickTestConfig()
	due := parseTestDate(t, "2026-03-01")
	candidates := [2]*task.Task{
		{ID: 1, Status: "todo", Priority: "high", Class: "fixed-date"},
		{ID: 2, Status: "todo", Priority: "low", Class: "fixed-date", Due: due},
	}
	slice := candidates[:]

	sortPickCandidates(slice, cfg)
	// Task 2 has a due date, task 1 doesn't (nil sorts last in compareDue).
	// So task 2 should come first.
	if len(slice) < 1 {
		t.Fatal("no candidates after sort")
	}
	if slice[0].ID != 2 {
		t.Errorf("first candidate ID = %d, want 2 (has due date)", slice[0].ID)
	}
}

// ---------------------------------------------------------------------------
// List — FilterUnblocked with Limit > result set
// ---------------------------------------------------------------------------

func TestList_LimitGreaterThanResults(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefault("Test Board")
	cfg.SetDir(dir)

	now := time.Now()
	writeTestTask(t, tasksDir, &task.Task{
		ID: 1, Title: "Only task", Status: "backlog", Priority: "medium",
		Created: now, Updated: now,
	})

	const largeLimit = 100
	// Limit 100, but only 1 task exists — should get 1 back.
	tasks, _, err := List(cfg, ListOptions{Limit: largeLimit})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("got %d tasks, want 1", len(tasks))
	}
}

// ---------------------------------------------------------------------------
// FindDependents — empty DependsOn and multiple deps
// ---------------------------------------------------------------------------

func TestFindDependents_EmptyDependsOn(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	writeTestTask(t, dir, &task.Task{
		ID: 1, Title: "Target", Status: "todo", Priority: "medium",
		Created: now, Updated: now,
	})
	writeTestTask(t, dir, &task.Task{
		ID: 2, Title: "No deps", Status: "todo", Priority: "medium",
		DependsOn: nil, Created: now, Updated: now,
	})

	msgs := FindDependents(dir, 1)
	if len(msgs) != 0 {
		t.Errorf("expected no dependents, got %d: %v", len(msgs), msgs)
	}
}

func TestFindDependents_MultipleDeps(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	writeTestTask(t, dir, &task.Task{
		ID: 1, Title: "Target", Status: "todo", Priority: "medium",
		Created: now, Updated: now,
	})
	// Task 2 depends on both 1 and 3 — should only produce one message for dep on 1.
	writeTestTask(t, dir, &task.Task{
		ID: 2, Title: "Multi-dep", Status: "todo", Priority: "medium",
		DependsOn: []int{1, 3}, Created: now, Updated: now,
	})
	writeTestTask(t, dir, &task.Task{
		ID: 3, Title: "Other", Status: "todo", Priority: "medium",
		Created: now, Updated: now,
	})

	msgs := FindDependents(dir, 1)
	if len(msgs) != 1 {
		t.Errorf("got %d messages, want 1 (only one task depends on #1)", len(msgs))
	}
}

// ---------------------------------------------------------------------------
// FilterUnblocked — tasks with no deps always included
// ---------------------------------------------------------------------------

func TestFilterUnblocked_NoDepsAlwaysIncluded(t *testing.T) {
	cfg := config.NewDefault("Test")
	tasks := []*task.Task{
		{ID: 1, Status: "todo", Priority: "medium"},
		{ID: 2, Status: "backlog", Priority: "low"},
	}

	result := FilterUnblocked(tasks, cfg)
	if len(result) != 2 {
		t.Errorf("got %d tasks, want 2 (all tasks have no deps)", len(result))
	}
}

// ---------------------------------------------------------------------------
// FilterUnblockedWithLookup — dep in archived status
// ---------------------------------------------------------------------------

func TestFilterUnblockedWithLookup_ArchivedDep(t *testing.T) {
	cfg := config.NewDefault("Test")
	allTasks := []*task.Task{
		{ID: 1, Status: "todo", DependsOn: []int{2}},
		{ID: 2, Status: "archived"}, // terminal
	}
	candidates := []*task.Task{allTasks[0]}

	result := FilterUnblockedWithLookup(candidates, allTasks, cfg)
	if len(result) != 1 {
		t.Errorf("got %d tasks, want 1 (dep is archived = terminal)", len(result))
	}
}

// ---------------------------------------------------------------------------
// CountByStatus
// ---------------------------------------------------------------------------

func TestCountByStatus_Empty(t *testing.T) {
	counts := CountByStatus(nil)
	if len(counts) != 0 {
		t.Errorf("got %d counts, want 0 for nil input", len(counts))
	}
}

func TestCountByStatus_MultipleTasks(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Status: "todo"},
		{ID: 2, Status: "todo"},
		{ID: 3, Status: "done"},
	}
	counts := CountByStatus(tasks)
	if counts["todo"] != 2 {
		t.Errorf("todo count = %d, want 2", counts["todo"])
	}
	if counts["done"] != 1 {
		t.Errorf("done count = %d, want 1", counts["done"])
	}
}

// ---------------------------------------------------------------------------
// ParseIDs — negative ID
// ---------------------------------------------------------------------------

func TestParseIDs_NegativeID(t *testing.T) {
	// Negative numbers are technically valid integers, but ParseIDs allows them.
	ids, err := ParseIDs("-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != -1 {
		t.Errorf("ids = %v, want [-1]", ids)
	}
}

// ---------------------------------------------------------------------------
// matchesLogFilter — combined action + taskID filter
// ---------------------------------------------------------------------------

func TestMatchesLogFilter_AllFiltersMatch(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		Action:    "move",
		TaskID:    42,
	}
	opts := LogFilterOptions{
		Since:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Action: "move",
		TaskID: 42,
	}
	if !matchesLogFilter(entry, opts) {
		t.Error("entry should match all filters")
	}
}

func TestMatchesLogFilter_ActionMismatch(t *testing.T) {
	entry := LogEntry{Action: "create", TaskID: 1}
	opts := LogFilterOptions{Action: "move"}
	if matchesLogFilter(entry, opts) {
		t.Error("entry should not match mismatched action")
	}
}

func TestMatchesLogFilter_TaskIDMismatch(t *testing.T) {
	entry := LogEntry{Action: "create", TaskID: 1}
	opts := LogFilterOptions{TaskID: 2}
	if matchesLogFilter(entry, opts) {
		t.Error("entry should not match mismatched taskID")
	}
}
