package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestExecuteArchiveCore_ArchivesAndIsIdempotent(t *testing.T) {
	kanbanDir := setupBoard(t)
	tasksDir := filepath.Join(kanbanDir, "tasks")
	createTaskFile(t, tasksDir, 1, "Archive me")

	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	archived, oldStatus, err := executeArchiveCore(cfg, 1)
	if err != nil {
		t.Fatalf("executeArchiveCore: %v", err)
	}
	if oldStatus != testBacklogStatus {
		t.Errorf("oldStatus = %q, want backlog", oldStatus)
	}
	if archived.Status != config.ArchivedStatus {
		t.Errorf("status = %q, want %q", archived.Status, config.ArchivedStatus)
	}

	again, oldStatus, err := executeArchiveCore(cfg, 1)
	if err != nil {
		t.Fatalf("executeArchiveCore second call: %v", err)
	}
	if oldStatus != "" {
		t.Errorf("oldStatus on idempotent archive = %q, want empty", oldStatus)
	}
	if again.Status != config.ArchivedStatus {
		t.Errorf("status after second call = %q, want %q", again.Status, config.ArchivedStatus)
	}
}

func TestSoftDeleteAndLog_KeepsFileAndArchivesTask(t *testing.T) {
	kanbanDir := setupBoard(t)
	tasksDir := filepath.Join(kanbanDir, "tasks")
	createTaskFile(t, tasksDir, 1, "Soft delete me")

	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatalf("read task: %v", err)
	}

	err = softDeleteAndLog(cfg, path, tk)
	if err != nil {
		t.Fatalf("softDeleteAndLog: %v", err)
	}

	_, err = os.Stat(path)
	if err != nil {
		t.Fatalf("task file should remain after soft-delete: %v", err)
	}

	updated, err := task.Read(path)
	if err != nil {
		t.Fatalf("read updated task: %v", err)
	}
	if updated.Status != config.ArchivedStatus {
		t.Errorf("status = %q, want %q", updated.Status, config.ArchivedStatus)
	}
}
