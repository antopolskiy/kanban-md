package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/antopolskiy/kanban-md/internal/clierr"
)

// idPrefixRe matches the numeric ID prefix of a task filename.
var idPrefixRe = regexp.MustCompile(`^(\d+)-`)

// FindByID scans the tasks directory for a file matching the given ID.
// Returns the full path to the task file.
func FindByID(tasksDir string, id int) (string, error) {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return "", fmt.Errorf("reading tasks directory: %w", err)
	}

	idStr := strconv.Itoa(id)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		// Strip leading zeros and check if the numeric prefix matches the ID.
		dash := strings.IndexByte(name, '-')
		if dash < 1 {
			continue
		}
		prefix := strings.TrimLeft(name[:dash], "0")
		if prefix == idStr {
			return filepath.Join(tasksDir, name), nil
		}
	}

	return "", clierr.Newf(clierr.TaskNotFound, "task not found: #%d", id).
		WithDetails(map[string]any{"id": id})
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
	matches := idPrefixRe.FindStringSubmatch(filename)
	if len(matches) < 2 { //nolint:mnd // regex capture group
		return 0, fmt.Errorf("cannot extract ID from filename %q", filename)
	}
	return strconv.Atoi(matches[1])
}
