package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// binPath holds the path to the compiled kanban-md binary.
var binPath string

// Constants used in multiple tests.
const (
	codeWIPLimitExceeded = "WIP_LIMIT_EXCEEDED"
	codeInvalidInput     = "INVALID_INPUT"
	codeInvalidDate      = "INVALID_DATE"
	statusBacklog        = "backlog"
	priorityHigh         = "high"
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "kanban-md-e2e-*")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}

	binName := "kanban-md"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath = filepath.Join(tmp, binName)

	// Build with -cover when GOCOVERDIR is requested. The coverage-instrumented
	// binary writes raw coverage data to the directory specified by GOCOVERDIR.
	buildArgs := []string{"build", "-o", binPath}
	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir != "" {
		buildArgs = append(buildArgs, "-cover",
			"-coverpkg=github.com/antopolskiy/kanban-md/...")
	}
	buildArgs = append(buildArgs, "../cmd/kanban-md")

	//nolint:gosec,noctx // building test binary in TestMain (no context available)
	build := exec.Command("go", buildArgs...)
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("building binary: " + err.Error())
	}

	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// result captures command execution output.
type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// taskJSON mirrors the task JSON output schema.
type taskJSON struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Priority string   `json:"priority"`
	Assignee string   `json:"assignee,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Due      string   `json:"due,omitempty"`
	Estimate string   `json:"estimate,omitempty"`
	Body     string   `json:"body,omitempty"`
	File     string   `json:"file,omitempty"`
	Created  string   `json:"created"`
	Updated  string   `json:"updated"`
}

// runKanban executes the binary with --dir prepended for test isolation.
func runKanban(t *testing.T, dir string, args ...string) result {
	t.Helper()

	fullArgs := append([]string{"--dir", dir}, args...)
	cmd := exec.Command(binPath, fullArgs...) //nolint:gosec,noctx // e2e test binary

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	r := result{
		stdout: stdout.String(),
		stderr: stderr.String(),
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			r.exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running kanban-md: %v", err)
		}
	}

	return r
}

// runKanbanEnv runs the kanban-md binary with extra environment variables.
func runKanbanEnv(t *testing.T, dir string, env []string, args ...string) result {
	t.Helper()

	fullArgs := append([]string{"--dir", dir}, args...)
	cmd := exec.Command(binPath, fullArgs...) //nolint:gosec,noctx // e2e test binary
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	r := result{
		stdout: stdout.String(),
		stderr: stderr.String(),
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			r.exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running kanban-md: %v", err)
		}
	}

	return r
}

// runKanbanJSON runs with --json and unmarshals stdout into dest.
func runKanbanJSON(t *testing.T, dir string, dest interface{}, args ...string) result {
	t.Helper()

	jsonArgs := append([]string{"--json"}, args...)
	r := runKanban(t, dir, jsonArgs...)

	if r.exitCode != 0 {
		return r
	}

	if err := json.Unmarshal([]byte(r.stdout), dest); err != nil {
		t.Fatalf("parsing JSON output: %v\nstdout: %s", err, r.stdout)
	}

	return r
}

// errorJSON captures the structured error JSON output.
type errorJSON struct {
	Error   string         `json:"error"`
	Code    string         `json:"code"`
	Details map[string]any `json:"details,omitempty"`
}

// runKanbanJSONError runs with --json and expects a non-zero exit code.
// It parses the structured error from stdout.
func runKanbanJSONError(t *testing.T, dir string, args ...string) errorJSON {
	t.Helper()

	jsonArgs := append([]string{"--json"}, args...)
	r := runKanban(t, dir, jsonArgs...)

	if r.exitCode == 0 {
		t.Fatalf("expected non-zero exit code, got 0\nstdout: %s", r.stdout)
	}

	var errResp errorJSON
	if err := json.Unmarshal([]byte(r.stdout), &errResp); err != nil {
		t.Fatalf("parsing error JSON: %v\nstdout: %s", err, r.stdout)
	}

	return errResp
}

// initBoard initializes a board in a fresh temp directory, returns kanban dir path.
func initBoard(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	cmd := exec.Command(binPath, "--dir", kanbanDir, "init") //nolint:gosec,noctx // e2e test binary

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("init board: %v\nstderr: %s", err, stderr.String())
	}

	return kanbanDir
}

// mustCreateTask creates a task and returns its parsed JSON.
func mustCreateTask(t *testing.T, dir, title string, extraArgs ...string) taskJSON {
	t.Helper()

	args := append([]string{"create", title}, extraArgs...)
	var task taskJSON
	r := runKanbanJSON(t, dir, &task, args...)
	if r.exitCode != 0 {
		t.Fatalf("create task %q failed (exit %d): %s", title, r.exitCode, r.stderr)
	}

	return task
}

// ---------------------------------------------------------------------------
// Init tests
// ---------------------------------------------------------------------------

func TestInitDefault(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	var got map[string]string
	r := runKanbanJSON(t, kanbanDir, &got, "init")

	if r.exitCode != 0 {
		t.Fatalf("init failed (exit %d): %s", r.exitCode, r.stderr)
	}

	if got["status"] != "initialized" {
		t.Errorf("status = %q, want %q", got["status"], "initialized")
	}

	// Verify files on disk.
	if _, err := os.Stat(filepath.Join(kanbanDir, "config.yml")); err != nil {
		t.Errorf("config.yml not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(kanbanDir, "tasks")); err != nil {
		t.Errorf("tasks/ not found: %v", err)
	}
}

func TestInitWithName(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	var got map[string]string
	runKanbanJSON(t, kanbanDir, &got, "init", "--name", "My Project")

	if got["name"] != "My Project" {
		t.Errorf("name = %q, want %q", got["name"], "My Project")
	}
}

func TestInitCustomStatuses(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	var got map[string]string
	runKanbanJSON(t, kanbanDir, &got, "init", "--statuses", "open,closed")

	if got["columns"] != "open,closed" {
		t.Errorf("columns = %q, want %q", got["columns"], "open,closed")
	}
}

func TestInitAlreadyInitialized(t *testing.T) {
	kanbanDir := initBoard(t)
	r := runKanban(t, kanbanDir, "init")

	if r.exitCode == 0 {
		t.Error("expected non-zero exit code for double init")
	}
	if !strings.Contains(r.stderr, "already initialized") {
		t.Errorf("stderr = %q, want 'already initialized'", r.stderr)
	}
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestCreateBasic(t *testing.T) {
	kanbanDir := initBoard(t)

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "create", "My first task")

	if r.exitCode != 0 {
		t.Fatalf("create failed: %s", r.stderr)
	}
	if task.ID != 1 {
		t.Errorf("ID = %d, want 1", task.ID)
	}
	if task.Title != "My first task" {
		t.Errorf("Title = %q, want %q", task.Title, "My first task")
	}
	if task.Status != statusBacklog {
		t.Errorf("Status = %q, want %q (default)", task.Status, statusBacklog)
	}
	if task.Priority != "medium" {
		t.Errorf("Priority = %q, want %q (default)", task.Priority, "medium")
	}
}

func TestCreateWithAllFlags(t *testing.T) {
	kanbanDir := initBoard(t)

	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "create", "Full task",
		"--status", "todo",
		"--priority", "high",
		"--assignee", "alice",
		"--tags", "backend,api",
		"--due", "2026-03-15",
		"--estimate", "4h",
		"--body", "Task description here",
	)

	if task.Status != "todo" { //nolint:goconst // test data
		t.Errorf("Status = %q, want %q", task.Status, "todo")
	}
	if task.Priority != priorityHigh {
		t.Errorf("Priority = %q, want %q", task.Priority, priorityHigh)
	}
	if task.Assignee != "alice" { //nolint:goconst // test data
		t.Errorf("Assignee = %q, want %q", task.Assignee, "alice")
	}
	if len(task.Tags) != 2 || task.Tags[0] != "backend" || task.Tags[1] != "api" {
		t.Errorf("Tags = %v, want [backend api]", task.Tags)
	}
	if task.Due != "2026-03-15" {
		t.Errorf("Due = %q, want %q", task.Due, "2026-03-15")
	}
	if task.Estimate != "4h" {
		t.Errorf("Estimate = %q, want %q", task.Estimate, "4h")
	}
	if task.Body != "Task description here" {
		t.Errorf("Body = %q, want %q", task.Body, "Task description here")
	}
}

func TestCreateTagAlias(t *testing.T) {
	kanbanDir := initBoard(t)

	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "create", "Tag alias test", "--tag", "bug,feature")

	if len(task.Tags) != 2 || task.Tags[0] != "bug" || task.Tags[1] != "feature" {
		t.Errorf("Tags = %v, want [bug feature]", task.Tags)
	}
}

func TestCreateIncrementID(t *testing.T) {
	kanbanDir := initBoard(t)

	t1 := mustCreateTask(t, kanbanDir, "First")
	t2 := mustCreateTask(t, kanbanDir, "Second")
	t3 := mustCreateTask(t, kanbanDir, "Third")

	if t1.ID != 1 || t2.ID != 2 || t3.ID != 3 {
		t.Errorf("IDs = [%d, %d, %d], want [1, 2, 3]", t1.ID, t2.ID, t3.ID)
	}
}

func TestCreateInvalidStatus(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "create", "Bad task", "--status", "nonexistent")
	if errResp.Code != "INVALID_STATUS" {
		t.Errorf("code = %q, want INVALID_STATUS", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "invalid status") {
		t.Errorf("error = %q, want 'invalid status'", errResp.Error)
	}
}

func TestCreateBadDateFormat(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "create", "Bad date", "--due", "02-15-2026")
	if errResp.Code != codeInvalidDate {
		t.Errorf("code = %q, want INVALID_DATE", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "invalid") {
		t.Errorf("error = %q, want 'invalid'", errResp.Error)
	}
}

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestListEmpty(t *testing.T) {
	kanbanDir := initBoard(t)

	var tasks []taskJSON
	runKanbanJSON(t, kanbanDir, &tasks, "list")
	if len(tasks) != 0 {
		t.Errorf("list returned %d tasks, want 0", len(tasks))
	}

	// Table output writes "No tasks found." to stderr.
	r := runKanban(t, kanbanDir, "--table", "list")
	if !strings.Contains(r.stderr, "No tasks found.") {
		t.Errorf("stderr = %q, want 'No tasks found.'", r.stderr)
	}
}

func TestListFilters(t *testing.T) {
	kanbanDir := initBoard(t)

	mustCreateTask(t, kanbanDir, "Backend API", "--status", "todo", "--priority", "high",
		"--assignee", "alice", "--tags", "backend,api")
	mustCreateTask(t, kanbanDir, "Frontend UI", "--status", "in-progress", "--priority", "medium",
		"--assignee", "bob", "--tags", "frontend")
	mustCreateTask(t, kanbanDir, "Database Migration", "--status", "todo", "--priority", "critical",
		"--assignee", "alice", "--tags", "backend,database")
	mustCreateTask(t, kanbanDir, "Docs Update", "--status", "done", "--priority", "low",
		"--tags", "docs")

	tests := []struct {
		name    string
		args    []string
		wantIDs []int
	}{
		{"status todo", []string{"--status", "todo"}, []int{1, 3}},
		{"multiple statuses", []string{"--status", "todo,done"}, []int{1, 3, 4}},
		{"assignee alice", []string{"--assignee", "alice"}, []int{1, 3}},
		{"tag backend", []string{"--tag", "backend"}, []int{1, 3}},
		{"priority high", []string{"--priority", "high"}, []int{1}},
		{"status+assignee", []string{"--status", "todo", "--assignee", "alice"}, []int{1, 3}},
		{"assignee+tag", []string{"--assignee", "alice", "--tag", "api"}, []int{1}},
		{"no match", []string{"--assignee", "nobody"}, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"list"}, tt.args...)
			var tasks []taskJSON
			runKanbanJSON(t, kanbanDir, &tasks, args...)

			gotIDs := make([]int, len(tasks))
			for i, task := range tasks {
				gotIDs[i] = task.ID
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("got IDs %v, want %v", gotIDs, tt.wantIDs)
			}
			for i, id := range gotIDs {
				if id != tt.wantIDs[i] {
					t.Errorf("task[%d].ID = %d, want %d", i, id, tt.wantIDs[i])
				}
			}
		})
	}
}

func TestListSortAndLimit(t *testing.T) {
	kanbanDir := initBoard(t)

	mustCreateTask(t, kanbanDir, "C task", "--priority", "low")
	mustCreateTask(t, kanbanDir, "A task", "--priority", "high")
	mustCreateTask(t, kanbanDir, "B task", "--priority", "critical")

	tests := []struct {
		name    string
		args    []string
		wantIDs []int
	}{
		{"default sort by id", nil, []int{1, 2, 3}},
		{"sort id reverse", []string{"--sort", "id", "--reverse"}, []int{3, 2, 1}},
		{"sort by priority", []string{"--sort", "priority"}, []int{1, 2, 3}},
		{"limit 2", []string{"--limit", "2"}, []int{1, 2}},
		{"reverse + limit 1", []string{"--sort", "id", "--reverse", "--limit", "1"}, []int{3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"list"}, tt.args...)
			var tasks []taskJSON
			runKanbanJSON(t, kanbanDir, &tasks, args...)

			gotIDs := make([]int, len(tasks))
			for i, task := range tasks {
				gotIDs[i] = task.ID
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("got IDs %v, want %v", gotIDs, tt.wantIDs)
			}
			for i, id := range gotIDs {
				if id != tt.wantIDs[i] {
					t.Errorf("position %d: got ID %d, want %d", i, id, tt.wantIDs[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Show tests
// ---------------------------------------------------------------------------

func TestShow(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Show me", "--body", "Detailed description",
		"--assignee", "alice", "--tags", "test")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "show", "1")

	if r.exitCode != 0 {
		t.Fatalf("show failed: %s", r.stderr)
	}
	if task.ID != 1 {
		t.Errorf("ID = %d, want 1", task.ID)
	}
	if task.Title != "Show me" {
		t.Errorf("Title = %q, want %q", task.Title, "Show me")
	}
	if !strings.Contains(task.Body, "Detailed description") {
		t.Errorf("Body = %q, want to contain %q", task.Body, "Detailed description")
	}
	if task.Assignee != "alice" {
		t.Errorf("Assignee = %q, want %q", task.Assignee, "alice")
	}
}

func TestShowNotFound(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "show", "999")
	if errResp.Code != "TASK_NOT_FOUND" {
		t.Errorf("code = %q, want TASK_NOT_FOUND", errResp.Code)
	}
}

func TestShowInvalidID(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "show", "abc")
	if errResp.Code != "INVALID_TASK_ID" {
		t.Errorf("code = %q, want INVALID_TASK_ID", errResp.Code)
	}
}

// ---------------------------------------------------------------------------
// Edit tests
// ---------------------------------------------------------------------------

func TestEditFields(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Edit target", "--tags", "original")

	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, task taskJSON)
	}{
		{
			name: "change status",
			args: []string{"--status", "todo"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				if task.Status != "todo" {
					t.Errorf("Status = %q, want %q", task.Status, "todo")
				}
			},
		},
		{
			name: "change priority",
			args: []string{"--priority", "high"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				if task.Priority != priorityHigh {
					t.Errorf("Priority = %q, want %q", task.Priority, priorityHigh)
				}
			},
		},
		{
			name: "add tag",
			args: []string{"--add-tag", "newtag"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				found := false
				for _, tag := range task.Tags {
					if tag == "newtag" {
						found = true
					}
				}
				if !found {
					t.Errorf("Tags %v missing %q", task.Tags, "newtag")
				}
			},
		},
		{
			name: "set due date",
			args: []string{"--due", "2026-06-15"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				if task.Due != "2026-06-15" {
					t.Errorf("Due = %q, want %q", task.Due, "2026-06-15")
				}
			},
		},
		{
			name: "set body",
			args: []string{"--body", "Updated body content"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				if !strings.Contains(task.Body, "Updated body content") {
					t.Errorf("Body = %q, want %q", task.Body, "Updated body content")
				}
			},
		},
	}

	// These run sequentially on the same task — each edit builds on previous state.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editArgs := append([]string{"edit", "1"}, tt.args...)
			var task taskJSON
			r := runKanbanJSON(t, kanbanDir, &task, editArgs...)
			if r.exitCode != 0 {
				t.Fatalf("edit failed: %s", r.stderr)
			}
			tt.check(t, task)
		})
	}
}

func TestEditTitleRename(t *testing.T) {
	kanbanDir := initBoard(t)
	original := mustCreateTask(t, kanbanDir, "Original title")

	var updated taskJSON
	runKanbanJSON(t, kanbanDir, &updated, "edit", "1", "--title", "New title")

	if updated.Title != "New title" {
		t.Errorf("Title = %q, want %q", updated.Title, "New title")
	}

	// Old file removed.
	if _, err := os.Stat(original.File); !os.IsNotExist(err) {
		t.Errorf("old file %q still exists", original.File)
	}

	// New file exists with correct slug.
	if _, err := os.Stat(updated.File); err != nil {
		t.Errorf("new file %q not found: %v", updated.File, err)
	}
	if !strings.Contains(filepath.Base(updated.File), "new-title") {
		t.Errorf("filename %q missing 'new-title'", filepath.Base(updated.File))
	}
}

func TestEditNoChanges(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Stable task")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1")
	if errResp.Code != "NO_CHANGES" {
		t.Errorf("code = %q, want NO_CHANGES", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "no changes") {
		t.Errorf("error = %q, want 'no changes'", errResp.Error)
	}
}

func TestEditClearDue(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Dated task", "--due", "2026-03-15")

	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--clear-due")

	if task.Due != "" {
		t.Errorf("Due = %q, want empty (cleared)", task.Due)
	}
}

// ---------------------------------------------------------------------------
// Move tests
// ---------------------------------------------------------------------------

func TestMoveDirectStatus(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Movable task")

	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "move", "1", "in-progress")

	if task.Status != "in-progress" {
		t.Errorf("Status = %q, want %q", task.Status, "in-progress")
	}
}

func TestMoveNextPrev(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Walk task") // starts at "backlog"

	var task taskJSON

	// backlog -> todo
	runKanbanJSON(t, kanbanDir, &task, "move", "1", "--next")
	if task.Status != "todo" {
		t.Errorf("after --next: Status = %q, want %q", task.Status, "todo")
	}

	// todo -> in-progress
	runKanbanJSON(t, kanbanDir, &task, "move", "1", "--next")
	if task.Status != "in-progress" {
		t.Errorf("after second --next: Status = %q, want %q", task.Status, "in-progress")
	}

	// in-progress -> todo
	runKanbanJSON(t, kanbanDir, &task, "move", "1", "--prev")
	if task.Status != "todo" {
		t.Errorf("after --prev: Status = %q, want %q", task.Status, "todo")
	}
}

func TestMoveBoundaryErrors(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Boundary task") // starts at "backlog" (first)

	// --prev at first status.
	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "--prev")
	if errResp.Code != "BOUNDARY_ERROR" {
		t.Errorf("code = %q, want BOUNDARY_ERROR", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "first") {
		t.Errorf("error = %q, want 'first'", errResp.Error)
	}

	// Move to last status, then try --next.
	runKanban(t, kanbanDir, "--json", "move", "1", "done")
	errResp = runKanbanJSONError(t, kanbanDir, "move", "1", "--next")
	if errResp.Code != "BOUNDARY_ERROR" {
		t.Errorf("code = %q, want BOUNDARY_ERROR", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "last") {
		t.Errorf("error = %q, want 'last'", errResp.Error)
	}
}

func TestMoveNoStatusSpecified(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "No direction")

	errResp := runKanbanJSONError(t, kanbanDir, "move", "1")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "provide a target status") {
		t.Errorf("error = %q, want 'provide a target status'", errResp.Error)
	}
}

func TestMoveIdempotentJSON(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Idempotent task") // starts at "backlog"

	// Move to same status should succeed with changed=false.
	var got struct {
		taskJSON
		Changed bool `json:"changed"`
	}
	r := runKanbanJSON(t, kanbanDir, &got, "move", "1", "backlog")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", r.exitCode, r.stderr)
	}
	if got.Changed {
		t.Error("Changed = true, want false for same-status move")
	}
	if got.Status != statusBacklog {
		t.Errorf("Status = %q, want %q", got.Status, statusBacklog)
	}
}

func TestMoveIdempotentHumanOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Idempotent human") // starts at "backlog"

	r := runKanban(t, kanbanDir, "--table", "move", "1", "backlog")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "already at") {
		t.Errorf("stdout = %q, want 'already at'", r.stdout)
	}
}

func TestMoveChangedTrue(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Changed task") // starts at "backlog"

	var got struct {
		taskJSON
		Changed bool `json:"changed"`
	}
	r := runKanbanJSON(t, kanbanDir, &got, "move", "1", "todo")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", r.exitCode, r.stderr)
	}
	if !got.Changed {
		t.Error("Changed = false, want true for status change")
	}
	if got.Status != "todo" {
		t.Errorf("Status = %q, want %q", got.Status, "todo")
	}
}

// ---------------------------------------------------------------------------
// WIP limit tests
// ---------------------------------------------------------------------------

// initBoardWithWIP creates a board with WIP limits on in-progress.
func initBoardWithWIP(t *testing.T, limit int) string {
	t.Helper()

	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	args := []string{
		"--dir", kanbanDir, "init",
		"--wip-limit", "in-progress:" + strconv.Itoa(limit),
	}
	cmd := exec.Command(binPath, args...) //nolint:gosec,noctx // e2e test binary

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("init board with WIP: %v\nstderr: %s", err, stderr.String())
	}

	return kanbanDir
}

func TestMoveRespectsWIPLimit(t *testing.T) {
	kanbanDir := initBoardWithWIP(t, 1) // limit 1 for in-progress

	// Create a task and move it to in-progress (fills the slot).
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")

	// Create another task and try to move it to in-progress.
	mustCreateTask(t, kanbanDir, "Task B")
	errResp := runKanbanJSONError(t, kanbanDir, "move", "2", "in-progress")
	if errResp.Code != codeWIPLimitExceeded {
		t.Errorf("code = %q, want WIP_LIMIT_EXCEEDED", errResp.Code)
	}
}

func TestMoveForceOverridesWIP(t *testing.T) {
	kanbanDir := initBoardWithWIP(t, 1)

	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")

	mustCreateTask(t, kanbanDir, "Task B")
	r := runKanban(t, kanbanDir, "--json", "move", "2", "in-progress", "--force")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 with --force\nstderr: %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stderr, "Warning") {
		t.Errorf("stderr = %q, want WIP warning", r.stderr)
	}
}

func TestCreateRespectsWIPLimit(t *testing.T) {
	kanbanDir := initBoardWithWIP(t, 2)

	// Fill in-progress to the limit.
	mustCreateTask(t, kanbanDir, "Task A", "--status", "in-progress")
	mustCreateTask(t, kanbanDir, "Task B", "--status", "in-progress")

	// Third task to in-progress should fail.
	errResp := runKanbanJSONError(t, kanbanDir, "create", "Task C", "--status", "in-progress")
	if errResp.Code != codeWIPLimitExceeded {
		t.Errorf("code = %q, want WIP_LIMIT_EXCEEDED", errResp.Code)
	}
}

func TestEditStatusRespectsWIPLimit(t *testing.T) {
	kanbanDir := initBoardWithWIP(t, 1)

	mustCreateTask(t, kanbanDir, "Task A", "--status", "in-progress")
	mustCreateTask(t, kanbanDir, "Task B")

	// Edit task B status to in-progress should fail.
	errResp := runKanbanJSONError(t, kanbanDir, "edit", "2", "--status", "in-progress")
	if errResp.Code != codeWIPLimitExceeded {
		t.Errorf("code = %q, want WIP_LIMIT_EXCEEDED", errResp.Code)
	}
}

func TestEditForceOverridesWIP(t *testing.T) {
	kanbanDir := initBoardWithWIP(t, 1)

	mustCreateTask(t, kanbanDir, "Task A", "--status", "in-progress")
	mustCreateTask(t, kanbanDir, "Task B")

	// Without --force, edit should fail.
	errResp := runKanbanJSONError(t, kanbanDir, "edit", "2", "--status", "in-progress")
	if errResp.Code != codeWIPLimitExceeded {
		t.Fatalf("code = %q, want WIP_LIMIT_EXCEEDED", errResp.Code)
	}

	// With --force, edit should succeed with a warning.
	r := runKanban(t, kanbanDir, "edit", "2", "--status", "in-progress", "--force")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 with --force\nstderr: %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stderr, "Warning") {
		t.Errorf("stderr = %q, want WIP warning", r.stderr)
	}
}

func TestWIPUnlimitedByDefault(t *testing.T) {
	kanbanDir := initBoard(t) // no WIP limits

	// Create many tasks in in-progress — should all succeed.
	for i := 1; i <= 5; i++ {
		mustCreateTask(t, kanbanDir, "Task "+strconv.Itoa(i), "--status", "in-progress")
	}
}

func TestInitWithWIPLimits(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	r := runKanban(t, kanbanDir, "--json", "init",
		"--wip-limit", "in-progress:3",
		"--wip-limit", "review:2")
	if r.exitCode != 0 {
		t.Fatalf("init with WIP limits failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// Create a task to verify the board works.
	mustCreateTask(t, kanbanDir, "Test task")
}

// ---------------------------------------------------------------------------
// Timestamp tests
// ---------------------------------------------------------------------------

func TestMoveStartedTimestamp(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	// Move from backlog (initial) to todo — should set started.
	var task map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", "todo")
	if r.exitCode != 0 {
		t.Fatalf("move failed: %s", r.stderr)
	}
	if task["started"] == nil {
		t.Error("started should be set on first move from initial status")
	}
}

func TestMoveCompletedTimestamp(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "move", "1", "todo")

	// Move to done (terminal) — should set completed.
	var task map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", "done")
	if r.exitCode != 0 {
		t.Fatalf("move failed: %s", r.stderr)
	}
	if task["completed"] == nil {
		t.Error("completed should be set on move to terminal status")
	}
}

func TestMoveBackClearsCompleted(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "move", "1", "done")

	// Move back from done — should clear completed.
	var task map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", "review")
	if r.exitCode != 0 {
		t.Fatalf("move failed: %s", r.stderr)
	}
	if task["completed"] != nil {
		t.Error("completed should be cleared when moving back from terminal")
	}
	if task["started"] == nil {
		t.Error("started should be preserved when moving back from terminal")
	}
}

func TestMoveStartedNeverOverwritten(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	// First move: backlog -> todo (sets started).
	var first map[string]interface{}
	runKanbanJSON(t, kanbanDir, &first, "move", "1", "todo")
	started1 := first["started"]

	// Second move: todo -> in-progress (should NOT change started).
	var second map[string]interface{}
	runKanbanJSON(t, kanbanDir, &second, "move", "1", "in-progress")
	started2 := second["started"]

	if started1 != started2 {
		t.Errorf("started changed: %v → %v", started1, started2)
	}
}

func TestMoveDirectToTerminal(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	// Move directly from backlog to done.
	var task map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", "done")
	if r.exitCode != 0 {
		t.Fatalf("move failed: %s", r.stderr)
	}
	if task["started"] == nil {
		t.Error("started should be set on direct move to terminal")
	}
	if task["completed"] == nil {
		t.Error("completed should be set on direct move to terminal")
	}
}

func TestEditStartedManualBackfill(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--started", "2026-01-15")
	if r.exitCode != 0 {
		t.Fatalf("edit --started failed: %s", r.stderr)
	}
	started, ok := edited["started"].(string)
	if !ok || !strings.HasPrefix(started, "2026-01-15") {
		t.Errorf("started = %v, want prefix 2026-01-15", edited["started"])
	}
}

func TestEditCompletedManualBackfill(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--completed", "2026-02-01")
	if r.exitCode != 0 {
		t.Fatalf("edit --completed failed: %s", r.stderr)
	}
	completed, ok := edited["completed"].(string)
	if !ok || !strings.HasPrefix(completed, "2026-02-01") {
		t.Errorf("completed = %v, want prefix 2026-02-01", edited["completed"])
	}
}

func TestEditClearStarted(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--started", "2026-01-15")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--clear-started")
	if r.exitCode != 0 {
		t.Fatalf("edit --clear-started failed: %s", r.stderr)
	}
	if edited["started"] != nil {
		t.Errorf("started = %v, want nil after clear", edited["started"])
	}
}

func TestEditClearCompleted(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--completed", "2026-02-01")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--clear-completed")
	if r.exitCode != 0 {
		t.Fatalf("edit --clear-completed failed: %s", r.stderr)
	}
	if edited["completed"] != nil {
		t.Errorf("completed = %v, want nil after clear", edited["completed"])
	}
}

func TestEditStartedAndClearStartedConflict(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--started", "2026-01-15", "--clear-started")
	if errResp.Code != "STATUS_CONFLICT" {
		t.Errorf("code = %q, want STATUS_CONFLICT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "cannot use") {
		t.Errorf("error = %q, want conflict error", errResp.Error)
	}
}

func TestShowDisplaysLeadCycleTime(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "move", "1", "todo")
	runKanban(t, kanbanDir, "--json", "move", "1", "done")

	r := runKanban(t, kanbanDir, "--table", "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Lead time") {
		t.Errorf("show output missing 'Lead time', got: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Cycle time") {
		t.Errorf("show output missing 'Cycle time', got: %s", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Dependency tests
// ---------------------------------------------------------------------------

func TestCreateWithParent(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Parent task")

	var child map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &child, "create", "Child task", "--parent", "1")
	if r.exitCode != 0 {
		t.Fatalf("create with parent failed: %s", r.stderr)
	}
	if child["parent"] != float64(1) {
		t.Errorf("parent = %v, want 1", child["parent"])
	}
}

func TestCreateWithDependsOn(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Dep A")
	mustCreateTask(t, kanbanDir, "Dep B")

	var child map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &child, "create", "Dependent", "--depends-on", "1,2")
	if r.exitCode != 0 {
		t.Fatalf("create with depends-on failed: %s", r.stderr)
	}
	deps, ok := child["depends_on"].([]interface{})
	if !ok || len(deps) != 2 {
		t.Errorf("depends_on = %v, want [1,2]", child["depends_on"])
	}
}

func TestCreateSelfDepErrors(t *testing.T) {
	kanbanDir := initBoard(t)
	// Task 1 will be created, then try to create task 2 depending on itself (ID 2).
	mustCreateTask(t, kanbanDir, "Existing task")

	// Next ID is 2. --depends-on 2 is self-reference.
	errResp := runKanbanJSONError(t, kanbanDir, "create", "Self dep", "--depends-on", "2")
	if errResp.Code != "SELF_REFERENCE" {
		t.Errorf("code = %q, want SELF_REFERENCE", errResp.Code)
	}
}

func TestCreateInvalidDepErrors(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "create", "Bad dep", "--depends-on", "99")
	if errResp.Code != "DEPENDENCY_NOT_FOUND" {
		t.Errorf("code = %q, want DEPENDENCY_NOT_FOUND", errResp.Code)
	}
}

func TestEditAddDep(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "2", "--add-dep", "1")
	if r.exitCode != 0 {
		t.Fatalf("edit add-dep failed: %s", r.stderr)
	}
	deps, ok := edited["depends_on"].([]interface{})
	if !ok || len(deps) != 1 || deps[0] != float64(1) {
		t.Errorf("depends_on = %v, want [1]", edited["depends_on"])
	}
}

func TestEditRemoveDep(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B", "--depends-on", "1")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "2", "--remove-dep", "1")
	if r.exitCode != 0 {
		t.Fatalf("edit remove-dep failed: %s", r.stderr)
	}
	// depends_on should be empty or absent.
	deps, _ := edited["depends_on"].([]interface{})
	if len(deps) != 0 {
		t.Errorf("depends_on = %v, want empty", edited["depends_on"])
	}
}

func TestEditSetParent(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Parent")
	mustCreateTask(t, kanbanDir, "Child")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "2", "--parent", "1")
	if r.exitCode != 0 {
		t.Fatalf("edit set parent failed: %s", r.stderr)
	}
	if edited["parent"] != float64(1) {
		t.Errorf("parent = %v, want 1", edited["parent"])
	}
}

func TestEditClearParent(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Parent")
	mustCreateTask(t, kanbanDir, "Child", "--parent", "1")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "2", "--clear-parent")
	if r.exitCode != 0 {
		t.Fatalf("edit clear parent failed: %s", r.stderr)
	}
	if edited["parent"] != nil {
		t.Errorf("parent = %v, want nil", edited["parent"])
	}
}

func TestEditSelfDepErrors(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--add-dep", "1")
	if errResp.Code != "SELF_REFERENCE" {
		t.Errorf("code = %q, want SELF_REFERENCE", errResp.Code)
	}
}

func TestListByParent(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Parent")
	mustCreateTask(t, kanbanDir, "Child A", "--parent", "1")
	mustCreateTask(t, kanbanDir, "Child B", "--parent", "1")
	mustCreateTask(t, kanbanDir, "Orphan")

	var tasks []map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &tasks, "list", "--parent", "1")
	if r.exitCode != 0 {
		t.Fatalf("list --parent failed: %s", r.stderr)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2 with parent 1", len(tasks))
	}
}

func TestListUnblocked(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Dep task")                          // #1
	mustCreateTask(t, kanbanDir, "Depends on 1", "--depends-on", "1") // #2
	mustCreateTask(t, kanbanDir, "No deps")                           // #3

	// Task 1 is in backlog (not done), so task 2 is blocked by deps.
	var tasks []map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &tasks, "list", "--unblocked")
	if r.exitCode != 0 {
		t.Fatalf("list --unblocked failed: %s", r.stderr)
	}
	// Only tasks 1 and 3 should appear (no unsatisfied deps).
	if len(tasks) != 2 {
		t.Errorf("got %d unblocked tasks, want 2", len(tasks))
	}
}

func TestListUnblockedAfterDepDone(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Dep task")                          // #1
	mustCreateTask(t, kanbanDir, "Depends on 1", "--depends-on", "1") // #2

	// Move dep to done.
	runKanban(t, kanbanDir, "--json", "move", "1", "done")

	var tasks []map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &tasks, "list", "--unblocked")
	if r.exitCode != 0 {
		t.Fatalf("list --unblocked failed: %s", r.stderr)
	}
	// Both tasks should now be unblocked.
	if len(tasks) != 2 {
		t.Errorf("got %d unblocked tasks, want 2 (dep satisfied)", len(tasks))
	}
}

func TestDeleteWithDependentsWarns(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Dep task")                          // #1
	mustCreateTask(t, kanbanDir, "Depends on 1", "--depends-on", "1") // #2

	r := runKanban(t, kanbanDir, "--json", "delete", "1", "--force")
	if r.exitCode != 0 {
		t.Fatalf("delete failed: %s", r.stderr)
	}
	if !strings.Contains(r.stderr, "depends on this task") {
		t.Errorf("stderr = %q, want dependent warning", r.stderr)
	}
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestDeleteWithForce(t *testing.T) {
	kanbanDir := initBoard(t)
	created := mustCreateTask(t, kanbanDir, "Doomed task")

	var got map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &got, "delete", "1", "--force")

	if r.exitCode != 0 {
		t.Fatalf("delete failed: %s", r.stderr)
	}
	if got["status"] != "deleted" {
		t.Errorf("status = %v, want %q", got["status"], "deleted")
	}

	// File should be gone.
	if _, err := os.Stat(created.File); !os.IsNotExist(err) {
		t.Errorf("task file %q still exists after delete", created.File)
	}
}

func TestDeleteWithoutForceNonTTY(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Protected task")

	errResp := runKanbanJSONError(t, kanbanDir, "delete", "1")
	if errResp.Code != "CONFIRMATION_REQUIRED" {
		t.Errorf("code = %q, want CONFIRMATION_REQUIRED", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "not a terminal") {
		t.Errorf("error = %q, want 'not a terminal'", errResp.Error)
	}
}

// ---------------------------------------------------------------------------
// Cross-cutting tests
// ---------------------------------------------------------------------------

func TestNoInitErrors(t *testing.T) {
	dir := t.TempDir() // empty, no board

	commands := []struct {
		name string
		args []string
	}{
		{"create", []string{"create", "Task"}},
		{"list", []string{"list"}},
		{"show", []string{"show", "1"}},
		{"edit", []string{"edit", "1", "--title", "New"}},
		{"move", []string{"move", "1", "done"}},
		{"delete", []string{"delete", "1", "--force"}},
	}

	for _, tt := range commands {
		t.Run(tt.name, func(t *testing.T) {
			r := runKanban(t, dir, tt.args...)
			if r.exitCode == 0 {
				t.Errorf("%s succeeded without an initialized board", tt.name)
			}
			if !strings.Contains(r.stderr, "no kanban board found") {
				t.Errorf("stderr = %q, want 'no kanban board found'", r.stderr)
			}
		})
	}
}

func TestCommandAliases(t *testing.T) {
	kanbanDir := initBoard(t)

	// add = create
	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "add", "Aliased task")
	if r.exitCode != 0 {
		t.Fatalf("'add' alias failed: %s", r.stderr)
	}
	if task.Title != "Aliased task" {
		t.Errorf("Title = %q, want %q", task.Title, "Aliased task")
	}

	// ls = list
	var tasks []taskJSON
	r = runKanbanJSON(t, kanbanDir, &tasks, "ls")
	if r.exitCode != 0 {
		t.Fatalf("'ls' alias failed: %s", r.stderr)
	}
	if len(tasks) != 1 {
		t.Errorf("ls returned %d tasks, want 1", len(tasks))
	}

	// rm = delete
	var deleted map[string]interface{}
	r = runKanbanJSON(t, kanbanDir, &deleted, "rm", "1", "--force")
	if r.exitCode != 0 {
		t.Fatalf("'rm' alias failed: %s", r.stderr)
	}
}

// ---------------------------------------------------------------------------
// Workflow & edge case tests
// ---------------------------------------------------------------------------

func TestFullLifecycle(t *testing.T) {
	kanbanDir := initBoard(t)

	// Create.
	task := mustCreateTask(t, kanbanDir, "Lifecycle task",
		"--priority", "high", "--assignee", "alice")
	if task.ID != 1 {
		t.Fatalf("create: ID = %d, want 1", task.ID)
	}

	// List.
	var tasks []taskJSON
	runKanbanJSON(t, kanbanDir, &tasks, "list")
	if len(tasks) != 1 {
		t.Fatalf("list: got %d tasks, want 1", len(tasks))
	}

	// Show.
	var shown taskJSON
	runKanbanJSON(t, kanbanDir, &shown, "show", "1")
	if shown.Assignee != "alice" {
		t.Errorf("show: Assignee = %q, want %q", shown.Assignee, "alice")
	}

	// Edit.
	var edited taskJSON
	runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--priority", "critical")
	if edited.Priority != "critical" {
		t.Errorf("edit: Priority = %q, want %q", edited.Priority, "critical")
	}

	// Move.
	var moved taskJSON
	runKanbanJSON(t, kanbanDir, &moved, "move", "1", "--next")
	if moved.Status != "todo" {
		t.Errorf("move: Status = %q, want %q", moved.Status, "todo")
	}

	// Delete.
	var deleted map[string]interface{}
	runKanbanJSON(t, kanbanDir, &deleted, "delete", "1", "--force")
	if deleted["status"] != "deleted" {
		t.Errorf("delete: status = %v, want %q", deleted["status"], "deleted")
	}

	// List (empty).
	runKanbanJSON(t, kanbanDir, &tasks, "list")
	if len(tasks) != 0 {
		t.Errorf("list after delete: got %d tasks, want 0", len(tasks))
	}
}

func TestCustomStatusesWorkflow(t *testing.T) {
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")

	var initResult map[string]string
	runKanbanJSON(t, kanbanDir, &initResult, "init", "--statuses", "open,wip,closed")

	// Default status is first: "open".
	task := mustCreateTask(t, kanbanDir, "Custom status task")
	if task.Status != "open" {
		t.Errorf("default status = %q, want %q", task.Status, "open")
	}

	// Move next: open -> wip.
	var moved taskJSON
	runKanbanJSON(t, kanbanDir, &moved, "move", "1", "--next")
	if moved.Status != "wip" {
		t.Errorf("after --next: status = %q, want %q", moved.Status, "wip")
	}

	// Move next: wip -> closed.
	runKanbanJSON(t, kanbanDir, &moved, "move", "1", "--next")
	if moved.Status != "closed" {
		t.Errorf("after second --next: status = %q, want %q", moved.Status, "closed")
	}

	// --next at last fails.
	r := runKanban(t, kanbanDir, "--json", "move", "1", "--next")
	if r.exitCode == 0 {
		t.Error("expected failure for --next at last status")
	}

	// Old default statuses rejected.
	r = runKanban(t, kanbanDir, "--json", "create", "Bad status", "--status", "backlog")
	if r.exitCode == 0 {
		t.Error("expected failure for status 'backlog' not in custom statuses")
	}
}

func TestTagOperations(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Tagged task", "--tags", "alpha,beta")

	var task taskJSON

	// Add new tag.
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--add-tag", "gamma")
	if len(task.Tags) != 3 {
		t.Fatalf("after add: Tags = %v, want 3 tags", task.Tags)
	}

	// Add duplicate (should not duplicate).
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--add-tag", "alpha")
	if len(task.Tags) != 3 {
		t.Errorf("after adding duplicate: Tags = %v, want 3 tags still", task.Tags)
	}

	// Remove tag.
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--remove-tag", "beta")
	if len(task.Tags) != 2 {
		t.Errorf("after remove: Tags = %v, want 2 tags", task.Tags)
	}
	for _, tag := range task.Tags {
		if tag == "beta" {
			t.Error("removed tag 'beta' still present")
		}
	}

	// Remove non-existent tag (should succeed).
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--remove-tag", "nonexistent")
	if r.exitCode != 0 {
		t.Errorf("removing non-existent tag failed: %s", r.stderr)
	}
}

func TestDueDateLifecycle(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Due task")

	// Set due date.
	var set taskJSON
	runKanbanJSON(t, kanbanDir, &set, "edit", "1", "--due", "2026-04-01")
	if set.Due != "2026-04-01" {
		t.Errorf("Due = %q, want %q", set.Due, "2026-04-01")
	}

	// Clear due date — use fresh struct so omitted fields don't carry over.
	var cleared taskJSON
	runKanbanJSON(t, kanbanDir, &cleared, "edit", "1", "--clear-due")
	if cleared.Due != "" {
		t.Errorf("Due = %q after clear, want empty", cleared.Due)
	}

	// Re-set due date.
	var reset taskJSON
	runKanbanJSON(t, kanbanDir, &reset, "edit", "1", "--due", "2026-05-01")
	if reset.Due != "2026-05-01" {
		t.Errorf("Due = %q, want %q", reset.Due, "2026-05-01")
	}
}

func TestSortByDueWithNilValues(t *testing.T) {
	kanbanDir := initBoard(t)

	mustCreateTask(t, kanbanDir, "No due")
	mustCreateTask(t, kanbanDir, "Late due", "--due", "2026-12-31")
	mustCreateTask(t, kanbanDir, "Early due", "--due", "2026-01-01")

	var tasks []taskJSON
	runKanbanJSON(t, kanbanDir, &tasks, "list", "--sort", "due")

	// Early first, late second, nil last.
	wantIDs := []int{3, 2, 1}
	if len(tasks) != len(wantIDs) {
		t.Fatalf("got %d tasks, want %d", len(tasks), len(wantIDs))
	}
	for i, want := range wantIDs {
		if tasks[i].ID != want {
			t.Errorf("position %d: ID = %d, want %d", i, tasks[i].ID, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Output format tests
// ---------------------------------------------------------------------------

func TestTextOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Text output test")

	// Table list.
	r := runKanban(t, kanbanDir, "--table", "list")
	if r.exitCode != 0 {
		t.Fatalf("table list failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Text output test") {
		t.Errorf("table list missing task title in stdout")
	}

	// Table show.
	r = runKanban(t, kanbanDir, "--table", "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("table show failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Task #1") {
		t.Errorf("show missing 'Task #1' in stdout")
	}

	// Table create.
	r = runKanban(t, kanbanDir, "--table", "create", "Another task")
	if r.exitCode != 0 {
		t.Fatalf("table create failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Created task #2") {
		t.Errorf("create missing 'Created task #2' in stdout")
	}
}

// ---------------------------------------------------------------------------
// Blocked state tests
// ---------------------------------------------------------------------------

func TestBlockTask(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Blockable task")

	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--block", "waiting for API")

	// Verify via show.
	var shown struct {
		taskJSON
		Blocked     bool   `json:"blocked"`
		BlockReason string `json:"block_reason"`
	}
	runKanbanJSON(t, kanbanDir, &shown, "show", "1")
	if !shown.Blocked {
		t.Error("Blocked = false, want true")
	}
	if shown.BlockReason != "waiting for API" {
		t.Errorf("BlockReason = %q, want %q", shown.BlockReason, "waiting for API")
	}
}

func TestUnblockTask(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Unblockable")

	// Block first.
	runKanban(t, kanbanDir, "--json", "edit", "1", "--block", "stuck")

	// Unblock.
	runKanban(t, kanbanDir, "--json", "edit", "1", "--unblock")

	var shown struct {
		taskJSON
		Blocked     bool   `json:"blocked"`
		BlockReason string `json:"block_reason"`
	}
	runKanbanJSON(t, kanbanDir, &shown, "show", "1")
	if shown.Blocked {
		t.Error("Blocked = true after unblock, want false")
	}
	if shown.BlockReason != "" {
		t.Errorf("BlockReason = %q after unblock, want empty", shown.BlockReason)
	}
}

func TestBlockRequiresReason(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "No reason")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--block", "")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "block reason is required") {
		t.Errorf("error = %q, want 'block reason is required'", errResp.Error)
	}
}

func TestBlockAndUnblockConflict(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Conflict")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--block", "reason", "--unblock")
	if errResp.Code != "STATUS_CONFLICT" {
		t.Errorf("code = %q, want STATUS_CONFLICT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "cannot use --block and --unblock together") {
		t.Errorf("error = %q, want conflict message", errResp.Error)
	}
}

func TestListBlocked(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Normal task")
	mustCreateTask(t, kanbanDir, "Blocked task")
	runKanban(t, kanbanDir, "--json", "edit", "2", "--block", "stuck on dep")

	var tasks []struct {
		taskJSON
		Blocked bool `json:"blocked"`
	}
	runKanbanJSON(t, kanbanDir, &tasks, "list", "--blocked")
	if len(tasks) != 1 {
		t.Fatalf("got %d blocked tasks, want 1", len(tasks))
	}
	if tasks[0].ID != 2 {
		t.Errorf("blocked task ID = %d, want 2", tasks[0].ID)
	}
}

func TestListNotBlocked(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Normal task")
	mustCreateTask(t, kanbanDir, "Blocked task")
	runKanban(t, kanbanDir, "--json", "edit", "2", "--block", "stuck")

	var tasks []struct {
		taskJSON
		Blocked bool `json:"blocked"`
	}
	runKanbanJSON(t, kanbanDir, &tasks, "list", "--not-blocked")
	if len(tasks) != 1 {
		t.Fatalf("got %d not-blocked tasks, want 1", len(tasks))
	}
	if tasks[0].ID != 1 {
		t.Errorf("not-blocked task ID = %d, want 1", tasks[0].ID)
	}
}

func TestMoveBlockedTaskWarns(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Blocked mover")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--block", "waiting")

	r := runKanban(t, kanbanDir, "--json", "move", "1", "todo")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (move should succeed)", r.exitCode)
	}
	if !strings.Contains(r.stderr, "Warning") || !strings.Contains(r.stderr, "blocked") {
		t.Errorf("stderr = %q, want warning about blocked task", r.stderr)
	}
}

// ---------------------------------------------------------------------------
// Board summary tests
// ---------------------------------------------------------------------------

func TestBoardSummaryEmpty(t *testing.T) {
	kanbanDir := initBoard(t)

	var summary struct {
		BoardName  string `json:"board_name"`
		TotalTasks int    `json:"total_tasks"`
		Statuses   []struct {
			Status  string `json:"status"`
			Count   int    `json:"count"`
			Blocked int    `json:"blocked"`
			Overdue int    `json:"overdue"`
		} `json:"statuses"`
		Priorities []struct {
			Priority string `json:"priority"`
			Count    int    `json:"count"`
		} `json:"priorities"`
	}
	runKanbanJSON(t, kanbanDir, &summary, "board")

	if summary.TotalTasks != 0 {
		t.Errorf("TotalTasks = %d, want 0", summary.TotalTasks)
	}
	if len(summary.Statuses) != 5 {
		t.Errorf("Statuses count = %d, want 5", len(summary.Statuses))
	}
	for _, ss := range summary.Statuses {
		if ss.Count != 0 {
			t.Errorf("status %q count = %d, want 0", ss.Status, ss.Count)
		}
	}
}

func TestBoardSummaryWithTasks(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A", "--priority", "high")
	mustCreateTask(t, kanbanDir, "Task B", "--priority", "high")
	mustCreateTask(t, kanbanDir, "Task C", "--priority", "low")
	runKanban(t, kanbanDir, "--json", "move", "1", "todo")
	runKanban(t, kanbanDir, "--json", "move", "3", "done")

	var summary struct {
		TotalTasks int `json:"total_tasks"`
		Statuses   []struct {
			Status string `json:"status"`
			Count  int    `json:"count"`
		} `json:"statuses"`
		Priorities []struct {
			Priority string `json:"priority"`
			Count    int    `json:"count"`
		} `json:"priorities"`
	}
	runKanbanJSON(t, kanbanDir, &summary, "board")

	if summary.TotalTasks != 3 {
		t.Fatalf("TotalTasks = %d, want 3", summary.TotalTasks)
	}

	statusCounts := make(map[string]int)
	for _, ss := range summary.Statuses {
		statusCounts[ss.Status] = ss.Count
	}
	if statusCounts["backlog"] != 1 {
		t.Errorf("backlog = %d, want 1", statusCounts["backlog"])
	}
	if statusCounts["todo"] != 1 {
		t.Errorf("todo = %d, want 1", statusCounts["todo"])
	}
	if statusCounts["done"] != 1 {
		t.Errorf("done = %d, want 1", statusCounts["done"])
	}

	prioMap := make(map[string]int)
	for _, pc := range summary.Priorities {
		prioMap[pc.Priority] = pc.Count
	}
	if prioMap["high"] != 2 {
		t.Errorf("high = %d, want 2", prioMap["high"])
	}
	if prioMap["low"] != 1 {
		t.Errorf("low = %d, want 1", prioMap["low"])
	}
}

func TestBoardSummaryTableOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Table test")

	r := runKanban(t, kanbanDir, "--table", "board")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", r.exitCode)
	}
	if !strings.Contains(r.stdout, "STATUS") || !strings.Contains(r.stdout, "COUNT") {
		t.Errorf("table header not found in output:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "PRIORITY") {
		t.Errorf("priority section not found in output:\n%s", r.stdout)
	}
}

func TestBoardSummaryAlias(t *testing.T) {
	kanbanDir := initBoard(t)

	var summary struct {
		TotalTasks int `json:"total_tasks"`
	}
	runKanbanJSON(t, kanbanDir, &summary, "summary")

	if summary.TotalTasks != 0 {
		t.Errorf("TotalTasks via alias = %d, want 0", summary.TotalTasks)
	}
}

// ---------------------------------------------------------------------------
// Metrics tests
// ---------------------------------------------------------------------------

func TestMetricsEmpty(t *testing.T) {
	kanbanDir := initBoard(t)

	var m struct {
		Throughput7d  int `json:"throughput_7d"`
		Throughput30d int `json:"throughput_30d"`
	}
	runKanbanJSON(t, kanbanDir, &m, "metrics")

	if m.Throughput7d != 0 {
		t.Errorf("Throughput7d = %d, want 0", m.Throughput7d)
	}
	if m.Throughput30d != 0 {
		t.Errorf("Throughput30d = %d, want 0", m.Throughput30d)
	}
}

func TestMetricsWithCompletedTasks(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	// Move tasks through the workflow to get timestamps.
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")
	runKanban(t, kanbanDir, "--json", "move", "1", "done")
	runKanban(t, kanbanDir, "--json", "move", "2", "in-progress")

	var m struct {
		Throughput7d  int `json:"throughput_7d"`
		Throughput30d int `json:"throughput_30d"`
		AgingItems    []struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"aging_items"`
	}
	runKanbanJSON(t, kanbanDir, &m, "metrics")

	if m.Throughput7d != 1 {
		t.Errorf("Throughput7d = %d, want 1", m.Throughput7d)
	}
	if m.Throughput30d != 1 {
		t.Errorf("Throughput30d = %d, want 1", m.Throughput30d)
	}
	if len(m.AgingItems) != 1 {
		t.Fatalf("AgingItems = %d, want 1", len(m.AgingItems))
	}
	if m.AgingItems[0].ID != 2 {
		t.Errorf("AgingItems[0].ID = %d, want 2", m.AgingItems[0].ID)
	}
}

func TestMetricsSinceFilter(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")
	runKanban(t, kanbanDir, "--json", "move", "1", "done")

	// Filter with a future date — completed task should be excluded from throughput.
	var m struct {
		Throughput7d  int `json:"throughput_7d"`
		Throughput30d int `json:"throughput_30d"`
	}
	runKanbanJSON(t, kanbanDir, &m, "metrics", "--since", "2099-01-01")

	if m.Throughput7d != 0 {
		t.Errorf("Throughput7d = %d, want 0 (filtered out)", m.Throughput7d)
	}
}

func TestMetricsTableOutput(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--table", "metrics")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", r.exitCode)
	}
	if !strings.Contains(r.stdout, "Throughput") {
		t.Errorf("table output missing throughput:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "lead time") || !strings.Contains(r.stdout, "cycle time") {
		t.Errorf("table output missing time fields:\n%s", r.stdout)
	}
}

func TestMetricsInvalidSinceStructuredError(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "metrics", "--since", "not-a-date")
	if errResp.Code != codeInvalidDate {
		t.Errorf("code = %q, want INVALID_DATE", errResp.Code)
	}
}

func TestLogInvalidSinceStructuredError(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "log", "--since", "not-a-date")
	if errResp.Code != codeInvalidDate {
		t.Errorf("code = %q, want INVALID_DATE", errResp.Code)
	}
}

// ---------------------------------------------------------------------------
// Activity log tests
// ---------------------------------------------------------------------------

type logEntry struct {
	Action string `json:"action"`
	TaskID int    `json:"task_id"`
	Detail string `json:"detail"`
}

func TestLogEmptyBoard(t *testing.T) {
	kanbanDir := initBoard(t)

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log")

	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestLogAfterCreate(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Log me")

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log")

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Action != "create" {
		t.Errorf("action = %q, want %q", entries[0].Action, "create")
	}
	if entries[0].TaskID != 1 {
		t.Errorf("task_id = %d, want 1", entries[0].TaskID)
	}
}

func TestLogAfterMove(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Mover")
	runKanban(t, kanbanDir, "--json", "move", "1", "todo")

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log")

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[1].Action != "move" {
		t.Errorf("action = %q, want %q", entries[1].Action, "move")
	}
	if !strings.Contains(entries[1].Detail, "->") {
		t.Errorf("detail = %q, want to contain '->'", entries[1].Detail)
	}
}

func TestLogAfterEdit(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Editor")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--priority", "high")

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log")

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[1].Action != "edit" {
		t.Errorf("action = %q, want %q", entries[1].Action, "edit")
	}
}

func TestLogAfterDelete(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Deletable")
	runKanban(t, kanbanDir, "--json", "delete", "1", "--force")

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log")

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[1].Action != "delete" {
		t.Errorf("action = %q, want %q", entries[1].Action, "delete")
	}
}

func TestLogBlockUnblock(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Blocker")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--block", "waiting")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--unblock")

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log")

	// create, edit+block, edit+unblock
	actions := make(map[string]int)
	for _, e := range entries {
		actions[e.Action]++
	}
	if actions["block"] != 1 {
		t.Errorf("block actions = %d, want 1", actions["block"])
	}
	if actions["unblock"] != 1 {
		t.Errorf("unblock actions = %d, want 1", actions["unblock"])
	}
}

func TestLogActionFilter(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "move", "1", "todo")

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log", "--action", "move")

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Action != "move" {
		t.Errorf("action = %q, want %q", entries[0].Action, "move")
	}
}

func TestLogTaskFilter(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log", "--task", "2")

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].TaskID != 2 {
		t.Errorf("task_id = %d, want 2", entries[0].TaskID)
	}
}

func TestLogLimit(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")
	mustCreateTask(t, kanbanDir, "Task C")

	var entries []logEntry
	runKanbanJSON(t, kanbanDir, &entries, "log", "--limit", "2")

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
}

func TestLogTableOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Table entry")

	r := runKanban(t, kanbanDir, "--table", "log")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", r.exitCode)
	}
	if !strings.Contains(r.stdout, "TIMESTAMP") || !strings.Contains(r.stdout, "ACTION") {
		t.Errorf("table header not found in output:\n%s", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Slug / output tests
// ---------------------------------------------------------------------------

func TestLongTitleSlugTruncation(t *testing.T) {
	kanbanDir := initBoard(t)

	longTitle := "This is a very long title that should be truncated at a word boundary to fit within the slug limit"
	task := mustCreateTask(t, kanbanDir, longTitle)

	basename := filepath.Base(task.File)
	// Remove "001-" prefix and ".md" suffix to get slug.
	slug := strings.TrimSuffix(strings.TrimPrefix(basename, "001-"), ".md")
	if len(slug) > 50 {
		t.Errorf("slug length = %d, want <= 50: %q", len(slug), slug)
	}
}

// ---------------------------------------------------------------------------
// Config command tests
// ---------------------------------------------------------------------------

func TestConfigShowAll(t *testing.T) {
	kanbanDir := initBoard(t)

	var cfg map[string]any
	runKanbanJSON(t, kanbanDir, &cfg, "config")

	// Verify expected keys are present.
	expectedKeys := []string{
		"version", "board.name", "board.description", "tasks_dir",
		"statuses", "priorities", "defaults.status", "defaults.priority",
		"wip_limits", "next_id",
	}
	for _, key := range expectedKeys {
		if _, ok := cfg[key]; !ok {
			t.Errorf("missing key %q in config output", key)
		}
	}

	// board.name is derived from CWD during init; just verify it's non-empty.
	if cfg["board.name"] == "" {
		t.Error("board.name is empty")
	}
}

func TestConfigGetBoardName(t *testing.T) {
	kanbanDir := initBoard(t)

	var name string
	runKanbanJSON(t, kanbanDir, &name, "config", "get", "board.name")
	if name == "" {
		t.Error("board.name is empty")
	}
}

func TestConfigGetStatuses(t *testing.T) {
	kanbanDir := initBoard(t)

	var statuses []string
	runKanbanJSON(t, kanbanDir, &statuses, "config", "get", "statuses")
	if len(statuses) != 5 {
		t.Fatalf("statuses = %v, want 5 items", statuses)
	}
	if statuses[0] != statusBacklog {
		t.Errorf("statuses[0] = %q, want %q", statuses[0], statusBacklog)
	}
}

func TestConfigSetDefaultPriority(t *testing.T) {
	kanbanDir := initBoard(t)

	// Set default priority to high.
	runKanban(t, kanbanDir, "--json", "config", "set", "defaults.priority", "high")

	// Verify it persisted.
	var val string
	runKanbanJSON(t, kanbanDir, &val, "config", "get", "defaults.priority")
	if val != priorityHigh {
		t.Errorf("defaults.priority = %q, want %q", val, priorityHigh)
	}
}

func TestConfigSetBoardName(t *testing.T) {
	kanbanDir := initBoard(t)

	runKanban(t, kanbanDir, "--json", "config", "set", "board.name", "My New Board")

	var val string
	runKanbanJSON(t, kanbanDir, &val, "config", "get", "board.name")
	if val != "My New Board" {
		t.Errorf("board.name = %q, want %q", val, "My New Board")
	}
}

func TestConfigSetReadOnlyKey(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "config", "set", "next_id", "99")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "read-only") {
		t.Errorf("error = %q, want 'read-only'", errResp.Error)
	}
}

func TestConfigGetInvalidKey(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "config", "get", "nonexistent.key")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "unknown config key") {
		t.Errorf("error = %q, want 'unknown config key'", errResp.Error)
	}
}

func TestConfigSetInvalidDefaultStatus(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "config", "set", "defaults.status", "nonexistent")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "invalid default status") {
		t.Errorf("error = %q, want 'invalid default status'", errResp.Error)
	}
}

func TestConfigTableOutput(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--table", "config")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", r.exitCode)
	}
	if !strings.Contains(r.stdout, "board.name") {
		t.Errorf("table output missing board.name:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "defaults.status") {
		t.Errorf("table output missing defaults.status:\n%s", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Context command tests
// ---------------------------------------------------------------------------

func TestContextEmptyBoard(t *testing.T) {
	kanbanDir := initBoard(t)

	var ctx struct {
		BoardName string `json:"board_name"`
		Summary   struct {
			TotalTasks int `json:"total_tasks"`
		} `json:"summary"`
		Sections []struct {
			Name string `json:"name"`
		} `json:"sections"`
	}
	runKanbanJSON(t, kanbanDir, &ctx, "context")

	if ctx.Summary.TotalTasks != 0 {
		t.Errorf("TotalTasks = %d, want 0", ctx.Summary.TotalTasks)
	}
	if len(ctx.Sections) != 0 {
		t.Errorf("Sections = %d, want 0", len(ctx.Sections))
	}
}

func TestContextWithTasks(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Active task")
	mustCreateTask(t, kanbanDir, "Blocked task")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")
	runKanban(t, kanbanDir, "--json", "edit", "2", "--block", "waiting")

	var ctx struct {
		Summary struct {
			TotalTasks int `json:"total_tasks"`
			Active     int `json:"active"`
			Blocked    int `json:"blocked"`
		} `json:"summary"`
		Sections []struct {
			Name  string `json:"name"`
			Items []struct {
				ID int `json:"id"`
			} `json:"items"`
		} `json:"sections"`
	}
	runKanbanJSON(t, kanbanDir, &ctx, "context")

	if ctx.Summary.TotalTasks != 2 {
		t.Errorf("TotalTasks = %d, want 2", ctx.Summary.TotalTasks)
	}
	if ctx.Summary.Active < 1 {
		t.Errorf("Active = %d, want >= 1", ctx.Summary.Active)
	}
	if ctx.Summary.Blocked != 1 {
		t.Errorf("Blocked = %d, want 1", ctx.Summary.Blocked)
	}

	// Should have at least in-progress and blocked sections.
	sectionNames := make(map[string]bool)
	for _, s := range ctx.Sections {
		sectionNames[s.Name] = true
	}
	if !sectionNames["in-progress"] {
		t.Error("missing in-progress section")
	}
	if !sectionNames["blocked"] {
		t.Error("missing blocked section")
	}
}

func TestContextJSON(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "JSON task")

	var ctx struct {
		BoardName string `json:"board_name"`
		Summary   struct {
			TotalTasks int `json:"total_tasks"`
		} `json:"summary"`
	}
	runKanbanJSON(t, kanbanDir, &ctx, "context")

	if ctx.BoardName == "" {
		t.Error("BoardName is empty")
	}
	if ctx.Summary.TotalTasks != 1 {
		t.Errorf("TotalTasks = %d, want 1", ctx.Summary.TotalTasks)
	}
}

func TestContextWriteToFile_NewFile(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Written task")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")

	outFile := filepath.Join(t.TempDir(), "context.md")
	runKanban(t, kanbanDir, "context", "--write-to", outFile)

	data, err := os.ReadFile(outFile) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<!-- BEGIN kanban-md context -->") {
		t.Error("missing begin marker")
	}
	if !strings.Contains(content, "<!-- END kanban-md context -->") {
		t.Error("missing end marker")
	}
	if !strings.Contains(content, "Written task") {
		t.Error("missing task in context")
	}
}

func TestContextWriteToFile_UpdateExisting(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "First run")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")

	outFile := filepath.Join(t.TempDir(), "AGENTS.md")

	// Write initial file with other content.
	initial := "# My Agents\n\nImportant notes here.\n"
	if err := os.WriteFile(outFile, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	// First write — appends context block.
	runKanban(t, kanbanDir, "context", "--write-to", outFile)

	// Create another task and move it, then write again — should update in place.
	mustCreateTask(t, kanbanDir, "Second run")
	runKanban(t, kanbanDir, "--json", "move", "2", "in-progress")
	runKanban(t, kanbanDir, "context", "--write-to", outFile)

	data, err := os.ReadFile(outFile) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "# My Agents") {
		t.Error("lost header content")
	}
	if !strings.Contains(content, "Important notes here") {
		t.Error("lost body content")
	}
	if !strings.Contains(content, "Second run") {
		t.Error("missing updated task")
	}

	// Should only have one begin/end marker pair.
	if strings.Count(content, "<!-- BEGIN kanban-md context -->") != 1 {
		t.Error("multiple begin markers found — in-place update failed")
	}
}

func TestContextWriteToFile_AppendToExisting(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Appended task")

	outFile := filepath.Join(t.TempDir(), "README.md")
	existing := "# README\n\nSome documentation.\n"
	if err := os.WriteFile(outFile, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	runKanban(t, kanbanDir, "context", "--write-to", outFile)

	data, err := os.ReadFile(outFile) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "# README") {
		t.Error("lost existing content")
	}
	if !strings.Contains(content, "<!-- BEGIN kanban-md context -->") {
		t.Error("context not appended")
	}
}

func TestContextSectionsFilter(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Filtered task")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--block", "test")

	var ctx struct {
		Sections []struct {
			Name string `json:"name"`
		} `json:"sections"`
	}
	runKanbanJSON(t, kanbanDir, &ctx, "context", "--sections", "blocked")

	if len(ctx.Sections) != 1 {
		t.Fatalf("Sections = %d, want 1", len(ctx.Sections))
	}
	if ctx.Sections[0].Name != "blocked" {
		t.Errorf("Section = %q, want %q", ctx.Sections[0].Name, "blocked")
	}
}

// ---------------------------------------------------------------------------
// Batch operations tests
// ---------------------------------------------------------------------------

type batchResultJSON struct {
	ID    int    `json:"id"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Code  string `json:"code,omitempty"`
}

func TestBatchMoveAll(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")
	mustCreateTask(t, kanbanDir, "Task C")

	var results []batchResultJSON
	runKanbanJSON(t, kanbanDir, &results, "move", "1,2,3", "todo")

	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	for _, r := range results {
		if !r.OK {
			t.Errorf("task #%d failed: %s", r.ID, r.Error)
		}
	}

	// Verify tasks actually moved.
	var tasks []taskJSON
	runKanbanJSON(t, kanbanDir, &tasks, "list", "--status", "todo")
	if len(tasks) != 3 {
		t.Errorf("todo tasks = %d, want 3", len(tasks))
	}
}

func TestBatchMovePartialFailure(t *testing.T) {
	kanbanDir := initBoardWithWIP(t, 1)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")
	mustCreateTask(t, kanbanDir, "Task C")

	// Move all 3 to in-progress with WIP limit of 1. First succeeds, rest fail.
	r := runKanban(t, kanbanDir, "--json", "move", "1,2,3", "in-progress")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit code for partial failure")
	}

	var results []batchResultJSON
	if err := json.Unmarshal([]byte(r.stdout), &results); err != nil {
		t.Fatalf("parsing batch results: %v\nstdout: %s", err, r.stdout)
	}

	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	if !results[0].OK {
		t.Error("first task should succeed")
	}
	if results[1].OK || results[2].OK {
		t.Error("remaining tasks should fail due to WIP limit")
	}
}

func TestBatchEditMultiple(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")
	mustCreateTask(t, kanbanDir, "Task C")

	var results []batchResultJSON
	runKanbanJSON(t, kanbanDir, &results, "edit", "1,2,3", "--priority", priorityHigh)

	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	for _, r := range results {
		if !r.OK {
			t.Errorf("task #%d failed: %s", r.ID, r.Error)
		}
	}

	// Verify tasks actually updated.
	var tasks []taskJSON
	runKanbanJSON(t, kanbanDir, &tasks, "list", "--priority", priorityHigh)
	if len(tasks) != 3 {
		t.Errorf("high priority tasks = %d, want 3", len(tasks))
	}
}

func TestBatchDeleteMultiple(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")
	mustCreateTask(t, kanbanDir, "Task C")

	var results []batchResultJSON
	runKanbanJSON(t, kanbanDir, &results, "delete", "1,2,3", "--force")

	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	for _, r := range results {
		if !r.OK {
			t.Errorf("task #%d failed: %s", r.ID, r.Error)
		}
	}

	// Verify tasks are gone.
	var tasks []taskJSON
	runKanbanJSON(t, kanbanDir, &tasks, "list")
	if len(tasks) != 0 {
		t.Errorf("remaining tasks = %d, want 0", len(tasks))
	}
}

func TestBatchExitCode(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	// Move existing + nonexistent task. Partial failure → exit code 1.
	r := runKanban(t, kanbanDir, "--json", "move", "1,99", "todo")
	if r.exitCode != 1 {
		t.Errorf("exit code = %d, want 1", r.exitCode)
	}
}

func TestBatchSingleIDBackcompat(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Solo task")

	// Single ID should return a task object (not batch results array).
	var result struct {
		ID      int    `json:"id"`
		Status  string `json:"status"`
		Changed bool   `json:"changed"`
	}
	runKanbanJSON(t, kanbanDir, &result, "move", "1", "todo")

	if result.ID != 1 {
		t.Errorf("ID = %d, want 1", result.ID)
	}
	if result.Status != "todo" {
		t.Errorf("Status = %q, want %q", result.Status, "todo")
	}
	if !result.Changed {
		t.Error("Changed = false, want true")
	}
}

func TestBatchDeleteWarnsDependents(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")
	mustCreateTask(t, kanbanDir, "Task C")

	// Make B depend on A.
	runKanban(t, kanbanDir, "edit", "2", "--add-dep", "1")

	// Batch delete A and C — should warn about B depending on A.
	r := runKanban(t, kanbanDir, "delete", "1,3", "--force")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stderr, "Warning") {
		t.Errorf("stderr should contain dependent-task warning, got: %q", r.stderr)
	}
}

func TestBatchDeleteRequiresForce(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	errResp := runKanbanJSONError(t, kanbanDir, "delete", "1,2")
	if errResp.Code != "CONFIRMATION_REQUIRED" {
		t.Errorf("code = %q, want CONFIRMATION_REQUIRED", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "batch delete requires --force") {
		t.Errorf("error = %q, want 'batch delete requires --force'", errResp.Error)
	}
}

func TestBatchMoveJSON(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	var results []batchResultJSON
	runKanbanJSON(t, kanbanDir, &results, "move", "1,2", "todo")

	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[0].ID != 1 || results[1].ID != 2 {
		t.Errorf("IDs = [%d,%d], want [1,2]", results[0].ID, results[1].ID)
	}
}

// ---------------------------------------------------------------------------
// Default output format tests (table is always the default)
// ---------------------------------------------------------------------------

func TestDefaultOutputIsTable(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Default output task", "--tags", "test,demo")

	// Default output should be table (even when piped/non-TTY).
	r := runKanban(t, kanbanDir, "list")
	if r.exitCode != 0 {
		t.Fatalf("list failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "ID") || !strings.Contains(r.stdout, "STATUS") {
		t.Errorf("default list should be table with headers, got:\n%s", r.stdout)
	}

	// Create should also default to table (message) output.
	r = runKanban(t, kanbanDir, "create", "Another task")
	if r.exitCode != 0 {
		t.Fatalf("create failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Created task #2") {
		t.Errorf("default create should show message, got:\n%s", r.stdout)
	}
}

func TestTableFlagOutputList(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Table output task", "--tags", "test,demo")

	r := runKanban(t, kanbanDir, "--table", "list")
	if r.exitCode != 0 {
		t.Fatalf("list failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "ID") || !strings.Contains(r.stdout, "STATUS") {
		t.Errorf("table output missing headers:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "TAGS") {
		t.Errorf("table output missing TAGS column:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "test,demo") {
		t.Errorf("table output missing tag values:\n%s", r.stdout)
	}
}

func TestTableFlagOutputCreate(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--table", "create", "New task via table")
	if r.exitCode != 0 {
		t.Fatalf("create failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Created task #1") {
		t.Errorf("table create missing confirmation:\n%s", r.stdout)
	}
}

func TestTableFlagOutputMove(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Movable task")

	r := runKanban(t, kanbanDir, "--table", "move", "1", "todo")
	if r.exitCode != 0 {
		t.Fatalf("move failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Moved task #1") {
		t.Errorf("table move missing confirmation:\n%s", r.stdout)
	}
}

func TestTableFlagOutputDelete(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Deletable task")

	r := runKanban(t, kanbanDir, "--table", "delete", "1", "--force")
	if r.exitCode != 0 {
		t.Fatalf("delete failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Deleted task #1") {
		t.Errorf("table delete missing confirmation:\n%s", r.stdout)
	}
}

func TestTableFlagOutputBoard(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Board task")

	r := runKanban(t, kanbanDir, "--table", "board")
	if r.exitCode != 0 {
		t.Fatalf("board failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "STATUS") || !strings.Contains(r.stdout, "COUNT") {
		t.Errorf("table board missing headers:\n%s", r.stdout)
	}
}

func TestVersionDefault(t *testing.T) {
	// Binary built without ldflags should report "dev".
	r := runKanban(t, t.TempDir(), "--version")
	if r.exitCode != 0 {
		t.Fatalf("--version failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "dev") {
		t.Errorf("default version should contain 'dev', got: %s", r.stdout)
	}
}

func TestVersionLdflags(t *testing.T) {
	// Build a separate binary with ldflags to verify version injection works.
	tmp := t.TempDir()
	binName := "kanban-md-version-test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	versionBin := filepath.Join(tmp, binName)
	wantVersion := "1.2.3-test"

	//nolint:gosec,noctx // building test binary with ldflags
	build := exec.Command("go", "build",
		"-ldflags", "-X github.com/antopolskiy/kanban-md/cmd.version="+wantVersion,
		"-o", versionBin, "../cmd/kanban-md")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("building binary with ldflags: %v", err)
	}

	//nolint:gosec,noctx // running test binary
	cmd := exec.Command(versionBin, "--version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running --version: %v", err)
	}

	if !strings.Contains(string(out), wantVersion) {
		t.Errorf("version should contain %q, got: %s", wantVersion, string(out))
	}
}

func TestTableFlagOutputMetrics(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--table", "metrics")
	if r.exitCode != 0 {
		t.Fatalf("metrics failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Throughput") {
		t.Errorf("table metrics missing throughput:\n%s", r.stdout)
	}
}

func TestREADMEDocumentsAllCommands(t *testing.T) {
	readmePath := filepath.Join("..", "README.md")
	data, err := os.ReadFile(readmePath) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("reading README: %v", err)
	}
	readme := string(data)

	// Every user-facing command must have a ### `command` section.
	commands := []string{
		"init", "create", "list", "show", "edit", "move", "delete",
		"board", "pick", "metrics", "log", "config", "context",
	}
	for _, cmd := range commands {
		heading := "### `" + cmd + "`"
		if !strings.Contains(readme, heading) {
			t.Errorf("README missing command section: %s", heading)
		}
	}

	// Key flags that must be documented somewhere in the README.
	requiredFlags := map[string][]string{
		"init":   {"--wip-limit"},
		"create": {"--parent", "--depends-on"},
		"edit": {
			"--started", "--clear-started", "--completed", "--clear-completed",
			"--parent", "--clear-parent", "--add-dep", "--remove-dep",
			"--block", "--unblock", "--claim", "--release", "--class", "--force",
		},
		"move":    {"--claim"},
		"list":    {"--blocked", "--not-blocked", "--parent", "--unblocked", "--unclaimed", "--claimed-by", "--class", "--group-by"},
		"board":   {"--group-by"},
		"pick":    {"--claim", "--status", "--move", "--tag"},
		"metrics": {"--since"},
		"log":     {"--since", "--limit", "--action", "--task"},
	}
	for cmd, flags := range requiredFlags {
		for _, flag := range flags {
			// Flag should appear in the README (in the command's section or flags table).
			if !strings.Contains(readme, "`"+flag+"`") {
				t.Errorf("README missing flag %s for command %s", flag, cmd)
			}
		}
	}

	// Config example must show current schema version.
	if !strings.Contains(readme, "version: 3") {
		t.Error("README config example still shows old version (should be version: 3)")
	}

	// Config example must mention wip_limits.
	if !strings.Contains(readme, "wip_limits") {
		t.Error("README config example missing wip_limits field")
	}
}

// ---------------------------------------------------------------------------
// Compact output format tests
// ---------------------------------------------------------------------------

func TestCompactOutputList(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Compact list task", "--tags", "test,demo", "--priority", "high")

	r := runKanban(t, kanbanDir, "--compact", "list")
	if r.exitCode != 0 {
		t.Fatalf("compact list failed: %s", r.stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(r.stdout), "#1 [") {
		t.Errorf("compact list should start with '#1 [', got:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Compact list task") {
		t.Errorf("compact list missing title:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "(test, demo)") {
		t.Errorf("compact list missing tags:\n%s", r.stdout)
	}
}

func TestCompactOutputShow(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Compact show task", "--priority", "high")

	r := runKanban(t, kanbanDir, "--compact", "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("compact show failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "#1 [backlog/high] Compact show task") {
		t.Errorf("compact show missing header:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "created:") {
		t.Errorf("compact show missing timestamps:\n%s", r.stdout)
	}
}

func TestCompactOutputBoard(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Board compact task")

	r := runKanban(t, kanbanDir, "--compact", "board")
	if r.exitCode != 0 {
		t.Fatalf("compact board failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "tasks)") {
		t.Errorf("compact board missing task count:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "backlog:") {
		t.Errorf("compact board missing status line:\n%s", r.stdout)
	}
}

func TestCompactOutputMetrics(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--compact", "metrics")
	if r.exitCode != 0 {
		t.Fatalf("compact metrics failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Throughput:") {
		t.Errorf("compact metrics missing throughput:\n%s", r.stdout)
	}
}

func TestCompactOutputLog(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Log compact task")

	r := runKanban(t, kanbanDir, "--compact", "log")
	if r.exitCode != 0 {
		t.Fatalf("compact log failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "create #1") {
		t.Errorf("compact log missing create entry:\n%s", r.stdout)
	}
}

func TestOnelineAlias(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Oneline alias task")

	compact := runKanban(t, kanbanDir, "--compact", "list")
	oneline := runKanban(t, kanbanDir, "--oneline", "list")

	if compact.stdout != oneline.stdout {
		t.Errorf("--oneline should produce same output as --compact\ncompact:\n%s\noneline:\n%s",
			compact.stdout, oneline.stdout)
	}
}

func TestCompactEnvVar(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Env compact task")

	r := runKanbanEnv(t, kanbanDir, []string{"KANBAN_OUTPUT=compact"}, "list")
	if r.exitCode != 0 {
		t.Fatalf("env compact list failed: %s", r.stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(r.stdout), "#1 [") {
		t.Errorf("KANBAN_OUTPUT=compact should produce compact output, got:\n%s", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Skill command tests
// ---------------------------------------------------------------------------

// runKanbanNoDir runs the binary without the --dir flag (for skill commands).
func runKanbanNoDir(t *testing.T, dir string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...) //nolint:gosec,noctx // e2e test binary
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	r := result{stdout: stdout.String(), stderr: stderr.String()}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			r.exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running kanban-md: %v", err)
		}
	}
	return r
}

func TestSkillShow(t *testing.T) {
	dir := t.TempDir()
	r := runKanbanNoDir(t, dir, "skill", "show", "--skill", "kanban-md")
	if r.exitCode != 0 {
		t.Fatalf("skill show failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "kanban-md") {
		t.Errorf("skill show output should contain skill content, got:\n%s", r.stdout)
	}
}

func TestSkillShowAll(t *testing.T) {
	dir := t.TempDir()
	r := runKanbanNoDir(t, dir, "skill", "show")
	if r.exitCode != 0 {
		t.Fatalf("skill show all failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "=== kanban-md ===") {
		t.Error("skill show should include kanban-md header")
	}
	if !strings.Contains(r.stdout, "=== kanban-based-development ===") {
		t.Error("skill show should include kanban-based-development header")
	}
}

func TestSkillInstallAndCheck(t *testing.T) {
	dir := t.TempDir()
	// Create a .claude directory so the agent is detected.
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Install skills for claude agent.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("skill install failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "SKILL.md") {
		t.Errorf("expected file paths in output, got:\n%s", r.stdout)
	}

	// Check that SKILL.md files were created.
	skillMD := filepath.Join(dir, ".claude", "skills", "kanban-md", "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Errorf("kanban-md SKILL.md not created: %v", err)
	}
	devSkillMD := filepath.Join(dir, ".claude", "skills", "kanban-based-development", "SKILL.md")
	if _, err := os.Stat(devSkillMD); err != nil {
		t.Errorf("kanban-based-development SKILL.md not created: %v", err)
	}

	// Check that references subdir was created.
	refPath := filepath.Join(dir, ".claude", "skills", "kanban-md", "references", "json-schemas.md")
	if _, err := os.Stat(refPath); err != nil {
		t.Errorf("references/json-schemas.md not created: %v", err)
	}

	// Run check — should be up to date.
	r = runKanbanNoDir(t, dir, "skill", "check", "--agent", "claude")
	if r.exitCode != 0 {
		t.Errorf("skill check should pass (exit 0), got exit %d: %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "up to date") {
		t.Errorf("expected 'up to date' in check output, got:\n%s", r.stdout)
	}
}

func TestSkillInstallIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// First install.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("first install failed: %s", r.stderr)
	}

	// Second install — should skip (already installed).
	r = runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("second install failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "skipped") || !strings.Contains(r.stdout, "up to date") {
		// Either "skipped" or "up to date" is acceptable.
		if !strings.Contains(r.stdout, "skipped") && !strings.Contains(r.stdout, "up to date") {
			t.Errorf("expected skip message on re-install, got:\n%s", r.stdout)
		}
	}
}

func TestSkillInstallForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// First install.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("first install failed: %s", r.stderr)
	}

	// Force re-install — should overwrite.
	r = runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude", "--force")
	if r.exitCode != 0 {
		t.Fatalf("force install failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "SKILL.md") {
		t.Errorf("force install should re-install, got:\n%s", r.stdout)
	}
}

func TestSkillInstallPath(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "custom-skills")

	// Install to a custom path — directory doesn't exist yet, should be created.
	r := runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir)
	if r.exitCode != 0 {
		t.Fatalf("skill install --path failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "SKILL.md") {
		t.Errorf("expected file paths in output, got:\n%s", r.stdout)
	}

	// Verify files were created at the custom path.
	skillMD := filepath.Join(targetDir, "kanban-md", "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Errorf("SKILL.md not created at custom path: %v", err)
	}
	refPath := filepath.Join(targetDir, "kanban-md", "references", "json-schemas.md")
	if _, err := os.Stat(refPath); err != nil {
		t.Errorf("references not created at custom path: %v", err)
	}
}

func TestSkillInstallPathSelectSkill(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "skills")

	// Install only kanban-md skill to custom path.
	r := runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir, "--skill", "kanban-md")
	if r.exitCode != 0 {
		t.Fatalf("skill install --path --skill failed: %s", r.stderr)
	}

	// kanban-md should exist, kanban-based-development should not.
	if _, err := os.Stat(filepath.Join(targetDir, "kanban-md", "SKILL.md")); err != nil {
		t.Error("kanban-md SKILL.md not created")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "kanban-based-development", "SKILL.md")); err == nil {
		t.Error("kanban-based-development should not be installed with --skill kanban-md")
	}
}

func TestSkillInstallCreatesNestedDirs(t *testing.T) {
	dir := t.TempDir()

	// Create a .cursor directory (parent exists, but skills/ subdir doesn't).
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o750); err != nil {
		t.Fatal(err)
	}

	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "cursor")
	if r.exitCode != 0 {
		t.Fatalf("install for cursor failed: %s", r.stderr)
	}

	// Verify the nested skills directory was created.
	skillMD := filepath.Join(dir, ".cursor", "skills", "kanban-md", "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Errorf("cursor skill not installed: %v", err)
	}
}

func TestSkillInstallExistingDirPreserved(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Create an existing file in the skills dir that we don't manage.
	userFile := filepath.Join(dir, ".claude", "skills", "my-custom-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("custom content"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Install kanban-md skills.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("install failed: %s", r.stderr)
	}

	// The user's custom skill should not be touched.
	data, err := os.ReadFile(userFile) //nolint:gosec // test reads from known temp directory
	if err != nil {
		t.Fatalf("user's custom skill deleted: %v", err)
	}
	if string(data) != "custom content" {
		t.Errorf("user's custom skill modified: %q", string(data))
	}
}

func TestSkillInstallSymlinkNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "skills"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Create a symlink where kanban-md/ would go, pointing to an external dir.
	externalDir := filepath.Join(dir, "external-data")
	if err := os.MkdirAll(externalDir, 0o750); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(externalDir, "important.txt")
	if err := os.WriteFile(sentinel, []byte("do not delete"), 0o600); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(dir, ".claude", "skills", "kanban-md")
	if err := os.Symlink(externalDir, symlinkPath); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	// Install should still succeed (writes through the symlink), but must NOT
	// delete the symlink target or the sentinel file.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude", "--force")
	if r.exitCode != 0 {
		t.Fatalf("install with symlink failed: %s", r.stderr)
	}

	// The sentinel file in the external dir must survive.
	data, err := os.ReadFile(sentinel) //nolint:gosec // test reads from known temp directory
	if err != nil {
		t.Fatalf("sentinel file in symlink target deleted: %v", err)
	}
	if string(data) != "do not delete" {
		t.Errorf("sentinel file corrupted: %q", string(data))
	}

	// The symlink itself should still be a symlink.
	fi, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("symlink gone: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("kanban-md is no longer a symlink after install")
	}
}

func TestSkillInstallOpenClawGlobalOnly(t *testing.T) {
	dir := t.TempDir()

	// Install for openclaw — should go to global dir, not project dir.
	// Use --path equivalent by checking that no "skills/" directory is created
	// in the project root.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "openclaw")
	if r.exitCode != 0 {
		t.Fatalf("openclaw install failed: %s", r.stderr)
	}

	// No skills/ directory should appear in the project root.
	if _, err := os.Stat(filepath.Join(dir, "skills")); err == nil {
		t.Error("openclaw install created skills/ in project root instead of global path")
	}

	// Output should reference .openclaw path.
	if !strings.Contains(r.stdout, ".openclaw") {
		t.Errorf("expected global .openclaw path in output, got:\n%s", r.stdout)
	}
}

func TestSkillInstallMultipleAgents(t *testing.T) {
	dir := t.TempDir()
	// Set up directories for two agents.
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o750); err != nil {
		t.Fatal(err)
	}

	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude,cursor")
	if r.exitCode != 0 {
		t.Fatalf("multi-agent install failed: %s", r.stderr)
	}

	// Both agents should have skills installed.
	for _, agentDir := range []string{".claude", ".cursor"} {
		skillMD := filepath.Join(dir, agentDir, "skills", "kanban-md", "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			t.Errorf("%s skill not installed: %v", agentDir, err)
		}
	}
}

func TestSkillInstallVersionCommentPresent(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "skills")

	r := runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir, "--skill", "kanban-md")
	if r.exitCode != 0 {
		t.Fatalf("install failed: %s", r.stderr)
	}

	data, err := os.ReadFile(filepath.Join(targetDir, "kanban-md", "SKILL.md")) //nolint:gosec // test temp dir
	if err != nil {
		t.Fatalf("reading installed skill: %v", err)
	}

	if !strings.Contains(string(data), "<!-- kanban-md-skill-version:") {
		t.Error("installed SKILL.md missing version comment")
	}
}

func TestSkillUpdateAndCheck(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "skills")

	// Install with --path.
	r := runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir)
	if r.exitCode != 0 {
		t.Fatalf("install failed: %s", r.stderr)
	}

	// Tamper with version to simulate outdated.
	skillMD := filepath.Join(targetDir, "kanban-md", "SKILL.md")
	oldContent := "---\nname: kanban-md\n---\n<!-- kanban-md-skill-version: 0.0.1 -->\n# old content\n"
	if err := os.WriteFile(skillMD, []byte(oldContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Re-install — kanban-md should be updated (0.0.1 → dev),
	// kanban-based-development may be skipped (already at dev).
	r = runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir, "--skill", "kanban-md")
	if r.exitCode != 0 {
		t.Fatalf("re-install failed: %s", r.stderr)
	}
	// Should have updated since version differs.
	if strings.Contains(r.stdout, "skipped") {
		t.Errorf("expected install to update outdated skill, but it was skipped:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Installed 1 skill") {
		t.Errorf("expected 1 skill installed, got:\n%s", r.stdout)
	}
}

func TestInitShowsSkillHint(t *testing.T) {
	kanbanDir := initBoard(t)
	_ = kanbanDir
	// Re-run init to capture output (initBoard doesn't return stdout).
	dir := t.TempDir()
	kanbanDir2 := filepath.Join(dir, "kanban")
	r := runKanban(t, kanbanDir2, "init")
	if r.exitCode != 0 {
		t.Fatalf("init failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "skill install") {
		t.Errorf("init output should hint about skill install, got:\n%s", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Claim timeout / enforcement e2e tests
// ---------------------------------------------------------------------------

// writeTaskFile writes a raw task markdown file into the tasks directory.
// The filename follows the convention: 001-<slug>.md (zero-padded ID + slug).
// The title is extracted from the YAML frontmatter.
func writeTaskFile(t *testing.T, kanbanDir string, id int, content string) {
	t.Helper()
	// Extract title from frontmatter for the slug.
	slug := "task"
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "title:") {
			slug = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			slug = strings.ToLower(slug)
			slug = strings.ReplaceAll(slug, " ", "-")
			break
		}
	}
	filename := fmt.Sprintf("%03d-%s.md", id, slug)
	taskPath := filepath.Join(kanbanDir, "tasks", filename)
	if err := os.WriteFile(taskPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing task file %d: %v", id, err)
	}
}

// setConfigClaimTimeout rewrites the claim_timeout in config.yml.
func setConfigClaimTimeout(t *testing.T, kanbanDir, timeout string) {
	t.Helper()
	cfgPath := filepath.Join(kanbanDir, "config.yml")
	data, err := os.ReadFile(cfgPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)
	// Replace existing claim_timeout line or append if missing.
	if strings.Contains(content, "claim_timeout:") {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "claim_timeout:") {
				lines[i] = "claim_timeout: " + timeout
			}
		}
		content = strings.Join(lines, "\n")
	} else {
		content += "claim_timeout: " + timeout + "\n"
	}
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
}

// bumpNextID updates next_id in config.yml so the CLI knows the next available ID.
func bumpNextID(t *testing.T, kanbanDir string, nextID int) {
	t.Helper()
	cfgPath := filepath.Join(kanbanDir, "config.yml")
	data, err := os.ReadFile(cfgPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "next_id:") {
			lines[i] = "next_id: " + strconv.Itoa(nextID)
		}
	}
	content = strings.Join(lines, "\n")
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
}

func TestClaimBlocksOtherAgentMove(t *testing.T) {
	kanbanDir := initBoard(t)

	// Create a task and claim it by writing the file directly with a recent timestamp.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Another agent (no --force) tries to move — should fail.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress")
	if r.exitCode == 0 {
		t.Fatal("move should fail when task is claimed by another agent")
	}
	if !strings.Contains(r.stderr, "claimed") && !strings.Contains(r.stdout, "claimed") {
		t.Errorf("error should mention claim, got stdout=%q stderr=%q", r.stdout, r.stderr)
	}
}

func TestClaimBlocksOtherAgentEdit(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Edit without --force or matching --claim should fail.
	r := runKanban(t, kanbanDir, "edit", "1", "--priority", "low")
	if r.exitCode == 0 {
		t.Fatal("edit should fail when task is claimed by another agent")
	}
}

func TestClaimBlocksOtherAgentDelete(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Delete without --force should fail.
	r := runKanban(t, kanbanDir, "delete", "1")
	if r.exitCode == 0 {
		t.Fatal("delete should fail when task is claimed by another agent")
	}
}

func TestExpiredClaimAllowsMove(t *testing.T) {
	kanbanDir := initBoard(t)
	// Default claim_timeout is 1h. Set claimed_at to 2 hours in the past.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Move should succeed — claim has expired.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress")
	if r.exitCode != 0 {
		t.Fatalf("move should succeed for expired claim, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestExpiredClaimAllowsEdit(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Edit should succeed — claim has expired.
	r := runKanban(t, kanbanDir, "edit", "1", "--priority", "low")
	if r.exitCode != 0 {
		t.Fatalf("edit should succeed for expired claim, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestExpiredClaimAllowsDelete(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Delete requires --force in non-TTY for confirmation bypass.
	r := runKanban(t, kanbanDir, "delete", "1", "--force")
	if r.exitCode != 0 {
		t.Fatalf("delete should succeed for expired claim, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestActiveClaimNotExpiredBeforeTimeout(t *testing.T) {
	kanbanDir := initBoard(t)
	// Use a long timeout (10h) and a claim from 1 minute ago.
	setConfigClaimTimeout(t, kanbanDir, "10h")

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Recently claimed
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-busy
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Should still be blocked.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress")
	if r.exitCode == 0 {
		t.Fatal("move should fail — claim has not expired yet")
	}
}

func TestForceOverridesActiveClaim(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Force override test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Move with --force should succeed despite active claim.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress", "--force")
	if r.exitCode != 0 {
		t.Fatalf("move --force should succeed, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestForceOverridesActiveClaimEdit(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Force edit test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Edit with --force should succeed.
	r := runKanban(t, kanbanDir, "edit", "1", "--priority", "low", "--force")
	if r.exitCode != 0 {
		t.Fatalf("edit --force should succeed, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestForceOverridesActiveClaimDelete(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Force delete test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Delete with --force should succeed.
	r := runKanban(t, kanbanDir, "delete", "1", "--force")
	if r.exitCode != 0 {
		t.Fatalf("delete --force should succeed, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestSameAgentCanMoveClaimed(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Same agent test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Same agent can move with --claim flag matching the current claimant.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", "agent-alpha")
	if r.exitCode != 0 {
		t.Fatalf("same agent should be able to move, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestSameAgentCanEditClaimed(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Same agent edit test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Same agent (via --claim) can edit.
	r := runKanban(t, kanbanDir, "edit", "1", "--priority", "low", "--claim", "agent-alpha")
	if r.exitCode != 0 {
		t.Fatalf("same agent should be able to edit, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestClaimBlocksJSONError(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: JSON error test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Verify structured JSON error output.
	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "in-progress")
	if errResp.Code != "TASK_CLAIMED" {
		t.Errorf("error code = %q, want TASK_CLAIMED", errResp.Code)
	}
	if errResp.Details["claimed_by"] != "agent-alpha" {
		t.Errorf("details.claimed_by = %v, want %q", errResp.Details["claimed_by"], "agent-alpha")
	}
}

func TestPickSkipsClaimedTasks(t *testing.T) {
	kanbanDir := initBoard(t)

	// Task 1: claimed (should be skipped).
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: critical
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-other
claimed_at: 2099-01-01T00:00:00Z
---
`)
	// Task 2: unclaimed (should be picked).
	writeTaskFile(t, kanbanDir, 2, `---
id: 2
title: Available task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 3)

	r := runKanban(t, kanbanDir, "pick", "--claim", "agent-me", "--json")
	if r.exitCode != 0 {
		t.Fatalf("pick failed (exit %d): %s", r.exitCode, r.stderr)
	}

	var picked taskJSON
	if err := json.Unmarshal([]byte(r.stdout), &picked); err != nil {
		t.Fatalf("parsing pick output: %v\nstdout: %s", err, r.stdout)
	}

	if picked.ID != 2 {
		t.Errorf("pick selected task #%d, want #2 (should skip claimed #1)", picked.ID)
	}
}

func TestPickSelectsExpiredClaimTask(t *testing.T) {
	kanbanDir := initBoard(t)

	// Task 1: expired claim (should be available for pick).
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim
status: todo
priority: critical
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	r := runKanban(t, kanbanDir, "pick", "--claim", "agent-me", "--json")
	if r.exitCode != 0 {
		t.Fatalf("pick should find expired-claim task, got exit %d: %s", r.exitCode, r.stderr)
	}

	var picked taskJSON
	if err := json.Unmarshal([]byte(r.stdout), &picked); err != nil {
		t.Fatalf("parsing pick output: %v\nstdout: %s", err, r.stdout)
	}

	if picked.ID != 1 {
		t.Errorf("pick selected task #%d, want #1 (expired claim should be available)", picked.ID)
	}
}

func TestListUnclaimedFilter(t *testing.T) {
	kanbanDir := initBoard(t)

	// Task 1: claimed (should be excluded from --unclaimed list).
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	// Task 2: unclaimed.
	writeTaskFile(t, kanbanDir, 2, `---
id: 2
title: Free task
status: todo
priority: medium
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
---
`)
	// Task 3: expired claim (should appear as unclaimed).
	writeTaskFile(t, kanbanDir, 3, `---
id: 3
title: Expired claim task
status: todo
priority: low
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 4)

	var tasks []taskJSON
	r := runKanbanJSON(t, kanbanDir, &tasks, "list", "--unclaimed")
	if r.exitCode != 0 {
		t.Fatalf("list --unclaimed failed (exit %d): %s", r.exitCode, r.stderr)
	}

	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2 (unclaimed + expired)", len(tasks))
		for _, tk := range tasks {
			t.Logf("  got task #%d %q", tk.ID, tk.Title)
		}
	}

	ids := make(map[int]bool, len(tasks))
	for _, tk := range tasks {
		ids[tk.ID] = true
	}
	if ids[1] {
		t.Error("task #1 (active claim) should NOT appear in --unclaimed list")
	}
	if !ids[2] {
		t.Error("task #2 (unclaimed) should appear in --unclaimed list")
	}
	if !ids[3] {
		t.Error("task #3 (expired claim) should appear in --unclaimed list")
	}
}

func TestListClaimedByFilter(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Agent alpha task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	writeTaskFile(t, kanbanDir, 2, `---
id: 2
title: Agent beta task
status: todo
priority: medium
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-beta
claimed_at: 2099-01-01T00:00:00Z
---
`)
	writeTaskFile(t, kanbanDir, 3, `---
id: 3
title: Unclaimed task
status: todo
priority: low
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 4)

	var tasks []taskJSON
	r := runKanbanJSON(t, kanbanDir, &tasks, "list", "--claimed-by", "agent-alpha")
	if r.exitCode != 0 {
		t.Fatalf("list --claimed-by failed (exit %d): %s", r.exitCode, r.stderr)
	}

	if len(tasks) != 1 {
		t.Errorf("got %d tasks, want 1", len(tasks))
	}
	if len(tasks) > 0 && tasks[0].ID != 1 {
		t.Errorf("got task #%d, want #1", tasks[0].ID)
	}
}

func TestExpiredClaimClearedAfterMutation(t *testing.T) {
	kanbanDir := initBoard(t)

	// Expired claim — move should succeed and clear the claim fields.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim cleared
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	r := runKanban(t, kanbanDir, "move", "1", "in-progress")
	if r.exitCode != 0 {
		t.Fatalf("move should succeed for expired claim, got exit %d: %s", r.exitCode, r.stderr)
	}

	// Read the task back and verify claim fields are cleared.
	var tk taskJSON
	r = runKanbanJSON(t, kanbanDir, &tk, "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// Verify the task file no longer has claim fields by reading raw content.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-expired-claim-cleared.md")
	data, err := os.ReadFile(taskPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading task file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "claimed_by:") {
		t.Error("task file should not contain claimed_by after expired claim is cleared")
	}
	if strings.Contains(content, "claimed_at:") {
		t.Error("task file should not contain claimed_at after expired claim is cleared")
	}
}

// ---------------------------------------------------------------------------
// Read command coverage tests (list, show, board, log, metrics)
// ---------------------------------------------------------------------------

func TestListGroupByStatus(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A", "--status", "todo")
	mustCreateTask(t, kanbanDir, "Task B", "--status", "todo")
	taskC := mustCreateTask(t, kanbanDir, "Task C")
	runKanban(t, kanbanDir, "move", strconv.Itoa(taskC.ID), "in-progress")

	// Table output with --group-by status.
	r := runKanban(t, kanbanDir, "list", "--group-by", "status")
	if r.exitCode != 0 {
		t.Fatalf("list --group-by status failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "todo") {
		t.Error("expected group header 'todo' in output")
	}

	// JSON output with --group-by status.
	var grouped struct {
		Groups []struct {
			Key string `json:"key"`
		} `json:"groups"`
	}
	r = runKanbanJSON(t, kanbanDir, &grouped, "list", "--group-by", "status")
	if r.exitCode != 0 {
		t.Fatalf("list --group-by --json failed (exit %d): %s", r.exitCode, r.stderr)
	}
	foundTodo := false
	for _, g := range grouped.Groups {
		if g.Key == "todo" {
			foundTodo = true
		}
	}
	if !foundTodo {
		t.Error("JSON output should have group with key 'todo'")
	}
}

func TestListGroupByPriority(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "High task", "--priority", "high")
	mustCreateTask(t, kanbanDir, "Low task", "--priority", "low")

	r := runKanban(t, kanbanDir, "list", "--group-by", "priority")
	if r.exitCode != 0 {
		t.Fatalf("list --group-by priority failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "high") {
		t.Error("expected group header 'high' in output")
	}
}

func TestListGroupByInvalid(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "list", "--group-by", "invalid-field")
	if errResp.Code != "INVALID_GROUP_BY" {
		t.Errorf("error code = %q, want INVALID_GROUP_BY", errResp.Code)
	}
}

func TestBoardGroupByStatus(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A", "--status", "todo")
	taskB := mustCreateTask(t, kanbanDir, "Task B")
	runKanban(t, kanbanDir, "move", strconv.Itoa(taskB.ID), "in-progress")

	// Table output.
	r := runKanban(t, kanbanDir, "board", "--group-by", "status")
	if r.exitCode != 0 {
		t.Fatalf("board --group-by status failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// JSON output.
	var grouped map[string]interface{}
	r = runKanbanJSON(t, kanbanDir, &grouped, "board", "--group-by", "status")
	if r.exitCode != 0 {
		t.Fatalf("board --group-by --json failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestBoardGroupByInvalid(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "board", "--group-by", "invalid-field")
	if errResp.Code != "INVALID_GROUP_BY" {
		t.Errorf("error code = %q, want INVALID_GROUP_BY", errResp.Code)
	}
}

func TestShowCompactOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Compact show test")

	r := runKanban(t, kanbanDir, "show", "1", "--compact")
	if r.exitCode != 0 {
		t.Fatalf("show --compact failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Compact show test") {
		t.Error("compact show output should contain task title")
	}
}

func TestLogCompactOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Log test task")

	r := runKanban(t, kanbanDir, "log", "--compact")
	if r.exitCode != 0 {
		t.Fatalf("log --compact failed (exit %d): %s", r.exitCode, r.stderr)
	}
	// Activity log should have the create entry.
	if !strings.Contains(r.stdout, "create") {
		t.Error("compact log output should contain 'create' action")
	}
}

func TestMetricsCompactOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Metrics test task")

	r := runKanban(t, kanbanDir, "metrics", "--compact")
	if r.exitCode != 0 {
		t.Fatalf("metrics --compact failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestBoardCompactOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Board compact task")

	r := runKanban(t, kanbanDir, "board", "--compact")
	if r.exitCode != 0 {
		t.Fatalf("board --compact failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestListCompactOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Compact list task")

	r := runKanban(t, kanbanDir, "list", "--compact")
	if r.exitCode != 0 {
		t.Fatalf("list --compact failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Compact list task") {
		t.Error("compact list output should contain task title")
	}
}

func TestLogWithFilters(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "First task")
	mustCreateTask(t, kanbanDir, "Second task")

	// Filter by action.
	r := runKanban(t, kanbanDir, "log", "--action", "create", "--json")
	if r.exitCode != 0 {
		t.Fatalf("log --action create failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// Filter by task ID.
	r = runKanban(t, kanbanDir, "log", "--task", "1", "--json")
	if r.exitCode != 0 {
		t.Fatalf("log --task 1 failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// Limit.
	r = runKanban(t, kanbanDir, "log", "--limit", "1", "--json")
	if r.exitCode != 0 {
		t.Fatalf("log --limit 1 failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestMetricsWithSince(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Old task")

	r := runKanban(t, kanbanDir, "metrics", "--since", "2020-01-01", "--json")
	if r.exitCode != 0 {
		t.Fatalf("metrics --since failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestMetricsWithBadSince(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "metrics", "--since", "not-a-date")
	if errResp.Code != codeInvalidDate {
		t.Errorf("error code = %q, want %q", errResp.Code, codeInvalidDate)
	}
}

func TestLogWithBadSince(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "log", "--since", "not-a-date")
	if errResp.Code != codeInvalidDate {
		t.Errorf("error code = %q, want %q", errResp.Code, codeInvalidDate)
	}
}
