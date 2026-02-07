package board

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendLogCreatesFile(t *testing.T) {
	dir := t.TempDir()
	entry := LogEntry{
		Timestamp: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Action:    "create",
		TaskID:    1,
		Detail:    "New task",
	}

	if err := AppendLog(dir, entry); err != nil {
		t.Fatalf("AppendLog: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "activity.jsonl")) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}

	var got LogEntry
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Action != "create" {
		t.Errorf("Action = %q, want %q", got.Action, "create")
	}
	if got.TaskID != 1 {
		t.Errorf("TaskID = %d, want 1", got.TaskID)
	}
}

func TestAppendLogAppends(t *testing.T) {
	dir := t.TempDir()

	for i := 1; i <= 3; i++ {
		entry := LogEntry{
			Timestamp: time.Now(),
			Action:    "move",
			TaskID:    i,
			Detail:    "moved",
		}
		if err := AppendLog(dir, entry); err != nil {
			t.Fatalf("AppendLog entry %d: %v", i, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "activity.jsonl")) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
}

func TestReadLogEmpty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ReadLog(dir, LogFilterOptions{})
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestReadLogNoFile(t *testing.T) {
	dir := t.TempDir()
	entries, err := ReadLog(dir, LogFilterOptions{})
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if entries != nil {
		t.Errorf("entries = %v, want nil", entries)
	}
}

func TestReadLogFilterSince(t *testing.T) {
	dir := t.TempDir()

	old := LogEntry{
		Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Action:    "create",
		TaskID:    1,
		Detail:    "old",
	}
	recent := LogEntry{
		Timestamp: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		Action:    "move",
		TaskID:    2,
		Detail:    "recent",
	}
	mustAppend(t, dir, old)
	mustAppend(t, dir, recent)

	since := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	entries, err := ReadLog(dir, LogFilterOptions{Since: since})
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].TaskID != 2 {
		t.Errorf("TaskID = %d, want 2", entries[0].TaskID)
	}
}

func TestReadLogFilterAction(t *testing.T) {
	dir := t.TempDir()

	mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "create", TaskID: 1, Detail: "a"})
	mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "move", TaskID: 2, Detail: "b"})
	mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "create", TaskID: 3, Detail: "c"})

	entries, err := ReadLog(dir, LogFilterOptions{Action: "create"})
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
}

func TestReadLogFilterTaskID(t *testing.T) {
	dir := t.TempDir()

	mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "create", TaskID: 1, Detail: "a"})
	mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "move", TaskID: 1, Detail: "b"})
	mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "create", TaskID: 2, Detail: "c"})

	entries, err := ReadLog(dir, LogFilterOptions{TaskID: 1})
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
}

func TestReadLogLimit(t *testing.T) {
	dir := t.TempDir()

	for i := 1; i <= 5; i++ {
		mustAppend(t, dir, LogEntry{Timestamp: time.Now(), Action: "create", TaskID: i, Detail: "x"})
	}

	entries, err := ReadLog(dir, LogFilterOptions{Limit: 2})
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	// Should return the last 2 entries.
	if entries[0].TaskID != 4 {
		t.Errorf("entries[0].TaskID = %d, want 4", entries[0].TaskID)
	}
	if entries[1].TaskID != 5 {
		t.Errorf("entries[1].TaskID = %d, want 5", entries[1].TaskID)
	}
}

func TestReadLogCombinedFilters(t *testing.T) {
	dir := t.TempDir()

	mustAppend(t, dir, LogEntry{
		Timestamp: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Action:    "create",
		TaskID:    1,
		Detail:    "a",
	})
	mustAppend(t, dir, LogEntry{
		Timestamp: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		Action:    "move",
		TaskID:    1,
		Detail:    "b",
	})
	mustAppend(t, dir, LogEntry{
		Timestamp: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		Action:    "create",
		TaskID:    2,
		Detail:    "c",
	})

	entries, err := ReadLog(dir, LogFilterOptions{
		Since:  time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC),
		Action: "move",
		TaskID: 1,
	})
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Detail != "b" {
		t.Errorf("Detail = %q, want %q", entries[0].Detail, "b")
	}
}

func mustAppend(t *testing.T, dir string, entry LogEntry) {
	t.Helper()
	if err := AppendLog(dir, entry); err != nil {
		t.Fatalf("AppendLog: %v", err)
	}
}
