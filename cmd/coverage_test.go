package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// setFlag sets a cobra command flag and fails the test on error.
func setFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatalf("setting flag --%s=%s: %v", name, value, err)
	}
}

// --- runCompletion: default (unknown shell) ---

func TestRunCompletion_UnknownShell(t *testing.T) {
	saveRootUse(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	// Call directly — bypasses Cobra's ValidArgs check.
	err := runCompletion(completionCmd, []string{"unknown"})
	if err != nil {
		t.Fatalf("runCompletion(unknown) error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for unknown shell, got %d bytes", buf.Len())
	}
}

// --- runShow unit tests ---

func TestRunShow_InvalidID(t *testing.T) {
	kanbanDir := setupBoard(t)
	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runShow(showCmd, []string{"abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
}

func TestRunShow_NotFound(t *testing.T) {
	kanbanDir := setupBoard(t)
	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runShow(showCmd, []string{"999"})
	if err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestRunShow_TableOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "show-me")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err = runShow(showCmd, []string{"1"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runShow error: %v", err)
	}
	if !containsSubstring(got, "show-me") {
		t.Errorf("expected 'show-me' in output, got: %s", got)
	}
}

func TestRunShow_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "show-json")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err = runShow(showCmd, []string{"1"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runShow error: %v", err)
	}
	if !containsSubstring(got, `"title"`) {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestRunShow_CompactOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "show-compact")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, false, true)
	r, w := captureStdout(t)

	err = runShow(showCmd, []string{"1"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runShow error: %v", err)
	}
	if !containsSubstring(got, "show-compact") {
		t.Errorf("expected compact output, got: %s", got)
	}
}

func TestRunShow_NoConfig(t *testing.T) {
	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	t.Chdir(t.TempDir())

	err := runShow(showCmd, []string{"1"})
	if err == nil {
		t.Fatal("expected error when no config")
	}
}

// --- runLog unit tests ---

func TestRunLog_TableOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	// Generate some log entries.
	logActivity(cfg, "create", 1, "test")
	logActivity(cfg, "move", 1, "backlog -> todo")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err = runLog(logCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runLog error: %v", err)
	}
	if !containsSubstring(got, "create") {
		t.Errorf("expected 'create' in output, got: %s", got)
	}
}

func TestRunLog_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	logActivity(cfg, "create", 1, "test")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err = runLog(logCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runLog error: %v", err)
	}
	if !containsSubstring(got, `"action"`) {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestRunLog_CompactOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	logActivity(cfg, "create", 1, "test")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, false, true)
	r, w := captureStdout(t)

	err = runLog(logCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runLog error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected non-empty compact output")
	}
}

func TestRunLog_JSONEmptyEntries(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err := runLog(logCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runLog error: %v", err)
	}
	// Should return empty JSON array, not null.
	if !containsSubstring(got, "[]") {
		t.Errorf("expected '[]' for empty entries, got: %s", got)
	}
}

func TestRunLog_InvalidSinceDate(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	// Set the --since flag value.
	setFlag(t, logCmd, "since", "not-a-date")
	t.Cleanup(func() { _ = logCmd.Flags().Set("since", "") })

	err := runLog(logCmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid since date")
	}
}

func TestRunLog_WithFilters(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	logActivity(cfg, "create", 1, "task one")
	logActivity(cfg, "move", 2, "backlog -> todo")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)

	// Test --action filter.
	setFlag(t, logCmd, "action", "create")
	t.Cleanup(func() { _ = logCmd.Flags().Set("action", "") })

	r, w := captureStdout(t)
	err = runLog(logCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runLog --action error: %v", err)
	}
	_ = got // just ensure no error
}

func TestRunLog_NoConfig(t *testing.T) {
	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	t.Chdir(t.TempDir())

	err := runLog(logCmd, nil)
	if err == nil {
		t.Fatal("expected error when no config")
	}
}

// --- runMetrics unit tests ---

func TestRunMetrics_TableOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "metrics-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err = runMetrics(metricsCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runMetrics error: %v", err)
	}
	// Should produce some metrics output.
	if len(got) == 0 {
		t.Error("expected non-empty metrics output")
	}
}

func TestRunMetrics_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err := runMetrics(metricsCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runMetrics error: %v", err)
	}
	if !containsSubstring(got, "{") {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestRunMetrics_CompactOutput(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, false, true)
	r, w := captureStdout(t)

	err := runMetrics(metricsCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runMetrics error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected non-empty compact metrics output")
	}
}

func TestRunMetrics_InvalidSinceDate(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlag(t, metricsCmd, "since", "bad-date")
	t.Cleanup(func() { _ = metricsCmd.Flags().Set("since", "") })

	err := runMetrics(metricsCmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid since date")
	}
}

func TestRunMetrics_NoConfig(t *testing.T) {
	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	t.Chdir(t.TempDir())

	err := runMetrics(metricsCmd, nil)
	if err == nil {
		t.Fatal("expected error when no config")
	}
}

// --- runList unit tests ---

func TestRunList_TableOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "list-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList error: %v", err)
	}
	if !containsSubstring(got, "list-task") {
		t.Errorf("expected 'list-task' in output, got: %s", got)
	}
}

func TestRunList_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "list-json")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList error: %v", err)
	}
	if !containsSubstring(got, `"title"`) {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestRunList_CompactOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "list-compact")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, false, true)
	r, w := captureStdout(t)

	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList error: %v", err)
	}
	if !containsSubstring(got, "list-compact") {
		t.Errorf("expected compact output, got: %s", got)
	}
}

func TestRunList_JSONEmptyTasks(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err := runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList error: %v", err)
	}
	// Should return empty array, not null.
	if !containsSubstring(got, "[]") {
		t.Errorf("expected '[]' for empty tasks, got: %s", got)
	}
}

func TestRunList_InvalidGroupBy(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlag(t, listCmd, "group-by", "nonexistent")
	t.Cleanup(func() { _ = listCmd.Flags().Set("group-by", "") })

	err := runList(listCmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid group-by field")
	}
}

func TestRunList_GroupByStatus(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "grouped-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, listCmd, "group-by", "status")
	t.Cleanup(func() { _ = listCmd.Flags().Set("group-by", "") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --group-by error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected non-empty grouped output")
	}
}

func TestRunList_GroupByJSON(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "grouped-json")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	setFlag(t, listCmd, "group-by", "status")
	t.Cleanup(func() { _ = listCmd.Flags().Set("group-by", "") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --group-by JSON error: %v", err)
	}
	if !containsSubstring(got, "{") {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestRunList_NoConfig(t *testing.T) {
	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	t.Chdir(t.TempDir())

	err := runList(listCmd, nil)
	if err == nil {
		t.Fatal("expected error when no config")
	}
}

func TestRunList_ArchivedFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create an archived task.
	tk := &task.Task{
		ID:       1,
		Title:    "archived-task",
		Status:   config.ArchivedStatus,
		Priority: "medium",
	}
	path := filepath.Join(cfg.TasksPath(), task.GenerateFilename(1, "archived-task"))
	if writeErr := task.Write(path, tk); writeErr != nil {
		t.Fatal(writeErr)
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, listCmd, "archived", "true")
	t.Cleanup(func() { _ = listCmd.Flags().Set("archived", "false") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --archived error: %v", err)
	}
	if !containsSubstring(got, "archived-task") {
		t.Errorf("expected archived task in output, got: %s", got)
	}
}

func TestRunLog_WithLimitAndTaskFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	logActivity(cfg, "create", 1, "task one")
	logActivity(cfg, "create", 2, "task two")
	logActivity(cfg, "move", 1, "backlog -> todo")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)

	// Test --limit flag.
	setFlag(t, logCmd, "limit", "1")
	t.Cleanup(func() {
		_ = logCmd.Flags().Set("limit", "0")
		_ = logCmd.Flags().Set("task", "0")
	})

	r, w := captureStdout(t)
	err = runLog(logCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runLog --limit error: %v", err)
	}
	_ = got

	// Test --task filter.
	setFlag(t, logCmd, "limit", "0")
	setFlag(t, logCmd, "task", "1")

	r2, w2 := captureStdout(t)
	err = runLog(logCmd, nil)
	got2 := drainPipe(t, r2, w2)

	if err != nil {
		t.Fatalf("runLog --task error: %v", err)
	}
	_ = got2
}

func TestRunLog_ValidSinceDate(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	logActivity(cfg, "create", 1, "test")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, logCmd, "since", "2020-01-01")
	t.Cleanup(func() { _ = logCmd.Flags().Set("since", "") })

	r, w := captureStdout(t)
	err = runLog(logCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runLog --since error: %v", err)
	}
	if !containsSubstring(got, "create") {
		t.Errorf("expected 'create' in filtered output, got: %s", got)
	}
}

// --- runMetrics: since filter with valid date + archived task exclusion ---

func TestRunMetrics_ValidSinceDate(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "recent-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, metricsCmd, "since", "2020-01-01")
	t.Cleanup(func() { _ = metricsCmd.Flags().Set("since", "") })

	r, w := captureStdout(t)
	err = runMetrics(metricsCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runMetrics --since error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected non-empty metrics with since filter")
	}
}

func TestRunMetrics_ExcludesArchived(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create an archived task — should be excluded from metrics.
	tk := &task.Task{
		ID:       1,
		Title:    "archived-task",
		Status:   config.ArchivedStatus,
		Priority: "medium",
	}
	path := filepath.Join(cfg.TasksPath(), task.GenerateFilename(1, "archived-task"))
	if writeErr := task.Write(path, tk); writeErr != nil {
		t.Fatal(writeErr)
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err = runMetrics(metricsCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runMetrics error: %v", err)
	}
	// Archived tasks excluded — throughput should be 0.
	if !containsSubstring(got, `"throughput_7d": 0`) {
		t.Errorf("expected throughput=0 (archived excluded), got: %s", got)
	}
}

// --- runList: filter branches ---

func TestRunList_BlockedFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a blocked task.
	tk := &task.Task{
		ID:          1,
		Title:       "blocked-task",
		Status:      "backlog",
		Priority:    "medium",
		Blocked:     true,
		BlockReason: "waiting",
	}
	path := filepath.Join(cfg.TasksPath(), task.GenerateFilename(1, "blocked-task"))
	if writeErr := task.Write(path, tk); writeErr != nil {
		t.Fatal(writeErr)
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, listCmd, "blocked", "true")
	t.Cleanup(func() { _ = listCmd.Flags().Set("blocked", "false") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --blocked error: %v", err)
	}
	if !containsSubstring(got, "blocked-task") {
		t.Errorf("expected blocked task in output, got: %s", got)
	}
}

func TestRunList_NotBlockedFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "normal-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, listCmd, "not-blocked", "true")
	t.Cleanup(func() { _ = listCmd.Flags().Set("not-blocked", "false") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --not-blocked error: %v", err)
	}
	if !containsSubstring(got, "normal-task") {
		t.Errorf("expected normal task in output, got: %s", got)
	}
}

func TestRunList_UnclaimedFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "unclaimed-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, listCmd, "unclaimed", "true")
	t.Cleanup(func() { _ = listCmd.Flags().Set("unclaimed", "false") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --unclaimed error: %v", err)
	}
	if !containsSubstring(got, "unclaimed-task") {
		t.Errorf("expected unclaimed task in output, got: %s", got)
	}
}

func TestRunList_ClaimedByFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "claimed-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, listCmd, "claimed-by", "test-agent")
	t.Cleanup(func() { _ = listCmd.Flags().Set("claimed-by", "") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	_ = drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --claimed-by error: %v", err)
	}
}

func TestRunList_ClassFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "standard-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, listCmd, "class", "standard")
	t.Cleanup(func() { _ = listCmd.Flags().Set("class", "") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	_ = drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --class error: %v", err)
	}
}

func TestRunList_ParentFilter(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	createTaskFile(t, cfg.TasksPath(), 1, "parent-task")

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	setFlag(t, listCmd, "parent", "1")
	t.Cleanup(func() { _ = listCmd.Flags().Set("parent", "0") })

	r, w := captureStdout(t)
	err = runList(listCmd, nil)
	_ = drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runList --parent error: %v", err)
	}
}

// --- resolveDir: flagDir path ---

func TestResolveDir_WithFlagDir(t *testing.T) {
	oldFlagDir := flagDir
	flagDir = "/some/path"
	t.Cleanup(func() { flagDir = oldFlagDir })

	got, err := resolveDir()
	if err != nil {
		t.Fatalf("resolveDir() error: %v", err)
	}
	if got != "/some/path" {
		t.Errorf("resolveDir() = %q, want %q", got, "/some/path")
	}
}

func TestResolveDir_FindsFromCwd(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	t.Chdir(kanbanDir)

	got, err := resolveDir()
	if err != nil {
		t.Fatalf("resolveDir() error: %v", err)
	}
	if got != kanbanDir {
		t.Errorf("resolveDir() = %q, want %q", got, kanbanDir)
	}
}

func TestResolveDir_NotFound(t *testing.T) {
	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	t.Chdir(t.TempDir())

	_, err := resolveDir()
	if err == nil {
		t.Fatal("expected error when no kanban dir found")
	}
}

// --- enforceWIPLimit: via move.go ---

func TestEnforceWIPLimit_NoLimit(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.SetDir(setupBoard(t))

	// Create tasks on disk to count.
	createTaskFile(t, cfg.TasksPath(), 1, "task-one")

	err := enforceWIPLimit(cfg, "backlog", "todo")
	if err != nil {
		t.Errorf("expected nil when no WIP limit, got %v", err)
	}
}

// --- Ensure WIP limit enforcement reads tasks from disk ---

func TestEnforceWIPLimit_ExceedsLimit(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg.WIPLimits = map[string]int{"todo": 1}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}
	// Reload to get saved config.
	cfg, err = config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a task already in todo.
	tk := &task.Task{
		ID:       1,
		Title:    "already-in-todo",
		Status:   "todo",
		Priority: "medium",
	}
	path := filepath.Join(cfg.TasksPath(), task.GenerateFilename(1, "already-in-todo"))
	if writeErr := task.Write(path, tk); writeErr != nil {
		t.Fatal(writeErr)
	}

	err = enforceWIPLimit(cfg, "backlog", "todo")
	if err == nil {
		t.Fatal("expected WIP limit error")
	}
}

// --- findProjectRoot helper ---

func TestFindProjectRoot_FromKanbanDir(t *testing.T) {
	kanbanDir := setupBoard(t)
	// findProjectRoot walks up from cwd looking for .git or kanban/.
	t.Chdir(kanbanDir)

	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot() error: %v", err)
	}
	// Should find the parent of the kanban directory.
	if root == "" {
		t.Error("expected non-empty project root")
	}
}
