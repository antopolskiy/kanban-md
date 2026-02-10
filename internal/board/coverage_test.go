package board

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ---------------------------------------------------------------------------
// AppendLog — file creation error path (75% → higher)
// ---------------------------------------------------------------------------

func TestAppendLog_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not restrict directory writes on Windows")
	}
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readOnlyDir, 0o444); err != nil { //nolint:gosec // intentionally restrict for test
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o750) }) //nolint:gosec // restore permissions in cleanup

	entry := LogEntry{Timestamp: time.Now(), Action: "create", TaskID: 1, Detail: "test"}
	err := AppendLog(readOnlyDir, entry)
	if err == nil {
		t.Error("expected error writing to read-only dir")
	}
}

// ---------------------------------------------------------------------------
// ReadLog — malformed lines (83.3% → higher)
// ---------------------------------------------------------------------------

func TestReadLog_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)

	// Write a mix of valid and malformed lines.
	content := `{"timestamp":"2025-06-15T12:00:00Z","action":"create","task_id":1,"detail":"good"}
this is not json
{"timestamp":"2025-06-15T13:00:00Z","action":"move","task_id":2,"detail":"also good"}

`
	if err := os.WriteFile(logPath, []byte(content), logFileMode); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadLog(dir, LogFilterOptions{})
	if err != nil {
		t.Fatalf("ReadLog error: %v", err)
	}
	// Should skip the malformed line and the empty line, return 2.
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2 (skip malformed)", len(entries))
	}
}

// ---------------------------------------------------------------------------
// truncateLogIfNeeded — under limit (88.9% → higher)
// ---------------------------------------------------------------------------

func TestTruncateLogIfNeeded_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)

	// Write fewer entries than the limit.
	var buf strings.Builder
	for i := 0; i < 10; i++ {
		buf.WriteString(`{"timestamp":"2025-06-15T12:00:00Z","action":"create","task_id":1,"detail":"x"}`)
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(logPath, []byte(buf.String()), logFileMode); err != nil {
		t.Fatal(err)
	}

	// Should be a no-op.
	if err := truncateLogIfNeeded(logPath); err != nil {
		t.Fatalf("truncateLogIfNeeded error: %v", err)
	}

	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	const expectedLines = 10
	if len(lines) != expectedLines {
		t.Errorf("got %d lines, want 10 (no truncation)", len(lines))
	}
}

// ---------------------------------------------------------------------------
// compareDue — nil cases (85.7% → 100%)
// ---------------------------------------------------------------------------

func TestCompareDue_BothNil(t *testing.T) {
	a := &task.Task{ID: 1}
	b := &task.Task{ID: 2}
	if compareDue(a, b) {
		t.Error("expected false when both due dates are nil")
	}
}

func TestCompareDue_FirstNilSortsLast(t *testing.T) {
	d := date.New(2026, time.March, 1)
	a := &task.Task{ID: 1} // nil due
	b := &task.Task{ID: 2, Due: &d}
	if compareDue(a, b) {
		t.Error("nil due should sort last (not less than non-nil)")
	}
}

func TestCompareDue_SecondNilSortsLast(t *testing.T) {
	d := date.New(2026, time.March, 1)
	a := &task.Task{ID: 1, Due: &d}
	b := &task.Task{ID: 2} // nil due
	if !compareDue(a, b) {
		t.Error("non-nil due should sort before nil")
	}
}

// ---------------------------------------------------------------------------
// sortPickCandidates — tie-breaking (81.8% → higher)
// ---------------------------------------------------------------------------

func TestSortPickCandidates_FixedDateSameDue(t *testing.T) {
	cfg := newPickTestConfig()
	sameDue := date.New(2026, time.March, 1)
	candidates := [2]*task.Task{
		{ID: 1, Status: "todo", Priority: "low", Class: "fixed-date", Due: &sameDue},
		{ID: 2, Status: "todo", Priority: "high", Class: "fixed-date", Due: &sameDue},
	}
	slice := candidates[:]

	sortPickCandidates(slice, cfg)
	// Same due date → fallback to priority. High is higher index → comes first.
	if len(slice) == 0 {
		t.Fatal("no candidates after sort")
	}
	if slice[0].ID != 2 {
		t.Errorf("first candidate ID = %d, want 2 (higher priority)", slice[0].ID)
	}
}

func TestSortPickCandidates_FixedDateDifferentDue(t *testing.T) {
	cfg := newPickTestConfig()
	early := date.New(2026, time.February, 1)
	late := date.New(2026, time.April, 1)
	candidates := [2]*task.Task{
		{ID: 1, Status: "todo", Priority: "high", Class: "fixed-date", Due: &late},
		{ID: 2, Status: "todo", Priority: "low", Class: "fixed-date", Due: &early},
	}
	slice := candidates[:]

	sortPickCandidates(slice, cfg)
	// Earlier due date first.
	if len(slice) == 0 {
		t.Fatal("no candidates after sort")
	}
	if slice[0].ID != 2 {
		t.Errorf("first candidate ID = %d, want 2 (earlier due)", slice[0].ID)
	}
}

// ---------------------------------------------------------------------------
// classOrder — edge cases (83.3% → higher)
// ---------------------------------------------------------------------------

func TestClassOrder_EmptyClassDefaultsToStandard(t *testing.T) {
	cfg := newPickTestConfig()
	tk := &task.Task{ID: 1, Class: ""}
	got := classOrder(tk, cfg)
	want := cfg.ClassIndex(classStandard)
	if got != want {
		t.Errorf("classOrder empty = %d, want %d (standard)", got, want)
	}
}

func TestClassOrder_UnknownClassDefaultsToStandard(t *testing.T) {
	cfg := newPickTestConfig()
	tk := &task.Task{ID: 1, Class: "nonexistent"}
	got := classOrder(tk, cfg)
	want := cfg.ClassIndex(classStandard)
	if got != want {
		t.Errorf("classOrder unknown = %d, want %d (standard)", got, want)
	}
}

// ---------------------------------------------------------------------------
// filterPickDeps — no statuses configured (90% → higher)
// ---------------------------------------------------------------------------

func TestFilterPickDeps_NoStatuses(t *testing.T) {
	cfg := &config.Config{} // no statuses
	candidates := []*task.Task{
		{ID: 1, Status: "todo", DependsOn: []int{2}},
		{ID: 2, Status: "done"},
	}
	result := filterPickDeps(cfg, candidates, candidates)
	// With no statuses configured, should return all candidates unchanged.
	if len(result) != len(candidates) {
		t.Errorf("got %d candidates, want %d (no status filter)", len(result), len(candidates))
	}
}

// ---------------------------------------------------------------------------
// RenderContextMarkdown — WIP warning and note (83.3% → higher)
// ---------------------------------------------------------------------------

func TestRenderContextMarkdown_WIPWarning(t *testing.T) {
	data := ContextData{
		BoardName: "Test",
		Summary: ContextSummary{
			TotalTasks: 1,
			WIPWarning: "WIP limit reached: in-progress (3/3)",
		},
	}
	md := RenderContextMarkdown(data)
	if !strings.Contains(md, "WIP limit reached") {
		t.Errorf("expected WIP warning in output:\n%s", md)
	}
}

func TestRenderContextMarkdown_ItemNote(t *testing.T) {
	data := ContextData{
		BoardName: "Test",
		Summary:   ContextSummary{TotalTasks: 1},
		Sections: []ContextSection{
			{
				Name: sectionOverdue,
				Items: []ContextItem{
					{ID: 1, Title: "Overdue task", Priority: "high", Note: "due 2026-01-01"},
				},
			},
		},
	}
	md := RenderContextMarkdown(data)
	if !strings.Contains(md, "due 2026-01-01") {
		t.Errorf("expected note in output:\n%s", md)
	}
}

// ---------------------------------------------------------------------------
// computeSummary — WIP limits (88.9% → higher)
// ---------------------------------------------------------------------------

func TestComputeSummary_WIPWarning(t *testing.T) {
	cfg := &config.Config{
		Board:     config.BoardConfig{Name: "Test"},
		Statuses:  []config.StatusConfig{{Name: "backlog"}, {Name: "in-progress"}, {Name: "done"}},
		WIPLimits: map[string]int{"in-progress": 2},
	}
	tasks := []*task.Task{
		{ID: 1, Status: "in-progress", Priority: "high"},
		{ID: 2, Status: "in-progress", Priority: "medium"},
	}
	summary := computeSummary(cfg, tasks, time.Now())
	if summary.WIPWarning == "" {
		t.Error("expected WIP warning for at-limit status")
	}
	if !strings.Contains(summary.WIPWarning, "in-progress") {
		t.Errorf("WIP warning should mention in-progress: %q", summary.WIPWarning)
	}
}

// ---------------------------------------------------------------------------
// matchesExtendedFilter — class filter (88.9% → higher)
// ---------------------------------------------------------------------------

func TestMatchesExtendedFilter_ClassMatch(t *testing.T) {
	tk := &task.Task{ID: 1, Class: "expedite"}
	opts := FilterOptions{Class: "expedite"}
	if !matchesExtendedFilter(tk, opts) {
		t.Error("expected class match")
	}
}

func TestMatchesExtendedFilter_ClassMismatch(t *testing.T) {
	tk := &task.Task{ID: 1, Class: "standard"}
	opts := FilterOptions{Class: "expedite"}
	if matchesExtendedFilter(tk, opts) {
		t.Error("expected class mismatch to filter out")
	}
}

// ---------------------------------------------------------------------------
// matchesStatus — exclude statuses (80% → higher)
// ---------------------------------------------------------------------------

func TestMatchesStatus_ExcludeOnly(t *testing.T) {
	if matchesStatus("archived", nil, []string{"archived"}) {
		t.Error("archived should be excluded")
	}
	if !matchesStatus("todo", nil, []string{"archived"}) {
		t.Error("todo should not be excluded")
	}
}

func TestMatchesStatus_IncludeAndExclude(t *testing.T) {
	// Include backlog+todo, exclude todo → only backlog passes.
	if !matchesStatus("backlog", []string{"backlog", "todo"}, []string{"todo"}) {
		t.Error("backlog should pass include+exclude")
	}
	if matchesStatus("todo", []string{"backlog", "todo"}, []string{"todo"}) {
		t.Error("todo should be excluded even though included")
	}
}

// ---------------------------------------------------------------------------
// WriteContextToFile — append without trailing newline (90% → higher)
// ---------------------------------------------------------------------------

func TestWriteContextToFile_AppendNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	// Existing file without trailing newline.
	if err := os.WriteFile(path, []byte("existing content"), logFileMode); err != nil {
		t.Fatal(err)
	}

	content := contextBeginMarker + "\nnew\n" + contextEndMarker + "\n"
	if err := WriteContextToFile(path, content); err != nil {
		t.Fatalf("WriteContextToFile error: %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)
	if !strings.Contains(result, "existing content") {
		t.Error("lost existing content")
	}
	if !strings.Contains(result, contextBeginMarker) {
		t.Error("missing new context")
	}
	// Should have double newline separator since original had no trailing newline.
	if !strings.Contains(result, "content\n\n"+contextBeginMarker) {
		t.Errorf("expected double newline separator, got:\n%s", result)
	}
}

// ---------------------------------------------------------------------------
// List — unblocked filter (84.6% → higher)
// ---------------------------------------------------------------------------

func TestList_UnblockedFilter(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefault("Test Board")
	cfg.SetDir(dir)

	now := time.Now()
	// Task 1: done (dependency).
	writeTestTask(t, tasksDir, &task.Task{
		ID: 1, Title: "Done dep", Status: "done", Priority: "medium",
		Created: now, Updated: now,
	})
	// Task 2: depends on done task 1 → unblocked.
	writeTestTask(t, tasksDir, &task.Task{
		ID: 2, Title: "Unblocked", Status: "todo", Priority: "high",
		DependsOn: []int{1}, Created: now, Updated: now,
	})
	// Task 3: depends on in-progress task 4 → blocked.
	writeTestTask(t, tasksDir, &task.Task{
		ID: 3, Title: "Blocked by dep", Status: "todo", Priority: "high",
		DependsOn: []int{4}, Created: now, Updated: now,
	})
	// Task 4: in-progress, no deps.
	writeTestTask(t, tasksDir, &task.Task{
		ID: 4, Title: "In progress", Status: "in-progress", Priority: "medium",
		Created: now, Updated: now,
	})

	tasks, _, err := List(cfg, ListOptions{Unblocked: true})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	// Task 3 should be filtered out (dep 4 not done). Tasks 1, 2, 4 remain.
	if len(tasks) != 3 {
		t.Errorf("got %d tasks, want 3 (unblocked only)", len(tasks))
		for _, tk := range tasks {
			t.Logf("  task #%d %q", tk.ID, tk.Title)
		}
	}
	for _, tk := range tasks {
		if tk.ID == 3 {
			t.Error("task #3 should be filtered (blocked dep)")
		}
	}
}

// ---------------------------------------------------------------------------
// extractGroupKeys — default/unknown field (85.7% → higher)
// ---------------------------------------------------------------------------

func TestExtractGroupKeys_UnknownField(t *testing.T) {
	tk := &task.Task{ID: 1}
	keys := extractGroupKeys(tk, "unknown_field")
	if len(keys) != 1 || keys[0] != "(all)" {
		t.Errorf("keys = %v, want [(all)]", keys)
	}
}

func TestExtractGroupKeys_PriorityField(t *testing.T) {
	tk := &task.Task{ID: 1, Priority: "high"}
	keys := extractGroupKeys(tk, fieldPriority)
	if len(keys) != 1 || keys[0] != "high" {
		t.Errorf("keys = %v, want [high]", keys)
	}
}

// ---------------------------------------------------------------------------
// sortGroupKeys — class and priority sorting (83.3% → higher)
// ---------------------------------------------------------------------------

func TestSortGroupKeys_ByPriority(t *testing.T) {
	cfg := newGroupTestConfig()
	groups := map[string][]*task.Task{
		"high":   {{ID: 1}},
		"low":    {{ID: 2}},
		"medium": {{ID: 3}},
	}
	keys := sortGroupKeys(groups, fieldPriority, cfg)
	// Should be sorted by config priority index: low, medium, high.
	if len(keys) != 3 {
		t.Fatalf("len = %d, want 3", len(keys))
	}
	if keys[0] != "low" {
		t.Errorf("keys[0] = %q, want %q", keys[0], "low")
	}
	if keys[2] != "high" {
		t.Errorf("keys[2] = %q, want %q", keys[2], "high")
	}
}

func TestSortGroupKeys_ByClass(t *testing.T) {
	cfg := newGroupTestConfig()
	groups := map[string][]*task.Task{
		classStandard: {{ID: 1}},
		"expedite":    {{ID: 2}},
		"intangible":  {{ID: 3}},
	}
	keys := sortGroupKeys(groups, "class", cfg)
	// Should be sorted by config class index: expedite, standard, intangible.
	if len(keys) != 3 {
		t.Fatalf("len = %d, want 3", len(keys))
	}
	if keys[0] != "expedite" {
		t.Errorf("keys[0] = %q, want %q", keys[0], "expedite")
	}
	if keys[2] != "intangible" {
		t.Errorf("keys[2] = %q, want %q", keys[2], "intangible")
	}
}

// ---------------------------------------------------------------------------
// CheckWIPLimit (0% → higher)
// ---------------------------------------------------------------------------

func TestCheckWIPLimit_NoLimit(t *testing.T) {
	cfg := config.NewDefault("Test")
	counts := map[string]int{"in-progress": 5}
	err := CheckWIPLimit(cfg, counts, "in-progress", "")
	if err != nil {
		t.Errorf("expected nil when no WIP limit, got %v", err)
	}
}

func TestCheckWIPLimit_WithinLimit(t *testing.T) {
	cfg := &config.Config{
		Statuses:  []config.StatusConfig{{Name: "backlog"}, {Name: "in-progress"}, {Name: "done"}},
		WIPLimits: map[string]int{"in-progress": 5},
	}
	counts := map[string]int{"in-progress": 3}
	err := CheckWIPLimit(cfg, counts, "in-progress", "")
	if err != nil {
		t.Errorf("expected nil within limit, got %v", err)
	}
}

func TestCheckWIPLimit_AtLimit(t *testing.T) {
	cfg := &config.Config{
		Statuses:  []config.StatusConfig{{Name: "backlog"}, {Name: "in-progress"}, {Name: "done"}},
		WIPLimits: map[string]int{"in-progress": 3},
	}
	counts := map[string]int{"in-progress": 3}
	err := CheckWIPLimit(cfg, counts, "in-progress", "")
	if err == nil {
		t.Fatal("expected error at WIP limit")
	}
}

func TestCheckWIPLimit_SameStatusNoCount(t *testing.T) {
	cfg := &config.Config{
		Statuses:  []config.StatusConfig{{Name: "backlog"}, {Name: "in-progress"}, {Name: "done"}},
		WIPLimits: map[string]int{"in-progress": 3},
	}
	counts := map[string]int{"in-progress": 3}
	// Task already in target status shouldn't add.
	err := CheckWIPLimit(cfg, counts, "in-progress", "in-progress")
	if err != nil {
		t.Errorf("expected nil when task already at target, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// matchesCoreFilter — priorities filter (92.3% → higher)
// ---------------------------------------------------------------------------

func TestMatchesCoreFilter_PrioritiesFilter(t *testing.T) {
	tk := &task.Task{ID: 1, Status: "todo", Priority: "high"}
	opts := FilterOptions{Priorities: []string{"low", "medium"}}
	if matchesCoreFilter(tk, opts) {
		t.Error("high priority should not match [low, medium] filter")
	}

	opts2 := FilterOptions{Priorities: []string{"high", "critical"}}
	if !matchesCoreFilter(tk, opts2) {
		t.Error("high priority should match [high, critical] filter")
	}
}

// ---------------------------------------------------------------------------
// FindDependents — both parent and dependency (91.7% → higher)
// ---------------------------------------------------------------------------

func TestFindDependents_BothParentAndDep(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	parentID := 1

	writeTestTask(t, dir, &task.Task{
		ID: 1, Title: "Target", Status: "todo", Priority: "medium",
		Created: now, Updated: now,
	})
	// Task 2 has target as parent.
	writeTestTask(t, dir, &task.Task{
		ID: 2, Title: "Child", Status: "todo", Priority: "medium",
		Parent: &parentID, Created: now, Updated: now,
	})
	// Task 3 depends on target.
	writeTestTask(t, dir, &task.Task{
		ID: 3, Title: "Dep", Status: "todo", Priority: "medium",
		DependsOn: []int{1}, Created: now, Updated: now,
	})

	msgs := FindDependents(dir, 1)
	if len(msgs) != 2 {
		t.Errorf("got %d messages, want 2 (parent + dep)", len(msgs))
	}
}
