package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
