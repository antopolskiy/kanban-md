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
	if task.Status != "backlog" {
		t.Errorf("Status = %q, want %q (default)", task.Status, "backlog")
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
	if task.Priority != "high" {
		t.Errorf("Priority = %q, want %q", task.Priority, "high")
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
	r := runKanban(t, kanbanDir, "--json", "create", "Bad task", "--status", "nonexistent")

	if r.exitCode == 0 {
		t.Error("expected non-zero exit for invalid status")
	}
	if !strings.Contains(r.stderr, "invalid status") {
		t.Errorf("stderr = %q, want 'invalid status'", r.stderr)
	}
}

func TestCreateBadDateFormat(t *testing.T) {
	kanbanDir := initBoard(t)
	r := runKanban(t, kanbanDir, "--json", "create", "Bad date", "--due", "02-15-2026")

	if r.exitCode == 0 {
		t.Error("expected non-zero exit for bad date format")
	}
	if !strings.Contains(r.stderr, "invalid due date") {
		t.Errorf("stderr = %q, want 'invalid due date'", r.stderr)
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

	r := runKanban(t, kanbanDir, "--json", "show", "999")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit for missing task")
	}
	if !strings.Contains(r.stderr, "not found") {
		t.Errorf("stderr = %q, want 'not found'", r.stderr)
	}
}

func TestShowInvalidID(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--json", "show", "abc")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit for non-numeric ID")
	}
	if !strings.Contains(r.stderr, "invalid task ID") {
		t.Errorf("stderr = %q, want 'invalid task ID'", r.stderr)
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
				if task.Priority != "high" {
					t.Errorf("Priority = %q, want %q", task.Priority, "high")
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

	r := runKanban(t, kanbanDir, "--json", "edit", "1")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit for no changes")
	}
	if !strings.Contains(r.stderr, "no changes") {
		t.Errorf("stderr = %q, want 'no changes'", r.stderr)
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
	r := runKanban(t, kanbanDir, "--json", "move", "1", "--prev")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit for --prev at first status")
	}
	if !strings.Contains(r.stderr, "first status") {
		t.Errorf("stderr = %q, want 'first status'", r.stderr)
	}

	// Move to last status, then try --next.
	runKanban(t, kanbanDir, "--json", "move", "1", "done")
	r = runKanban(t, kanbanDir, "--json", "move", "1", "--next")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit for --next at last status")
	}
	if !strings.Contains(r.stderr, "last status") {
		t.Errorf("stderr = %q, want 'last status'", r.stderr)
	}
}

func TestMoveNoStatusSpecified(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "No direction")

	r := runKanban(t, kanbanDir, "--json", "move", "1")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit when no status given")
	}
	if !strings.Contains(r.stderr, "provide a target status") {
		t.Errorf("stderr = %q, want 'provide a target status'", r.stderr)
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
	if got.Status != "backlog" {
		t.Errorf("Status = %q, want %q", got.Status, "backlog")
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
	r := runKanban(t, kanbanDir, "--json", "move", "2", "in-progress")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit when WIP limit reached")
	}
	if !strings.Contains(r.stderr, "WIP limit") {
		t.Errorf("stderr = %q, want WIP limit message", r.stderr)
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
	r := runKanban(t, kanbanDir, "--json", "create", "Task C", "--status", "in-progress")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit when WIP limit reached on create")
	}
	if !strings.Contains(r.stderr, "WIP limit") {
		t.Errorf("stderr = %q, want WIP limit message", r.stderr)
	}
}

func TestEditStatusRespectsWIPLimit(t *testing.T) {
	kanbanDir := initBoardWithWIP(t, 1)

	mustCreateTask(t, kanbanDir, "Task A", "--status", "in-progress")
	mustCreateTask(t, kanbanDir, "Task B")

	// Edit task B status to in-progress should fail.
	r := runKanban(t, kanbanDir, "--json", "edit", "2", "--status", "in-progress")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit when WIP limit reached on edit --status")
	}
	if !strings.Contains(r.stderr, "WIP limit") {
		t.Errorf("stderr = %q, want WIP limit message", r.stderr)
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

	r := runKanban(t, kanbanDir, "--json", "edit", "1", "--started", "2026-01-15", "--clear-started")
	if r.exitCode == 0 {
		t.Error("expected error for --started + --clear-started conflict")
	}
	if !strings.Contains(r.stderr, "cannot use") {
		t.Errorf("stderr = %q, want conflict error", r.stderr)
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
	r := runKanban(t, kanbanDir, "--json", "create", "Self dep", "--depends-on", "2")
	if r.exitCode == 0 {
		t.Error("expected error for self-dependency")
	}
	if !strings.Contains(r.stderr, "depend on itself") {
		t.Errorf("stderr = %q, want self-dep error", r.stderr)
	}
}

func TestCreateInvalidDepErrors(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--json", "create", "Bad dep", "--depends-on", "99")
	if r.exitCode == 0 {
		t.Error("expected error for invalid dependency ID")
	}
	if !strings.Contains(r.stderr, "not found") {
		t.Errorf("stderr = %q, want 'not found'", r.stderr)
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

	r := runKanban(t, kanbanDir, "--json", "edit", "1", "--add-dep", "1")
	if r.exitCode == 0 {
		t.Error("expected error for self-dependency on edit")
	}
	if !strings.Contains(r.stderr, "depend on itself") {
		t.Errorf("stderr = %q, want self-dep error", r.stderr)
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

	r := runKanban(t, kanbanDir, "--json", "delete", "1")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit for delete without --force in non-TTY")
	}
	if !strings.Contains(r.stderr, "not a terminal") {
		t.Errorf("stderr = %q, want 'not a terminal'", r.stderr)
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

	r := runKanban(t, kanbanDir, "--json", "edit", "1", "--block", "")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit for empty block reason")
	}
	if !strings.Contains(r.stderr, "block reason is required") {
		t.Errorf("stderr = %q, want 'block reason is required'", r.stderr)
	}
}

func TestBlockAndUnblockConflict(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Conflict")

	r := runKanban(t, kanbanDir, "--json", "edit", "1", "--block", "reason", "--unblock")
	if r.exitCode == 0 {
		t.Error("expected non-zero exit for --block + --unblock")
	}
	if !strings.Contains(r.stderr, "cannot use --block and --unblock together") {
		t.Errorf("stderr = %q, want conflict message", r.stderr)
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
