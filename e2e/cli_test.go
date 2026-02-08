package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
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

	//nolint:gosec,noctx // building test binary in TestMain (no context available)
	build := exec.Command("go", "build", "-o", binPath, "..")
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
	if errResp.Code != "INVALID_DATE" {
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
			InProgress int `json:"in_progress"`
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
	if ctx.Summary.InProgress < 1 {
		t.Errorf("InProgress = %d, want >= 1", ctx.Summary.InProgress)
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
// Default output format tests (table is default, no --json or --table flag)
// ---------------------------------------------------------------------------

func TestDefaultOutputList(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Default output task", "--tags", "test,demo")

	r := runKanban(t, kanbanDir, "list")
	if r.exitCode != 0 {
		t.Fatalf("list failed: %s", r.stderr)
	}
	// Should be table output with column headers, not JSON.
	if !strings.Contains(r.stdout, "ID") || !strings.Contains(r.stdout, "STATUS") {
		t.Errorf("default output missing table headers:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Default output task") {
		t.Errorf("default output missing task title:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "TAGS") {
		t.Errorf("default output missing TAGS column:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "test,demo") {
		t.Errorf("default output missing tag values:\n%s", r.stdout)
	}
	// Must NOT be JSON.
	if strings.HasPrefix(strings.TrimSpace(r.stdout), "[") {
		t.Errorf("default output is JSON, expected table:\n%s", r.stdout)
	}
}

func TestDefaultOutputCreate(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "create", "New task via default")
	if r.exitCode != 0 {
		t.Fatalf("create failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Created task #1") {
		t.Errorf("default create missing confirmation:\n%s", r.stdout)
	}
	// Must NOT be JSON.
	if strings.HasPrefix(strings.TrimSpace(r.stdout), "{") {
		t.Errorf("default create output is JSON, expected table:\n%s", r.stdout)
	}
}

func TestDefaultOutputMove(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Movable task")

	r := runKanban(t, kanbanDir, "move", "1", "todo")
	if r.exitCode != 0 {
		t.Fatalf("move failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Moved task #1") {
		t.Errorf("default move missing confirmation:\n%s", r.stdout)
	}
}

func TestDefaultOutputDelete(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Deletable task")

	r := runKanban(t, kanbanDir, "delete", "1", "--force")
	if r.exitCode != 0 {
		t.Fatalf("delete failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Deleted task #1") {
		t.Errorf("default delete missing confirmation:\n%s", r.stdout)
	}
}

func TestDefaultOutputBoard(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Board task")

	r := runKanban(t, kanbanDir, "board")
	if r.exitCode != 0 {
		t.Fatalf("board failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "STATUS") || !strings.Contains(r.stdout, "COUNT") {
		t.Errorf("default board missing table headers:\n%s", r.stdout)
	}
	// Must NOT be JSON.
	if strings.HasPrefix(strings.TrimSpace(r.stdout), "{") {
		t.Errorf("default board output is JSON, expected table:\n%s", r.stdout)
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
		"-o", versionBin, "..")
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

func TestDefaultOutputMetrics(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "metrics")
	if r.exitCode != 0 {
		t.Fatalf("metrics failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Throughput") {
		t.Errorf("default metrics missing throughput:\n%s", r.stdout)
	}
}
