package board

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ---------------------------------------------------------------------------
// ParseIDs
// ---------------------------------------------------------------------------

func TestParseIDsSingle(t *testing.T) {
	ids, err := ParseIDs("42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 42 {
		t.Errorf("ids = %v, want [42]", ids)
	}
}

func TestParseIDsMultiple(t *testing.T) {
	ids, err := ParseIDs("1,2,3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [3]int{1, 2, 3}
	if len(ids) != 3 {
		t.Fatalf("len = %d, want 3", len(ids))
	}
	for i, id := range ids {
		if id != want[i] {
			t.Errorf("ids[%d] = %d, want %d", i, id, want[i])
		}
	}
}

func TestParseIDsDeduplicates(t *testing.T) {
	ids, err := ParseIDs("1,2,1,3,2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [3]int{1, 2, 3}
	if len(ids) != 3 {
		t.Fatalf("len = %d, want 3", len(ids))
	}
	for i, id := range ids {
		if id != want[i] {
			t.Errorf("ids[%d] = %d, want %d", i, id, want[i])
		}
	}
}

func TestParseIDsTrimsWhitespace(t *testing.T) {
	ids, err := ParseIDs(" 1 , 2 , 3 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("len = %d, want 3", len(ids))
	}
}

func TestParseIDsSkipsEmptyElements(t *testing.T) {
	ids, err := ParseIDs("1,,2,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("len = %d, want 2", len(ids))
	}
}

func TestParseIDsErrorOnInvalid(t *testing.T) {
	_, err := ParseIDs("abc")
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
}

func TestParseIDsErrorOnEmpty(t *testing.T) {
	_, err := ParseIDs("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseIDsErrorOnOnlyCommas(t *testing.T) {
	_, err := ParseIDs(",,,")
	if err == nil {
		t.Fatal("expected error for only commas")
	}
}

// ---------------------------------------------------------------------------
// FindDependents
// ---------------------------------------------------------------------------

// writeTestTask writes a task file in the standard format.
func writeTestTask(t *testing.T, dir string, tsk *task.Task) {
	t.Helper()
	slug := task.GenerateSlug(tsk.Title)
	filename := task.GenerateFilename(tsk.ID, slug)
	path := filepath.Join(dir, filename)
	if err := task.Write(path, tsk); err != nil {
		t.Fatalf("writing task %d: %v", tsk.ID, err)
	}
}

func TestFindDependentsAsParent(t *testing.T) {
	dir := t.TempDir()
	parentID := 1

	writeTestTask(t, dir, &task.Task{
		ID: 1, Title: "Parent task", Status: "todo", Priority: "medium",
		Created: time.Now(), Updated: time.Now(),
	})
	writeTestTask(t, dir, &task.Task{
		ID: 2, Title: "Child task", Status: "todo", Priority: "medium",
		Parent: &parentID, Created: time.Now(), Updated: time.Now(),
	})

	msgs := FindDependents(dir, 1)
	if len(msgs) != 1 {
		t.Fatalf("len = %d, want 1", len(msgs))
	}
	if !strings.Contains(msgs[0], "parent") {
		t.Errorf("msg = %q, want mention of 'parent'", msgs[0])
	}
}

func TestFindDependentsAsDependency(t *testing.T) {
	dir := t.TempDir()

	writeTestTask(t, dir, &task.Task{
		ID: 1, Title: "Dep task", Status: "todo", Priority: "medium",
		Created: time.Now(), Updated: time.Now(),
	})
	writeTestTask(t, dir, &task.Task{
		ID: 2, Title: "Dependent task", Status: "todo", Priority: "medium",
		DependsOn: []int{1}, Created: time.Now(), Updated: time.Now(),
	})

	msgs := FindDependents(dir, 1)
	if len(msgs) != 1 {
		t.Fatalf("len = %d, want 1", len(msgs))
	}
	if !strings.Contains(msgs[0], "depends") {
		t.Errorf("msg = %q, want mention of 'depends'", msgs[0])
	}
}

func TestFindDependentsNone(t *testing.T) {
	dir := t.TempDir()

	writeTestTask(t, dir, &task.Task{
		ID: 1, Title: "Standalone", Status: "todo", Priority: "medium",
		Created: time.Now(), Updated: time.Now(),
	})

	msgs := FindDependents(dir, 1)
	if len(msgs) != 0 {
		t.Errorf("expected no dependents, got %d: %v", len(msgs), msgs)
	}
}

func TestFindDependentsInvalidDir(t *testing.T) {
	msgs := FindDependents("/nonexistent/path", 1)
	if msgs != nil {
		t.Errorf("expected nil for invalid dir, got %v", msgs)
	}
}

// ---------------------------------------------------------------------------
// LogMutation
// ---------------------------------------------------------------------------

func TestLogMutationCreatesEntry(t *testing.T) {
	dir := t.TempDir()

	LogMutation(dir, "create", 1, "Test task")

	// Read back the log and verify.
	entries, err := ReadLog(dir, LogFilterOptions{})
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].Action != "create" {
		t.Errorf("action = %q, want %q", entries[0].Action, "create")
	}
	if entries[0].TaskID != 1 {
		t.Errorf("taskID = %d, want 1", entries[0].TaskID)
	}
	if entries[0].Detail != "Test task" {
		t.Errorf("detail = %q, want %q", entries[0].Detail, "Test task")
	}
}

func TestLogMutationSilentOnError(t *testing.T) {
	// Using a read-only directory should cause AppendLog to fail,
	// but LogMutation should not panic.
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs") // doesn't exist — no tasks subdir
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Make the log file unwritable.
	logFile := filepath.Join(dir, "activity.log")
	if err := os.WriteFile(logFile, []byte("existing"), 0o400); err != nil {
		t.Fatalf("write: %v", err)
	}

	// This should not panic even though writing will fail.
	LogMutation(dir, "test", 1, "should not panic")
}

// ---------------------------------------------------------------------------
// ValidGroupByFields
// ---------------------------------------------------------------------------

func TestValidGroupByFields(t *testing.T) {
	fields := ValidGroupByFields()

	const expectedLen = 5
	if len(fields) != expectedLen {
		t.Fatalf("len = %d, want %d", len(fields), expectedLen)
	}

	required := map[string]bool{
		"assignee": false,
		"tag":      false,
		"class":    false,
		"priority": false,
		"status":   false,
	}
	for _, f := range fields {
		if _, ok := required[f]; !ok {
			t.Errorf("unexpected field %q", f)
		}
		required[f] = true
	}
	for field, found := range required {
		if !found {
			t.Errorf("missing required field %q", field)
		}
	}
}

// ---------------------------------------------------------------------------
// sectionTitle
// ---------------------------------------------------------------------------

func TestSectionTitle(t *testing.T) {
	tests := [...]struct {
		input string
		want  string
	}{
		{sectionInProgress, "In Progress"},
		{sectionBlocked, "Blocked"},
		{sectionReady, "Ready to Start"},
		{sectionOverdue, "Overdue"},
		{sectionRecentlyCompleted, "Recently Completed"},
		{"custom-section", "custom-section"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sectionTitle(tt.input)
		if got != tt.want {
			t.Errorf("sectionTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// List (integration with temp dir)
// ---------------------------------------------------------------------------

func TestListEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := config.NewDefault("Test Board")
	cfg.SetDir(dir)

	tasks, warnings, err := List(cfg, ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %d, want 0", len(warnings))
	}
	if len(tasks) != 0 {
		t.Errorf("tasks = %d, want 0", len(tasks))
	}
}

func TestListWithLimit(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.NewDefault("Test Board")
	cfg.SetDir(dir)

	for i := 1; i <= 5; i++ {
		writeTestTask(t, tasksDir, &task.Task{
			ID: i, Title: "Task", Status: "backlog", Priority: "medium",
			Created: time.Now(), Updated: time.Now(),
		})
	}

	tasks, _, err := List(cfg, ListOptions{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("tasks = %d, want 2", len(tasks))
	}
}

func TestListWithFilter(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.NewDefault("Test Board")
	cfg.SetDir(dir)

	writeTestTask(t, tasksDir, &task.Task{
		ID: 1, Title: "Backlog", Status: "backlog", Priority: "medium",
		Created: time.Now(), Updated: time.Now(),
	})
	writeTestTask(t, tasksDir, &task.Task{
		ID: 2, Title: "In progress", Status: "in-progress", Priority: "high",
		Created: time.Now(), Updated: time.Now(),
	})

	tasks, _, err := List(cfg, ListOptions{
		Filter: FilterOptions{Statuses: []string{"in-progress"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(tasks))
	}
	if tasks[0].Status != "in-progress" {
		t.Errorf("status = %q, want %q", tasks[0].Status, "in-progress")
	}
}

func TestListSortReverse(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.NewDefault("Test Board")
	cfg.SetDir(dir)

	for i := 1; i <= 3; i++ {
		writeTestTask(t, tasksDir, &task.Task{
			ID: i, Title: "Task", Status: "backlog", Priority: "medium",
			Created: time.Now(), Updated: time.Now(),
		})
	}

	tasks, _, err := List(cfg, ListOptions{SortBy: "id", Reverse: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("tasks = %d, want 3", len(tasks))
	}
	if tasks[0].ID != 3 {
		t.Errorf("first task ID = %d, want 3 (reverse order)", tasks[0].ID)
	}
	if tasks[2].ID != 1 {
		t.Errorf("last task ID = %d, want 1 (reverse order)", tasks[2].ID)
	}
}
