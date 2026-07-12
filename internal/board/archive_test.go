package board

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const testBacklogStatus = "backlog"

// Helper copied from cmd tests
func setupArchiveBoard(t *testing.T) string {
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

func writeArchiveTask(t *testing.T, cfg *config.Config, tk *task.Task) {
	path := filepath.Join(cfg.TasksPath(), "1.md")
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}
}

func TestArchive_BasicArchive(t *testing.T) {
	kanbanDir := setupArchiveBoard(t)
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

	res, err := Archive(cfg, 1, "", time.Now())
	if err != nil {
		t.Fatalf("Archive error: %v", err)
	}
	if res.OldStatus != testBacklogStatus {
		t.Errorf("oldStatus = %q, want %q", res.OldStatus, testBacklogStatus)
	}
	if res.Task.Status != config.ArchivedStatus {
		t.Errorf("Status = %q, want %q", res.Task.Status, config.ArchivedStatus)
	}
}

func TestArchive_AlreadyArchived(t *testing.T) {
	kanbanDir := setupArchiveBoard(t)
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

	res, err := Archive(cfg, 1, "", time.Now())
	if err != nil {
		t.Fatalf("Archive error: %v", err)
	}
	if res.OldStatus != "" {
		t.Errorf("oldStatus = %q, want empty for already-archived", res.OldStatus)
	}
	if res.Task.Status != config.ArchivedStatus {
		t.Errorf("Status = %q, want %q", res.Task.Status, config.ArchivedStatus)
	}
}

func TestArchive_TaskNotFound(t *testing.T) {
	kanbanDir := setupArchiveBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Archive(cfg, 999, "", time.Now())
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestArchive_LogsActivity(t *testing.T) {
	kanbanDir := setupArchiveBoard(t)
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

	_, err = Archive(cfg, 1, "", time.Now())
	if err != nil {
		t.Fatalf("Archive error: %v", err)
	}

	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(string(data), "move") {
		t.Errorf("expected 'move' in activity log, got: %s", data)
	}
}
