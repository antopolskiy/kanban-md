package board

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func setupMutateBoard(t *testing.T) string {
	dir := t.TempDir()
	cfg := config.NewDefault(dir)
	cfg.SetDir(dir)
	if err := os.MkdirAll(cfg.TasksPath(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}

func CreateTaskFileWithStatus(t *testing.T, tasksDir string, id int, title, status string) {
	tk := &task.Task{ID: id, Title: title, Status: status}
	if err := task.Write(filepath.Join(tasksDir, strconv.Itoa(id)+".md"), tk); err != nil {
		t.Fatal(err)
	}
}

func WriteTaskWithClass(t *testing.T, tasksDir string, id int, title, status, class string) {
	tk := &task.Task{ID: id, Title: title, Status: status, Class: class}
	if err := task.Write(filepath.Join(tasksDir, strconv.Itoa(id)+".md"), tk); err != nil {
		t.Fatal(err)
	}
}

// --- logEditActivity tests ---



func TestLogEditActivity_BlockTransition(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", Blocked: true, BlockReason: "dependency"}
	logEditTransitions(cfg, tk, false, "")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "block") {
		t.Errorf("expected 'block' action in log, got: %s", got)
	}
}

func TestLogEditActivity_UnblockTransition(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", Blocked: false}
	logEditTransitions(cfg, tk, true, "")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "unblock") {
		t.Errorf("expected 'unblock' action in log, got: %s", got)
	}
}

func TestLogEditActivity_ClaimTransition(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", ClaimedBy: "agent-1"}
	logEditTransitions(cfg, tk, false, "")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "claim") {
		t.Errorf("expected 'claim' action in log, got: %s", got)
	}
}

func TestLogEditActivity_ReleaseTransition(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	tk := &task.Task{ID: 1, Title: "test", ClaimedBy: ""}
	logEditTransitions(cfg, tk, false, "agent-1")

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !containsSubstring(got, "release") {
		t.Errorf("expected 'release' action in log, got: %s", got)
	}
}

// --- enforceWIPLimit ---

func TestEnforceWIPLimit_NoLimit(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	err = enforceMoveWIP(cfg, &task.Task{Status: "backlog"}, "todo")
	if err != nil {
		t.Errorf("expected nil with no WIP limit, got: %v", err)
	}
}

func TestEnforceWIPLimit_Exceeded(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.WIPLimits = map[string]int{"todo": 1}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}
	CreateTaskFileWithStatus(t, cfg.TasksPath(), 1, "in-todo", "todo")

	cfg, err = config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	err = enforceMoveWIP(cfg, &task.Task{Status: "backlog"}, "todo")
	if err == nil {
		t.Fatal("expected WIP limit exceeded error")
	}
}

// --- enforceWIPLimitForClass ---

func TestEnforceWIPLimitForClass_NilClass(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Task with unknown class — ClassByName returns nil.
	tk := &task.Task{ID: 1, Class: "nonexistent", Status: "backlog"}
	err = enforceClassWIP(cfg, tk, "todo")
	if err != nil {
		t.Errorf("expected nil when class not configured, got: %v", err)
	}
}

func TestEnforceWIPLimitForClass_BypassColumnWIP(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	// Set a tight column WIP limit that would normally fail.
	cfg.WIPLimits = map[string]int{"todo": 1}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}
	CreateTaskFileWithStatus(t, cfg.TasksPath(), 2, "existing-todo", "todo")

	cfg, err = config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Expedite class bypasses column WIP by default.
	tk := &task.Task{ID: 1, Class: "expedite", Status: "backlog"}
	err = enforceClassWIP(cfg, tk, "todo")
	if err != nil {
		t.Errorf("expected nil when class bypasses column WIP, got: %v", err)
	}
}

func TestEnforceWIPLimitForClass_ClassWIPExceeded(t *testing.T) {
	kanbanDir := setupMutateBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create an expedite task (default WIP limit = 1).
	WriteTaskWithClass(t, cfg.TasksPath(), 2, "other-expedite", "todo", "expedite")

	tk := &task.Task{ID: 3, Class: "expedite", Status: "backlog"}
	err = enforceClassWIP(cfg, tk, "todo")
	if err == nil {
		t.Fatal("expected class WIP exceeded error")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T: %v", err, err)
	}
	if cliErr.Code != clierr.ClassWIPExceeded {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.ClassWIPExceeded)
	}
}

// --- countByClass ---

func TestCountByClass_Counts(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Class: "expedite"},
		{ID: 2, Class: "expedite"},
		{ID: 3, Class: "standard"},
	}
	// Exclude ID 1 — should only count ID 2.
	got := countByClass(tasks, "expedite", 1)
	if got != 1 {
		t.Errorf("countByClass = %d, want 1", got)
	}
}

func TestCountByClass_NoneMatch(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Class: "standard"},
	}
	got := countByClass(tasks, "expedite", 0)
	if got != 0 {
		t.Errorf("countByClass = %d, want 0", got)
	}
}
