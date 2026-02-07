package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// ErrNotFound indicates a task was not found.
var ErrNotFound = errors.New("task not found")

// FindByID scans the tasks directory for a file matching the given ID.
// Returns the full path to the task file.
func FindByID(tasksDir string, id int) (string, error) {
	pattern := fmt.Sprintf(`^0*%d-.*\.md$`, id)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("compiling pattern: %w", err)
	}

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return "", fmt.Errorf("reading tasks directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && re.MatchString(entry.Name()) {
			return filepath.Join(tasksDir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("%w: #%d", ErrNotFound, id)
}

// ReadAll reads all task files from the given directory.
func ReadAll(tasksDir string) ([]*Task, error) {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading tasks directory: %w", err)
	}

	var tasks []*Task
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		path := filepath.Join(tasksDir, entry.Name())
		t, err := Read(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

// ExtractIDFromFilename extracts the numeric ID from a task filename.
func ExtractIDFromFilename(filename string) (int, error) {
	re := regexp.MustCompile(`^(\d+)-`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) < 2 { //nolint:mnd // regex capture group
		return 0, fmt.Errorf("cannot extract ID from filename %q", filename)
	}
	return strconv.Atoi(matches[1])
}
