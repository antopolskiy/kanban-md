package board

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	logFileName = "activity.jsonl"
	logFileMode = 0o600
)

// LogEntry represents a single activity log entry.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	TaskID    int       `json:"task_id"`
	Detail    string    `json:"detail"`
}

// LogFilterOptions controls how log entries are filtered.
type LogFilterOptions struct {
	Since  time.Time
	Limit  int
	Action string
	TaskID int
}

// AppendLog appends a log entry to the activity log file.
func AppendLog(kanbanDir string, entry LogEntry) error {
	path := filepath.Join(kanbanDir, logFileName)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFileMode) //nolint:gosec // log path from trusted kanban dir
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling log entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing log entry: %w", err)
	}

	return nil
}

// ReadLog reads and filters log entries from the activity log file.
func ReadLog(kanbanDir string, opts LogFilterOptions) ([]LogEntry, error) {
	path := filepath.Join(kanbanDir, logFileName)

	f, err := os.Open(path) //nolint:gosec // log path from trusted kanban dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}

		if !matchesLogFilter(entry, opts) {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading log file: %w", err)
	}

	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[len(entries)-opts.Limit:]
	}

	return entries, nil
}

func matchesLogFilter(entry LogEntry, opts LogFilterOptions) bool {
	if !opts.Since.IsZero() && entry.Timestamp.Before(opts.Since) {
		return false
	}
	if opts.Action != "" && entry.Action != opts.Action {
		return false
	}
	if opts.TaskID > 0 && entry.TaskID != opts.TaskID {
		return false
	}
	return true
}
