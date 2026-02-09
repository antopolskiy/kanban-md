package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// Shared test constants for the cmd package.
const (
	testBoardName = "TestBoard"
	priorityHigh  = "high"
)

func TestRootCommand(t *testing.T) {
	if rootCmd.Use != "kanban-md" {
		t.Errorf("rootCmd.Use = %v, want kanban-md", rootCmd.Use)
	}
}

// --- parseIDs tests ---

func TestParseIDs_Single(t *testing.T) {
	ids, err := parseIDs("42")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 42 {
		t.Errorf("parseIDs(\"42\") = %v, want [42]", ids)
	}
}

func TestParseIDs_Multiple(t *testing.T) {
	ids, err := parseIDs("1,2,3")
	if err != nil {
		t.Fatal(err)
	}
	want := [3]int{1, 2, 3}
	if len(ids) != 3 {
		t.Fatalf("len = %d, want 3", len(ids))
	}
	if [3]int{ids[0], ids[1], ids[2]} != want {
		t.Errorf("parseIDs(\"1,2,3\") = %v, want %v", ids, want)
	}
}

func TestParseIDs_Deduplicates(t *testing.T) {
	ids, err := parseIDs("1,2,1,3,2")
	if err != nil {
		t.Fatal(err)
	}
	want := [3]int{1, 2, 3}
	if len(ids) != 3 {
		t.Fatalf("len = %d, want 3", len(ids))
	}
	if [3]int{ids[0], ids[1], ids[2]} != want {
		t.Errorf("parseIDs(\"1,2,1,3,2\") = %v, want %v", ids, want)
	}
}

func TestParseIDs_TrimsSpaces(t *testing.T) {
	ids, err := parseIDs(" 1 , 2 ")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Errorf("parseIDs(\" 1 , 2 \") = %v, want [1, 2]", ids)
	}
}

func TestParseIDs_InvalidID(t *testing.T) {
	_, err := parseIDs("abc")
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidTaskID {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidTaskID)
	}
}

func TestParseIDs_EmptyString(t *testing.T) {
	_, err := parseIDs("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

// --- checkWIPLimit tests ---

func TestCheckWIPLimit_NoLimit(t *testing.T) {
	cfg := config.NewDefault("Test")
	counts := map[string]int{"in-progress": 5}
	err := checkWIPLimit(cfg, counts, "in-progress", "")
	if err != nil {
		t.Errorf("expected nil when no WIP limit set, got %v", err)
	}
}

func TestCheckWIPLimit_WithinLimit(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.WIPLimits = map[string]int{"in-progress": 5}
	counts := map[string]int{"in-progress": 3}
	err := checkWIPLimit(cfg, counts, "in-progress", "")
	if err != nil {
		t.Errorf("expected nil within limit, got %v", err)
	}
}

func TestCheckWIPLimit_AtLimit(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.WIPLimits = map[string]int{"in-progress": 3}
	counts := map[string]int{"in-progress": 3}
	err := checkWIPLimit(cfg, counts, "in-progress", "")
	if err == nil {
		t.Fatal("expected error at limit")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.WIPLimitExceeded {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.WIPLimitExceeded)
	}
}

func TestCheckWIPLimit_SameStatus(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.WIPLimits = map[string]int{"in-progress": 3}
	counts := map[string]int{"in-progress": 3}
	// Task already in target status should not count as adding.
	err := checkWIPLimit(cfg, counts, "in-progress", "in-progress")
	if err != nil {
		t.Errorf("expected nil when task already at target, got %v", err)
	}
}

// --- loadConfig tests ---

// setupBoard creates a temp kanban board and returns the directory path.
func setupBoard(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	_, err := config.Init(kanbanDir, testBoardName)
	if err != nil {
		t.Fatal(err)
	}
	return kanbanDir
}

func TestLoadConfig_WithFlagDir(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}
	if cfg.Board.Name != testBoardName {
		t.Errorf("board name = %q, want %q", cfg.Board.Name, testBoardName)
	}
}

func TestLoadConfig_FromCwd(t *testing.T) {
	kanbanDir := setupBoard(t)

	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	// Change to the kanban directory so config.FindDir discovers it.
	t.Chdir(kanbanDir)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}
	if cfg.Board.Name != testBoardName {
		t.Errorf("board name = %q, want %q", cfg.Board.Name, testBoardName)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	t.Chdir(dir)

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error when no kanban board exists")
	}
}

// --- outputFormat tests ---

func TestOutputFormat_Default(t *testing.T) {
	oldJSON, oldTable, oldCompact := flagJSON, flagTable, flagCompact
	flagJSON, flagTable, flagCompact = false, false, false
	t.Cleanup(func() {
		flagJSON, flagTable, flagCompact = oldJSON, oldTable, oldCompact
	})
	t.Setenv("KANBAN_OUTPUT", "")

	if got := outputFormat(); got != output.FormatTable {
		t.Errorf("outputFormat() = %v, want FormatTable", got)
	}
}

func TestOutputFormat_JSONFlag(t *testing.T) {
	oldJSON, oldTable, oldCompact := flagJSON, flagTable, flagCompact
	flagJSON, flagTable, flagCompact = true, false, false
	t.Cleanup(func() {
		flagJSON, flagTable, flagCompact = oldJSON, oldTable, oldCompact
	})

	if got := outputFormat(); got != output.FormatJSON {
		t.Errorf("outputFormat() = %v, want FormatJSON", got)
	}
}

func TestOutputFormat_CompactFlag(t *testing.T) {
	oldJSON, oldTable, oldCompact := flagJSON, flagTable, flagCompact
	flagJSON, flagTable, flagCompact = false, false, true
	t.Cleanup(func() {
		flagJSON, flagTable, flagCompact = oldJSON, oldTable, oldCompact
	})

	if got := outputFormat(); got != output.FormatCompact {
		t.Errorf("outputFormat() = %v, want FormatCompact", got)
	}
}

func TestOutputFormat_EnvJSON(t *testing.T) {
	oldJSON, oldTable, oldCompact := flagJSON, flagTable, flagCompact
	flagJSON, flagTable, flagCompact = false, false, false
	t.Cleanup(func() {
		flagJSON, flagTable, flagCompact = oldJSON, oldTable, oldCompact
	})
	t.Setenv("KANBAN_OUTPUT", "json")

	if got := outputFormat(); got != output.FormatJSON {
		t.Errorf("outputFormat() = %v, want FormatJSON", got)
	}
}

func TestOutputFormat_EnvCompact(t *testing.T) {
	oldJSON, oldTable, oldCompact := flagJSON, flagTable, flagCompact
	flagJSON, flagTable, flagCompact = false, false, false
	t.Cleanup(func() {
		flagJSON, flagTable, flagCompact = oldJSON, oldTable, oldCompact
	})
	t.Setenv("KANBAN_OUTPUT", "compact")

	if got := outputFormat(); got != output.FormatCompact {
		t.Errorf("outputFormat() = %v, want FormatCompact", got)
	}
}

// --- printWarnings tests ---

func TestPrintWarnings_Empty(t *testing.T) {
	// Redirect stderr to capture output.
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })

	printWarnings(nil)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if buf.Len() != 0 {
		t.Errorf("expected no output for nil warnings, got %q", buf.String())
	}
}

func TestPrintWarnings_WithWarnings(t *testing.T) {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })

	warnings := []task.ReadWarning{
		{File: "bad-task.md", Err: errors.New("parse error")},
		{File: "broken.md", Err: errors.New("missing frontmatter")},
	}
	printWarnings(warnings)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	got := buf.String()
	if !containsSubstring(got, "bad-task.md") {
		t.Errorf("expected warning about bad-task.md, got: %s", got)
	}
	if !containsSubstring(got, "broken.md") {
		t.Errorf("expected warning about broken.md, got: %s", got)
	}
	if !containsSubstring(got, "parse error") {
		t.Errorf("expected 'parse error' in output, got: %s", got)
	}
}

// --- validateDepIDs tests ---

func TestValidateDepIDs_ValidDeps(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o750); err != nil {
		t.Fatal(err)
	}

	// Create a task file that can be found.
	createTaskFile(t, tasksDir, 2, "dependency-task")

	err := validateDepIDs(tasksDir, 1, []int{2})
	if err != nil {
		t.Errorf("expected nil for valid dependency, got %v", err)
	}
}

func TestValidateDepIDs_SelfReference(t *testing.T) {
	dir := t.TempDir()

	err := validateDepIDs(dir, 1, []int{1})
	if err == nil {
		t.Fatal("expected error for self-reference")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.SelfReference {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.SelfReference)
	}
}

func TestValidateDepIDs_NotFound(t *testing.T) {
	dir := t.TempDir()

	err := validateDepIDs(dir, 1, []int{99})
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.DependencyNotFound {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.DependencyNotFound)
	}
}

// --- logActivity tests ---

func TestLogActivity_WritesLogEntry(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	logActivity(cfg, "create", 1, "test detail")

	// Verify the log file was created.
	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path from t.TempDir
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	got := string(data)
	if !containsSubstring(got, "create") {
		t.Errorf("log should contain action 'create', got: %s", got)
	}
	if !containsSubstring(got, "test detail") {
		t.Errorf("log should contain detail, got: %s", got)
	}
}

// --- checkClaim tests ---

func TestCheckClaim_Unclaimed(t *testing.T) {
	tk := &task.Task{ID: 1}
	err := checkClaim(tk, "", false, 0)
	if err != nil {
		t.Errorf("expected nil for unclaimed task, got %v", err)
	}
}

func TestCheckClaim_SameClaimant(t *testing.T) {
	tk := &task.Task{ID: 1, ClaimedBy: "agent-1"}
	now := time.Now()
	tk.ClaimedAt = &now

	err := checkClaim(tk, "agent-1", false, time.Hour)
	if err != nil {
		t.Errorf("expected nil for same claimant, got %v", err)
	}
}

func TestCheckClaim_DifferentClaimant(t *testing.T) {
	tk := &task.Task{ID: 1, ClaimedBy: "agent-1"}
	now := time.Now()
	tk.ClaimedAt = &now

	err := checkClaim(tk, "agent-2", false, time.Hour)
	if err == nil {
		t.Fatal("expected error for different claimant")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.TaskClaimed {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.TaskClaimed)
	}
}

func TestCheckClaim_Force(t *testing.T) {
	tk := &task.Task{ID: 1, ClaimedBy: "agent-1"}
	now := time.Now()
	tk.ClaimedAt = &now

	err := checkClaim(tk, "agent-2", true, time.Hour)
	if err != nil {
		t.Errorf("expected nil with force, got %v", err)
	}
	if tk.ClaimedBy != "" {
		t.Errorf("expected claim to be cleared with force, got %q", tk.ClaimedBy)
	}
}

func TestCheckClaim_Expired(t *testing.T) {
	tk := &task.Task{ID: 1, ClaimedBy: "agent-1"}
	past := time.Now().Add(-2 * time.Hour)
	tk.ClaimedAt = &past

	err := checkClaim(tk, "agent-2", false, time.Hour)
	if err != nil {
		t.Errorf("expected nil for expired claim, got %v", err)
	}
	if tk.ClaimedBy != "" {
		t.Errorf("expected expired claim to be cleared, got %q", tk.ClaimedBy)
	}
}

// --- validateDeps tests ---

func TestValidateDeps_NoDepNorParent(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.SetDir(t.TempDir())
	tk := &task.Task{ID: 1}

	err := validateDeps(cfg, tk)
	if err != nil {
		t.Errorf("expected nil for task with no deps, got %v", err)
	}
}

func TestValidateDeps_ValidParent(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	createTaskFile(t, cfg.TasksPath(), 1, "parent-task")
	createTaskFile(t, cfg.TasksPath(), 2, "child-task")

	parentID := 1
	tk := &task.Task{ID: 2, Parent: &parentID}

	if err := validateDeps(cfg, tk); err != nil {
		t.Errorf("expected nil for valid parent, got %v", err)
	}
}

func TestValidateDeps_InvalidParent(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	parentID := 99
	tk := &task.Task{ID: 1, Parent: &parentID}

	err = validateDeps(cfg, tk)
	if err == nil {
		t.Fatal("expected error for invalid parent")
	}
	if !containsSubstring(err.Error(), "invalid parent") {
		t.Errorf("expected 'invalid parent' in error, got: %v", err)
	}
}

func TestValidateDeps_ValidDependsOn(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	createTaskFile(t, cfg.TasksPath(), 1, "task-one")
	createTaskFile(t, cfg.TasksPath(), 2, "task-two")

	tk := &task.Task{ID: 1, DependsOn: []int{2}}

	if err := validateDeps(cfg, tk); err != nil {
		t.Errorf("expected nil for valid depends_on, got %v", err)
	}
}

func TestValidateDeps_InvalidDependsOn(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, DependsOn: []int{99}}

	err = validateDeps(cfg, tk)
	if err == nil {
		t.Fatal("expected error for missing depends_on target")
	}
}

func TestValidateDeps_SelfReferenceParent(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	createTaskFile(t, cfg.TasksPath(), 1, "self-ref")

	parentID := 1
	tk := &task.Task{ID: 1, Parent: &parentID}

	err = validateDeps(cfg, tk)
	if err == nil {
		t.Fatal("expected error for self-referencing parent")
	}
}

// --- runBatch tests ---

// captureStdout replaces os.Stdout with a pipe and returns it.
// The cleanup function restores the original stdout.
func captureStdout(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })
	return r, w
}

// captureStderr replaces os.Stderr with a pipe and returns it.
func captureStderr(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })
	return r, w
}

// drainPipe closes the writer and reads all content from the reader.
func drainPipe(t *testing.T, r, w *os.File) string {
	t.Helper()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// setFlags overrides the global output flags and restores them on cleanup.
func setFlags(t *testing.T, json, table, compact bool) {
	t.Helper()
	oldJSON, oldTable, oldCompact := flagJSON, flagTable, flagCompact
	flagJSON, flagTable, flagCompact = json, table, compact
	t.Cleanup(func() {
		flagJSON, flagTable, flagCompact = oldJSON, oldTable, oldCompact
	})
}

func TestRunBatch_AllSucceed(t *testing.T) {
	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	batchErr := runBatch([]int{1, 2, 3}, func(_ int) error {
		return nil
	})

	got := drainPipe(t, r, w)

	if batchErr != nil {
		t.Errorf("expected nil error when all succeed, got %v", batchErr)
	}
	if !containsSubstring(got, "3/3") {
		t.Errorf("expected '3/3' in output, got: %s", got)
	}
}

func TestRunBatch_SomeFail(t *testing.T) {
	setFlags(t, false, true, false)
	rOut, wOut := captureStdout(t)
	rErr, wErr := captureStderr(t)

	const failID = 2
	batchErr := runBatch([]int{1, failID, 3}, func(id int) error {
		if id == failID {
			return clierr.New(clierr.TaskNotFound, "task not found")
		}
		return nil
	})

	stdout := drainPipe(t, rOut, wOut)
	stderr := drainPipe(t, rErr, wErr)

	if batchErr == nil {
		t.Fatal("expected error when some operations fail")
	}
	var silent *clierr.SilentError
	if !errors.As(batchErr, &silent) {
		t.Fatalf("expected SilentError, got %T", batchErr)
	}
	if silent.Code != 1 {
		t.Errorf("exit code = %d, want 1", silent.Code)
	}

	if !containsSubstring(stdout, "2/3") {
		t.Errorf("expected '2/3' in stdout, got: %s", stdout)
	}

	if !containsSubstring(stderr, "task not found") {
		t.Errorf("expected error detail in stderr, got: %s", stderr)
	}
}

func TestRunBatch_AllFail(t *testing.T) {
	setFlags(t, false, true, false)
	rOut, wOut := captureStdout(t)
	rErr, wErr := captureStderr(t)

	batchErr := runBatch([]int{1, 2}, func(_ int) error {
		return errors.New("generic error")
	})

	stdout := drainPipe(t, rOut, wOut)
	_ = drainPipe(t, rErr, wErr) // drain stderr

	if batchErr == nil {
		t.Fatal("expected error when all operations fail")
	}
	if !containsSubstring(stdout, "0/2") {
		t.Errorf("expected '0/2' in stdout, got: %s", stdout)
	}
}

func TestRunBatch_JSONOutput(t *testing.T) {
	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	batchErr := runBatch([]int{1}, func(_ int) error {
		return nil
	})

	got := drainPipe(t, r, w)

	if batchErr != nil {
		t.Errorf("expected nil error, got %v", batchErr)
	}
	// JSON output should contain the batch result object.
	if !containsSubstring(got, `"ok": true`) {
		t.Errorf("expected JSON with ok:true, got: %s", got)
	}
}

func TestRunBatch_CompactOutput(t *testing.T) {
	setFlags(t, false, false, true)
	r, w := captureStdout(t)

	batchErr := runBatch([]int{1, 2}, func(_ int) error {
		return nil
	})

	got := drainPipe(t, r, w)

	if batchErr != nil {
		t.Errorf("expected nil error, got %v", batchErr)
	}
	// Compact output uses the same text path as table.
	if !containsSubstring(got, "2/2") {
		t.Errorf("expected '2/2' in output, got: %s", got)
	}
}

func TestRunBatch_JSONOutputWithCliError(t *testing.T) {
	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	_ = runBatch([]int{1}, func(_ int) error {
		return clierr.New(clierr.TaskNotFound, "not found")
	})

	got := drainPipe(t, r, w)

	if !containsSubstring(got, `"ok": false`) {
		t.Errorf("expected JSON with ok:false, got: %s", got)
	}
	if !containsSubstring(got, clierr.TaskNotFound) {
		t.Errorf("expected error code in JSON, got: %s", got)
	}
}

// --- helpers ---

// createTaskFile creates a minimal task markdown file in the given directory.
func createTaskFile(t *testing.T, tasksDir string, id int, title string) {
	t.Helper()
	slug := task.GenerateSlug(title)
	filename := task.GenerateFilename(id, slug)
	content := "---\nid: " + idStr(id) + "\ntitle: " + title + "\nstatus: backlog\npriority: medium\ncreated: 2025-01-01T00:00:00Z\nupdated: 2025-01-01T00:00:00Z\n---\n"
	path := filepath.Join(tasksDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func idStr(id int) string {
	return strconv.Itoa(id)
}
