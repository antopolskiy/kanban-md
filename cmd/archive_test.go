package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const testBacklogStatus = "backlog"

// writeArchiveTask creates a task file in the config's tasks directory.
func writeArchiveTask(t *testing.T, cfg *config.Config, tk *task.Task) {
	t.Helper()
	slug := task.GenerateSlug(tk.Title)
	filename := task.GenerateFilename(tk.ID, slug)
	path := filepath.Join(cfg.TasksPath(), filename)
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}
}

// --- executeArchiveCore tests ---

func TestExecuteArchiveCore_BasicArchive(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "Archive me",
		Status:   testBacklogStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	tk, oldStatus, err := executeArchiveCore(cfg, 1)
	if err != nil {
		t.Fatalf("executeArchiveCore error: %v", err)
	}
	if oldStatus != testBacklogStatus {
		t.Errorf("oldStatus = %q, want %q", oldStatus, testBacklogStatus)
	}
	if tk.Status != config.ArchivedStatus {
		t.Errorf("Status = %q, want %q", tk.Status, config.ArchivedStatus)
	}
}

func TestExecuteArchiveCore_AlreadyArchived(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "Already archived",
		Status:   config.ArchivedStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	tk, oldStatus, err := executeArchiveCore(cfg, 1)
	if err != nil {
		t.Fatalf("executeArchiveCore error: %v", err)
	}
	if oldStatus != "" {
		t.Errorf("oldStatus = %q, want empty for already-archived", oldStatus)
	}
	if tk.Status != config.ArchivedStatus {
		t.Errorf("Status = %q, want %q", tk.Status, config.ArchivedStatus)
	}
}

func TestExecuteArchiveCore_TaskNotFound(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = executeArchiveCore(cfg, 999)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestExecuteArchiveCore_LogsActivity(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "Log this",
		Status:   "todo",
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	_, _, err = executeArchiveCore(cfg, 1)
	if err != nil {
		t.Fatalf("executeArchiveCore error: %v", err)
	}

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath) //nolint:gosec // test path from t.TempDir
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(string(data), "move") {
		t.Errorf("expected 'move' in activity log, got: %s", data)
	}
}

// --- executeArchive tests ---

func TestExecuteArchive_BasicArchive(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "Archive via wrapper",
		Status:   testBacklogStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	if err = executeArchive(cfg, 1); err != nil {
		t.Fatalf("executeArchive error: %v", err)
	}
}

func TestExecuteArchive_TaskNotFound(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	err = executeArchive(cfg, 999)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

// --- archiveSingleTask tests ---

func TestArchiveSingleTask_TableOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "Table archive",
		Status:   testBacklogStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err = archiveSingleTask(cfg, 1)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("archiveSingleTask error: %v", err)
	}
	if !containsSubstring(got, "Archived task #1") {
		t.Errorf("expected 'Archived task #1' in output, got: %s", got)
	}
}

func TestArchiveSingleTask_JSONOutput(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "JSON archive",
		Status:   testBacklogStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err = archiveSingleTask(cfg, 1)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("archiveSingleTask error: %v", err)
	}
	if !containsSubstring(got, `"changed": true`) {
		t.Errorf("expected '\"changed\": true' in JSON, got: %s", got)
	}
}

func TestArchiveSingleTask_AlreadyArchivedTable(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "Already done",
		Status:   config.ArchivedStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err = archiveSingleTask(cfg, 1)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("archiveSingleTask error: %v", err)
	}
	if !containsSubstring(got, "already archived") {
		t.Errorf("expected 'already archived' in output, got: %s", got)
	}
}

func TestArchiveSingleTask_AlreadyArchivedJSON(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "Already done JSON",
		Status:   config.ArchivedStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	setFlags(t, true, false, false)
	r, w := captureStdout(t)

	err = archiveSingleTask(cfg, 1)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("archiveSingleTask error: %v", err)
	}
	if !containsSubstring(got, `"changed": false`) {
		t.Errorf("expected '\"changed\": false' in JSON, got: %s", got)
	}
}

func TestArchiveSingleTask_TaskNotFound(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	err = archiveSingleTask(cfg, 999)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

// --- runArchive tests ---

func TestRunArchive_SingleTask(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeArchiveTask(t, cfg, &task.Task{
		ID:       1,
		Title:    "Run archive single",
		Status:   testBacklogStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	})

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err = runArchive(nil, []string{"1"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runArchive error: %v", err)
	}
	if !containsSubstring(got, "Archived task #1") {
		t.Errorf("expected 'Archived task #1' in output, got: %s", got)
	}
}

func TestRunArchive_BatchMode(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	for i := 1; i <= 3; i++ {
		writeArchiveTask(t, cfg, &task.Task{
			ID:       i,
			Title:    "Batch task " + idStr(i),
			Status:   testBacklogStatus,
			Priority: "medium",
			Created:  now,
			Updated:  now,
		})
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	setFlags(t, false, true, false)
	r, w := captureStdout(t)

	err = runArchive(nil, []string{"1,2,3"})
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runArchive batch error: %v", err)
	}
	if !containsSubstring(got, "3/3") {
		t.Errorf("expected '3/3' in batch output, got: %s", got)
	}
}

func TestRunArchive_InvalidID(t *testing.T) {
	err := runArchive(nil, []string{"abc"})
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}
