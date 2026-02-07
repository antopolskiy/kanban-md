package task

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

const fileMode = 0o600

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

	return os.WriteFile(path, buf.Bytes(), fileMode)
}

// splitFrontmatter splits a markdown file into YAML frontmatter and body.
// The file must start with "---\n". Returns frontmatter bytes and body string.
func splitFrontmatter(data []byte) ([]byte, string, error) {
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return nil, "", errors.New("file does not start with YAML frontmatter (---)")
	}

	// Find the closing ---.
	rest := content[4:] // skip opening ---\n
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		// Check if file ends with \n---\n or \n--- at EOF.
		closingLen := len("---")
		if strings.HasSuffix(rest, "\n---") {
			idx = len(rest) - closingLen
		} else {
			return nil, "", errors.New("unclosed frontmatter (missing closing ---)")
		}
	}

	fm := rest[:idx]
	body := ""
	closingEnd := idx + len("\n---\n")
	if closingEnd < len(rest) {
		body = strings.TrimLeft(rest[closingEnd:], "\n")
	}

	return []byte(fm), body, nil
}
