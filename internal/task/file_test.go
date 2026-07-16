package task

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// ---------------------------------------------------------------------------
// File protection: chmod behavior in Write/WriteAndRename
// ---------------------------------------------------------------------------

const testClaimAgent = "agent-x"

func newTestTask(title string) *Task {
	now := time.Now()
	return &Task{
		ID:       1,
		Title:    title,
		Status:   "todo",
		Priority: "medium",
		Created:  now,
		Updated:  now,
	}
}

func TestWrite_ClaimedTaskBecomesReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "001-claimed.md")

	tk := newTestTask("Claimed")
	now := time.Now()
	tk.ClaimedBy = testClaimAgent
	tk.ClaimedAt = &now

	if err := Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm()&0o200 != 0 {
		t.Errorf("claimed file should be read-only (0o444), got %o", info.Mode().Perm())
	}
}

func TestWrite_UnclaimedTaskStaysWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "001-unclaimed.md")

	tk := newTestTask("Unclaimed")

	if err := Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm()&0o200 == 0 {
		t.Errorf("unclaimed file should be writable (0o600), got %o", info.Mode().Perm())
	}
}

func TestWrite_OverwriteReadOnlyFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "001-overwrite.md")

	now := time.Now()
	tk := newTestTask("Overwrite")
	tk.ClaimedBy = testClaimAgent
	tk.ClaimedAt = &now

	// First write → becomes 0o444.
	if err := Write(path, tk); err != nil {
		t.Fatalf("first Write() error: %v", err)
	}

	// Second write on same 0o444 file should succeed.
	tk.Priority = "high"
	if err := Write(path, tk); err != nil {
		t.Fatalf("second Write() on read-only file error: %v", err)
	}

	// Still read-only after overwrite.
	info, _ := os.Stat(path)
	if info.Mode().Perm()&0o200 != 0 {
		t.Errorf("file should still be read-only after overwrite, got %o", info.Mode().Perm())
	}
}

func TestWrite_ReleaseRestoresWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "001-release.md")

	now := time.Now()
	tk := newTestTask("Release")
	tk.ClaimedBy = testClaimAgent
	tk.ClaimedAt = &now

	// Write claimed → 0o444.
	if err := Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm()&0o200 != 0 {
		t.Fatal("precondition: claimed file should be read-only")
	}

	// Release claim and write again → 0o600.
	tk.ClaimedBy = ""
	tk.ClaimedAt = nil
	if err := Write(path, tk); err != nil {
		t.Fatalf("Write() after release error: %v", err)
	}
	info, _ = os.Stat(path)
	if info.Mode().Perm()&0o200 == 0 {
		t.Errorf("released file should be writable, got %o", info.Mode().Perm())
	}
}

func TestWriteAndRename_ClaimedPreservesProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "001-old-title.md")

	now := time.Now()
	tk := newTestTask("Old title")
	tk.ClaimedBy = testClaimAgent
	tk.ClaimedAt = &now

	if err := Write(oldPath, tk); err != nil {
		t.Fatalf("initial Write() error: %v", err)
	}

	// Rename by changing title.
	tk.Title = "New title"
	newPath, err := WriteAndRename(oldPath, tk, "Old title")
	if err != nil {
		t.Fatalf("WriteAndRename() error: %v", err)
	}

	// Old file gone.
	if _, statErr := os.Stat(oldPath); !os.IsNotExist(statErr) {
		t.Error("old file should be removed after rename")
	}

	// New file is read-only.
	info, err := os.Stat(newPath)
	if err != nil {
		t.Fatalf("Stat new file error: %v", err)
	}
	if info.Mode().Perm()&0o200 != 0 {
		t.Errorf("renamed claimed file should be read-only, got %o", info.Mode().Perm())
	}
}

func TestReadMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001-missing-fields.md")
	content := `---
title: Missing id
status: backlog
priority: medium
created: 2026-02-24T12:00:00Z
updated: 2026-02-24T12:00:00Z
---
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "missing required field: id") {
		t.Fatalf("error = %v, want missing required field message", err)
	}
}
