package output

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// captureStdout redirects os.Stdout to a pipe and returns the captured output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	const captureSize = 64 * 1024
	buf := make([]byte, captureSize)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestTaskTableColumnAlignment(t *testing.T) {
	// Force ANSI color output even in non-TTY (test) environments.
	// This is critical to catch the bug where %-*s counts ANSI bytes as width.
	oldHeader, oldDim := headerStyle, dimStyle
	t.Cleanup(func() {
		headerStyle = oldHeader
		dimStyle = oldDim
	})
	lipgloss.SetColorProfile(termenv.ANSI256)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	now := time.Now()
	due := date.New(2025, 6, 15)

	tasks := []*task.Task{
		{
			ID: 1, Title: "Task with all fields", Status: "in-progress",
			Priority: "high", Assignee: "alice", Tags: []string{"feature"},
			Due: &due, Created: now, Updated: now,
		},
		{
			ID: 2, Title: "Task with empty fields", Status: "backlog",
			Priority: "medium", Assignee: "", Tags: nil,
			Due: nil, Created: now, Updated: now,
		},
		{
			ID: 3, Title: "Another task", Status: "todo",
			Priority: "low", Assignee: "bob", Tags: []string{"bug", "urgent"},
			Due: &due, Created: now, Updated: now,
		},
	}

	output := captureStdout(t, func() { TaskTable(tasks) })
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	const expectedMinLines = 4 // header + 3 data rows
	if len(lines) < expectedMinLines {
		t.Fatalf("expected at least 4 lines, got %d:\n%s", len(lines), output)
	}

	// Measure visible widths of all rows. The key test: rows with "--" placeholders
	// (which contain ANSI escape codes) should NOT be wider than rows with real data,
	// because the padding should be based on visible width, not byte length.
	//
	// Row 1 has all fields populated (no ANSI in data columns).
	// Row 2 has empty assignee, tags, and due (all rendered with dimStyle → ANSI codes).
	// If %-*s counts ANSI bytes as width, row 2 will be visibly wider than row 1.
	row1Width := lipgloss.Width(lines[1])
	row2Width := lipgloss.Width(lines[2])
	row3Width := lipgloss.Width(lines[3])

	const maxDrift = 3 // allow tiny rounding differences
	if abs(row1Width-row2Width) > maxDrift {
		t.Errorf("column misalignment: row 1 visible width = %d, row 2 visible width = %d (drift > %d)\nrow1: %s\nrow2: %s",
			row1Width, row2Width, maxDrift, lines[1], lines[2])
	}
	if abs(row1Width-row3Width) > maxDrift {
		t.Errorf("column misalignment: row 1 visible width = %d, row 3 visible width = %d (drift > %d)\nrow1: %s\nrow3: %s",
			row1Width, row3Width, maxDrift, lines[1], lines[3])
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestFormatDuration_Days(t *testing.T) {
	d := 50 * time.Hour
	got := FormatDuration(d)
	if got != "2d 2h" {
		t.Errorf("FormatDuration(50h) = %q, want %q", got, "2d 2h")
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	d := 3*time.Hour + 30*time.Minute
	got := FormatDuration(d)
	if got != "3h 30m" {
		t.Errorf("FormatDuration(3h30m) = %q, want %q", got, "3h 30m")
	}
}

func TestFormatDuration_Zero(t *testing.T) {
	got := FormatDuration(0)
	if got != "0h 0m" {
		t.Errorf("FormatDuration(0) = %q, want %q", got, "0h 0m")
	}
}

func TestFormatDuration_ExactDays(t *testing.T) {
	d := 48 * time.Hour
	got := FormatDuration(d)
	if got != "2d 0h" {
		t.Errorf("FormatDuration(48h) = %q, want %q", got, "2d 0h")
	}
}
