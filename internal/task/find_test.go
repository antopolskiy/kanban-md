package task

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/clierr"
)

func createTestTask(t *testing.T, dir string, id int, title, status string) {
	t.Helper()
	slug := GenerateSlug(title)
	filename := GenerateFilename(id, slug)
	path := filepath.Join(dir, filename)

	task := &Task{
		ID:       id,
		Title:    title,
		Status:   status,
		Priority: "medium",
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	if err := Write(path, task); err != nil {
		t.Fatalf("creating test task: %v", err)
	}
}

func TestFindByID(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "First task", "backlog")
	createTestTask(t, dir, 2, "Second task", "todo")

	path, err := FindByID(dir, 1)
	if err != nil {
		t.Fatalf("FindByID(1) error: %v", err)
	}
	if filepath.Base(path) != "001-first-task.md" {
		t.Errorf("FindByID(1) = %q, want 001-first-task.md", filepath.Base(path))
	}

	path, err = FindByID(dir, 2)
	if err != nil {
		t.Fatalf("FindByID(2) error: %v", err)
	}
	if filepath.Base(path) != "002-second-task.md" {
		t.Errorf("FindByID(2) = %q, want 002-second-task.md", filepath.Base(path))
	}
}

func TestFindByIDNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindByID(dir, 99)
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) || cliErr.Code != clierr.TaskNotFound {
		t.Errorf("FindByID(99) error = %v, want *clierr.Error with code TASK_NOT_FOUND", err)
	}
}

func TestReadAll(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "First", "backlog")
	createTestTask(t, dir, 2, "Second", "todo")
	createTestTask(t, dir, 3, "Third", "done")

	// Add a non-md file that should be skipped.
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("ignore me"), 0o600); err != nil {
		t.Fatal(err)
	}

	tasks, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("ReadAll() returned %d tasks, want 3", len(tasks))
	}
}

func TestReadAllEmptyDir(t *testing.T) {
	dir := t.TempDir()
	tasks, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("ReadAll() returned %d tasks, want 0", len(tasks))
	}
}

func TestReadAllNonexistentDir(t *testing.T) {
	tasks, err := ReadAll("/nonexistent/path")
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if tasks != nil {
		t.Errorf("ReadAll() = %v, want nil", tasks)
	}
}

func BenchmarkFindByID(b *testing.B) {
	dir := b.TempDir()
	// Create several tasks so the directory scan has realistic work.
	for i := 1; i <= 10; i++ {
		slug := GenerateSlug("benchmark task")
		filename := GenerateFilename(i, slug)
		path := filepath.Join(dir, filename)
		task := &Task{
			ID: i, Title: "benchmark task", Status: "backlog",
			Priority: "medium", Created: time.Now(), Updated: time.Now(),
		}
		if err := Write(path, task); err != nil {
			b.Fatalf("creating test task: %v", err)
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = FindByID(dir, 5)
	}
}

func BenchmarkExtractIDFromFilename(b *testing.B) {
	for b.Loop() {
		_, _ = ExtractIDFromFilename("042-fix-bug.md")
	}
}

func TestExtractIDFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		wantID   int
		wantErr  bool
	}{
		{"001-setup-database.md", 1, false},
		{"042-fix-bug.md", 42, false},
		{"1000-big-project.md", 1000, false},
		{"no-id.md", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		id, err := ExtractIDFromFilename(tt.filename)
		if (err != nil) != tt.wantErr {
			t.Errorf("ExtractIDFromFilename(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
			continue
		}
		if id != tt.wantID {
			t.Errorf("ExtractIDFromFilename(%q) = %d, want %d", tt.filename, id, tt.wantID)
		}
	}
}
