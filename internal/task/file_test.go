package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/date"
)

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001-test-task.md")

	due := date.New(2026, time.March, 15)
	original := &Task{
		ID:       1,
		Title:    "Test task",
		Status:   "todo",
		Priority: "high",
		Created:  time.Date(2026, 2, 7, 10, 0, 0, 0, time.UTC),
		Updated:  time.Date(2026, 2, 7, 10, 0, 0, 0, time.UTC),
		Assignee: "santiago",
		Tags:     []string{"backend", "api"},
		Due:      &due,
		Body:     "This is the task body.\n\n- Item 1\n- Item 2\n",
	}

	if err := Write(path, original); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// Verify file was created.
	data, err := os.ReadFile(path) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	content := string(data)
	if content[:4] != "---\n" {
		t.Errorf("file should start with ---\\n, got %q", content[:4])
	}

	// Read it back.
	loaded, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	if loaded.ID != 1 {
		t.Errorf("ID = %d, want 1", loaded.ID)
	}
	if loaded.Title != "Test task" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Test task")
	}
	if loaded.Status != "todo" {
		t.Errorf("Status = %q, want %q", loaded.Status, "todo")
	}
	if loaded.Assignee != "santiago" {
		t.Errorf("Assignee = %q, want %q", loaded.Assignee, "santiago")
	}
	if loaded.Due == nil || loaded.Due.String() != "2026-03-15" {
		t.Errorf("Due = %v, want 2026-03-15", loaded.Due)
	}
	if loaded.Body != "This is the task body.\n\n- Item 1\n- Item 2\n" {
		t.Errorf("Body = %q", loaded.Body)
	}
	if loaded.File != path {
		t.Errorf("File = %q, want %q", loaded.File, path)
	}
}

func TestWriteNoBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "002-no-body.md")

	task := &Task{
		ID:       2,
		Title:    "No body task",
		Status:   "backlog",
		Priority: "low",
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	if err := Write(path, task); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	loaded, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	if loaded.Body != "" {
		t.Errorf("Body = %q, want empty", loaded.Body)
	}
}
