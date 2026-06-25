package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// Shared test constants for the cmd package.
const (
	testBoardName    = "TestBoard"
	priorityHigh     = "high"
	statusInProgress = "in-progress"
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

// --- logActivity tests ---

func TestLogActivity_WritesLogEntry(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	board.LogMutation(cfg.Dir(), "create", 1, "test detail")

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

// --- runBatch tests ---

// pipeReaders maps pipe read-ends to their async drain goroutines.
// This prevents pipe buffer deadlocks on Windows where the default
// anonymous pipe buffer is only 4 KB. Without an async reader, writing
// more than 4 KB to the pipe blocks the writer indefinitely because
// drainPipe (which reads the pipe) hasn't been called yet.
var pipeReaders sync.Map // *os.File → chan string

// captureStdout replaces os.Stdout with a pipe and returns it.
// An async reader goroutine drains the read end to prevent deadlocks.
func captureStdout(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })
	startPipeReader(r)
	return r, w
}

// captureStderr replaces os.Stderr with a pipe and returns it.
// An async reader goroutine drains the read end to prevent deadlocks.
func captureStderr(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })
	startPipeReader(r)
	return r, w
}

// startPipeReader spawns a goroutine that reads all data from the pipe
// read-end into a buffer, storing the result channel in pipeReaders.
func startPipeReader(r *os.File) {
	ch := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		ch <- buf.String()
	}()
	pipeReaders.Store(r, ch)
}

// drainPipe closes the writer, waits for the async reader goroutine
// to finish, and returns the captured output.
func drainPipe(t *testing.T, r, w *os.File) string {
	t.Helper()
	_ = w.Close()
	v, ok := pipeReaders.LoadAndDelete(r)
	if !ok {
		t.Fatal("drainPipe: no async reader for pipe — was captureStdout/captureStderr called?")
	}
	return <-(v.(chan string))
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
