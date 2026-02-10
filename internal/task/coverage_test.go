package task

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- splitFrontmatter error paths ---

func TestSplitFrontmatter_NoPrefix(t *testing.T) {
	_, _, err := splitFrontmatter([]byte("no frontmatter here"))
	if err == nil {
		t.Fatal("expected error for missing --- prefix")
	}
	if !strings.Contains(err.Error(), "does not start with YAML frontmatter") {
		t.Errorf("error = %v, want 'does not start with YAML frontmatter'", err)
	}
}

func TestSplitFrontmatter_UnclosedFrontmatter(t *testing.T) {
	_, _, err := splitFrontmatter([]byte("---\ntitle: test\nno closing\n"))
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter")
	}
	if !strings.Contains(err.Error(), "unclosed frontmatter") {
		t.Errorf("error = %v, want 'unclosed frontmatter'", err)
	}
}

func TestSplitFrontmatter_ClosingAtEOF(t *testing.T) {
	// File ends with \n--- (no trailing newline after closing).
	data := []byte("---\ntitle: test\n---")
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("splitFrontmatter error: %v", err)
	}
	if string(fm) != "title: test\n" {
		t.Errorf("frontmatter = %q, want %q", string(fm), "title: test\n")
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestSplitFrontmatter_EmptyBody(t *testing.T) {
	// Closing --- followed by nothing.
	data := []byte("---\ntitle: test\n---\n")
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("splitFrontmatter error: %v", err)
	}
	if string(fm) != "title: test" {
		t.Errorf("frontmatter = %q, want %q", string(fm), "title: test")
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

// --- Read error paths ---

func TestRead_FileNotFound(t *testing.T) {
	_, err := Read("/nonexistent/path/task.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "reading task file") {
		t.Errorf("error = %v, want 'reading task file'", err)
	}
}

func TestRead_InvalidFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.md")
	// Write file with no frontmatter delimiter.
	if err := os.WriteFile(path, []byte("just plain text"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for invalid frontmatter")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("error = %v, want to contain 'parsing'", err)
	}
}

func TestRead_InvalidYAMLFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-yaml.md")
	// Write file with valid frontmatter delimiters but invalid YAML.
	content := "---\n: [invalid yaml\n---\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML in frontmatter")
	}
	if !strings.Contains(err.Error(), "parsing frontmatter") {
		t.Errorf("error = %v, want to contain 'parsing frontmatter'", err)
	}
}

// --- Write error paths ---

func TestWrite_PermissionError(t *testing.T) {
	task := &Task{ID: 1, Title: "Test", Status: "backlog", Priority: "medium"}
	err := Write("/nonexistent/dir/task.md", task)
	if err == nil {
		t.Fatal("expected error when write fails")
	}
}

func TestWrite_BodyWithTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001-test.md")
	task := &Task{
		ID:       1,
		Title:    "Test",
		Status:   "backlog",
		Priority: "medium",
		Body:     "Line one\nLine two\n",
	}

	if err := Write(path, task); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // test file
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	// Body already ends with \n, so Write should NOT add a double newline.
	if strings.Contains(content, "Line two\n\n") {
		t.Error("Write should not add extra newline when body already ends with one")
	}
	if !strings.HasSuffix(content, "Line two\n") {
		t.Errorf("file should end with body content + single newline, got: %q", content[len(content)-20:])
	}
}

func TestWrite_BodyWithoutTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001-test.md")
	task := &Task{
		ID:       1,
		Title:    "Test",
		Status:   "backlog",
		Priority: "medium",
		Body:     "No trailing newline",
	}

	if err := Write(path, task); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // test file
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	// Write should append a newline.
	if !strings.HasSuffix(content, "No trailing newline\n") {
		t.Errorf("file should end with body + added newline, got: %q", content[len(content)-30:])
	}
}

// --- FindByID error path ---

func TestFindByID_ReadDirError(t *testing.T) {
	_, err := FindByID("/nonexistent/path", 1)
	if err == nil {
		t.Fatal("expected error for unreadable directory")
	}
	if !strings.Contains(err.Error(), "reading tasks directory") {
		t.Errorf("error = %v, want 'reading tasks directory'", err)
	}
}

func TestFindByID_SkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory that looks like a task.
	if err := os.MkdirAll(filepath.Join(dir, "001-subdir.md"), 0o750); err != nil {
		t.Fatal(err)
	}
	createTestTask(t, dir, 2, "Real task", "backlog")

	path, err := FindByID(dir, 2)
	if err != nil {
		t.Fatalf("FindByID(2) error: %v", err)
	}
	if !strings.Contains(filepath.Base(path), "002") {
		t.Errorf("FindByID(2) = %q, want file with 002 prefix", path)
	}
}

func TestFindByID_SkipsNonMDFiles(t *testing.T) {
	dir := t.TempDir()
	// Create a non-.md file with a valid ID prefix.
	if err := os.WriteFile(filepath.Join(dir, "001-notes.txt"), []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	createTestTask(t, dir, 1, "Real task", "backlog")

	path, err := FindByID(dir, 1)
	if err != nil {
		t.Fatalf("FindByID(1) error: %v", err)
	}
	if !strings.HasSuffix(path, ".md") {
		t.Errorf("FindByID should return .md file, got %q", path)
	}
}

// --- ReadAll error path ---

func TestReadAll_MalformedFile(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "Good task", "backlog")
	// Write a malformed .md file.
	if err := os.WriteFile(filepath.Join(dir, "002-broken.md"), []byte("not frontmatter"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadAll(dir)
	if err == nil {
		t.Fatal("expected error from ReadAll with malformed file")
	}
	if !strings.Contains(err.Error(), "002-broken.md") {
		t.Errorf("error = %v, want to mention broken file", err)
	}
}

func TestReadAll_ReadDirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.ReadDir on a file path does not reliably error on Windows")
	}
	// Use a file path instead of a directory to trigger ReadDir error.
	tmpFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadAll(tmpFile)
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
}

// --- ReadAllLenient ---

func TestReadAllLenient_NonexistentDir(t *testing.T) {
	tasks, warnings, err := ReadAllLenient("/nonexistent/path")
	if err != nil {
		t.Fatalf("ReadAllLenient() error: %v", err)
	}
	if tasks != nil {
		t.Errorf("tasks = %v, want nil", tasks)
	}
	if warnings != nil {
		t.Errorf("warnings = %v, want nil", warnings)
	}
}

func TestReadAllLenient_ReadDirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.ReadDir on a file path does not reliably error on Windows")
	}
	// Use a file path instead of a directory.
	tmpFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := ReadAllLenient(tmpFile)
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
}
