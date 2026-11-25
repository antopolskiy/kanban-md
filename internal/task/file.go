package task

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

const fileMode = 0o600

// fileModeReadOnly is used for claimed task files to prevent external modification.
const fileModeReadOnly = 0o444

// Read parses a task file and returns the Task with body populated.
func Read(path string) (*Task, error) {
	data, err := os.ReadFile(path) //nolint:gosec // task path from trusted source
	if err != nil {
		return nil, fmt.Errorf("reading task file: %w", err)
	}

	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	var t Task
	if err := yaml.Unmarshal(fm, &t); err != nil {
		return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}
	if err := validateRequiredFields(&t); err != nil {
		return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}

	t.Body = body
	t.File = path

	return &t, nil
}

// Write serializes a task to a markdown file with YAML frontmatter.
func Write(path string, t *Task) error {
	fm, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshaling frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fm)
	buf.WriteString("---\n")
	if t.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(t.Body)
		if !strings.HasSuffix(t.Body, "\n") {
			buf.WriteString("\n")
		}
	}

	// If the file exists and is read-only (claimed), make it writable before writing.
	unlockForWrite(path)

	if err := os.WriteFile(path, buf.Bytes(), fileMode); err != nil {
		return err
	}

	// If the task is claimed, lock the file to prevent external modifications.
	if t.ClaimedBy != "" {
		lockFile(path)
	}

	return nil
}

// splitFrontmatter splits a markdown file into YAML frontmatter and body.
// The file must start with a "---" line. Returns frontmatter bytes and body string.
func splitFrontmatter(data []byte) ([]byte, string, error) {
	content := string(data)

	openingLen := 0
	switch {
	case strings.HasPrefix(content, "---\r\n"):
		openingLen = len("---\r\n")
	case strings.HasPrefix(content, "---\n"):
		openingLen = len("---\n")
	default:
		return nil, "", errors.New("file does not start with YAML frontmatter (---)")
	}

	fmStart := openingLen
	for pos := openingLen; pos < len(content); {
		lineStart := pos
		lineEnd := len(content)
		next := len(content)
		if newline := strings.IndexByte(content[pos:], '\n'); newline >= 0 {
			lineEnd = pos + newline
			next = lineEnd + 1
		}

		line := strings.TrimSuffix(content[lineStart:lineEnd], "\r")
		if line == "---" {
			fmEnd := lineStart
			if fmEnd > fmStart && content[fmEnd-1] == '\n' {
				fmEnd--
				if fmEnd > fmStart && content[fmEnd-1] == '\r' {
					fmEnd--
				}
			}

			body := ""
			if next < len(content) {
				body = strings.TrimLeft(content[next:], "\r\n")
			}
			return []byte(content[fmStart:fmEnd]), body, nil
		}
		if next == len(content) {
			break
		}
		pos = next
	}

	return nil, "", errors.New("unclosed frontmatter (missing closing ---)")
}

// WriteAndRename writes the task to disk and renames the file if the title
// changed (so the filename slug stays in sync). Returns the new file path.
func WriteAndRename(path string, t *Task, oldTitle string) (string, error) {
	newPath := path
	if t.Title != oldTitle {
		slug := GenerateSlug(t.Title)
		filename := GenerateFilename(t.ID, slug)
		newPath = filepath.Join(filepath.Dir(path), filename)
	}

	if err := Write(newPath, t); err != nil {
		return "", fmt.Errorf("writing task: %w", err)
	}

	if newPath != path {
		if err := os.Remove(path); err != nil {
			return "", fmt.Errorf("removing old file: %w", err)
		}
	}
	return newPath, nil
}

// unlockForWrite makes a file writable if it exists and is read-only.
// Errors are silently ignored (best-effort for filesystems without Unix permissions).
func unlockForWrite(path string) {
	_ = os.Chmod(path, fileMode)
}

// lockFile makes a file read-only to prevent external modifications.
// Errors are silently ignored (best-effort for filesystems without Unix permissions).
func lockFile(path string) {
	_ = os.Chmod(path, fileModeReadOnly)
}

func validateRequiredFields(t *Task) error {
	if t.ID < 1 {
		return errors.New("missing required field: id")
	}
	if strings.TrimSpace(t.Title) == "" {
		return errors.New("missing required field: title")
	}
	if strings.TrimSpace(t.Status) == "" {
		return errors.New("missing required field: status")
	}
	return nil
}
