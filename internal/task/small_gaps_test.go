package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
)

// ---------------------------------------------------------------------------
// ValidateClaimRequired — basic instantiation test (0% → covered)
// ---------------------------------------------------------------------------

func TestValidateClaimRequired_ReturnsError(t *testing.T) {
	err := ValidateClaimRequired("in-progress")
	if err == nil {
		t.Fatal("expected non-nil error from ValidateClaimRequired")
	}
	if err.Code == "" {
		t.Error("error code should be set")
	}
	wantSubstr := "in-progress"
	if msg := err.Error(); !containsStr(msg, wantSubstr) {
		t.Errorf("error = %q, want to contain %q", msg, wantSubstr)
	}
}

// ---------------------------------------------------------------------------
// UpdateTimestamps — reopen from terminal status
// ---------------------------------------------------------------------------

func TestUpdateTimestamps_ReopenFromTerminal(t *testing.T) {
	cfg := config.NewDefault("Test")
	started := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	tk := &Task{
		Status:    "todo",
		Started:   &started,
		Completed: &completed,
	}

	// Move from terminal ("done") back to non-terminal ("todo").
	UpdateTimestamps(tk, "done", "todo", cfg)

	if tk.Completed != nil {
		t.Error("Completed should be cleared when reopening from terminal status")
	}
	if tk.Started == nil {
		t.Error("Started should be preserved when reopening")
	}
}

// ---------------------------------------------------------------------------
// UpdateTimestamps — direct move to terminal sets both
// ---------------------------------------------------------------------------

func TestUpdateTimestamps_DirectToTerminalSetsStarted(t *testing.T) {
	cfg := config.NewDefault("Test")
	tk := &Task{Status: "done"}

	// Direct move from initial to terminal (backlog → done).
	UpdateTimestamps(tk, "backlog", "done", cfg)

	if tk.Started == nil {
		t.Error("Started should be set on direct move to terminal")
	}
	if tk.Completed == nil {
		t.Error("Completed should be set on direct move to terminal")
	}
}

// ---------------------------------------------------------------------------
// UpdateTimestamps — move to archived (terminal) sets Completed
// ---------------------------------------------------------------------------

func TestUpdateTimestamps_MoveToArchived(t *testing.T) {
	cfg := config.NewDefault("Test")
	started := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	tk := &Task{Status: "archived", Started: &started}

	UpdateTimestamps(tk, "done", "archived", cfg)

	if tk.Completed == nil {
		t.Fatal("Completed should be set on move to archived")
	}
}

// ---------------------------------------------------------------------------
// Write — yaml.Marshal error path (not easily triggered, but WriteFile
// error on directory serves as proxy)
// ---------------------------------------------------------------------------

func TestWrite_DirectoryAsPath(t *testing.T) {
	dir := t.TempDir()
	tk := &Task{ID: 1, Title: "Test", Status: "backlog", Priority: "medium"}
	// Try to write to a path that is a directory.
	err := Write(dir, tk)
	if err == nil {
		t.Fatal("expected error when writing task to a directory path")
	}
}

// ---------------------------------------------------------------------------
// FindByID — edge case IDs
// ---------------------------------------------------------------------------

func TestFindByID_ZeroID(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "Task one", "backlog")

	_, err := FindByID(dir, 0)
	if err == nil {
		t.Fatal("expected error for ID 0")
	}
}

func TestFindByID_LargeID(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "Task one", "backlog")

	_, err := FindByID(dir, 999999)
	if err == nil {
		t.Fatal("expected error for very large ID not present")
	}
}

func TestFindByID_NoDashInFilename(t *testing.T) {
	dir := t.TempDir()
	// Create a .md file without a dash in the name.
	if err := os.WriteFile(filepath.Join(dir, "nodash.md"), []byte("---\nid: 1\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := FindByID(dir, 1)
	// Should not find it (no dash prefix match), so error expected.
	if err == nil {
		t.Fatal("expected error for filename without ID-dash pattern")
	}
}

// ---------------------------------------------------------------------------
// ReadAllLenient — mixed valid and invalid files
// ---------------------------------------------------------------------------

func TestReadAllLenient_MixedFiles(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "Good task", "backlog")
	createTestTask(t, dir, 2, "Another good", "todo")

	// Write two broken files.
	if err := os.WriteFile(filepath.Join(dir, "003-broken.md"), []byte("not frontmatter"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "004-also-broken.md"), []byte("bad content"), 0o600); err != nil {
		t.Fatal(err)
	}

	tasks, warnings, err := ReadAllLenient(dir)
	if err != nil {
		t.Fatalf("ReadAllLenient() error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("tasks = %d, want 2", len(tasks))
	}
	if len(warnings) != 2 {
		t.Errorf("warnings = %d, want 2", len(warnings))
	}
}

// ---------------------------------------------------------------------------
// ReadAllLenient — directory with only non-.md files
// ---------------------------------------------------------------------------

func TestReadAllLenient_OnlyNonMDFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	tasks, warnings, err := ReadAllLenient(dir)
	if err != nil {
		t.Fatalf("ReadAllLenient() error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("tasks = %d, want 0", len(tasks))
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %d, want 0", len(warnings))
	}
}

// ---------------------------------------------------------------------------
// Write — empty body
// ---------------------------------------------------------------------------

func TestWrite_EmptyBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001-test.md")
	tk := &Task{
		ID:       1,
		Title:    "Test",
		Status:   "backlog",
		Priority: "medium",
		Body:     "",
	}

	if err := Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // test file
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	// Should end with closing --- and no extra body content.
	if !containsStr(content, "---\n") {
		t.Error("file should contain closing frontmatter delimiter")
	}
}

// ---------------------------------------------------------------------------
// FindByID — leading zero handling
// ---------------------------------------------------------------------------

func TestFindByID_LeadingZeros(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 42, "Task forty-two", "backlog")

	path, err := FindByID(dir, 42)
	if err != nil {
		t.Fatalf("FindByID(42) error: %v", err)
	}
	if path == "" {
		t.Error("FindByID should find task with zero-padded filename")
	}
}

// Note: containsStr helper is defined in validate_extended_test.go.
