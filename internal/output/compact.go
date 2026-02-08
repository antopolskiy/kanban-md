package output

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// TaskCompact renders a list of tasks in one-line-per-record compact format.
func TaskCompact(w io.Writer, tasks []*task.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(os.Stderr, "No tasks found.")
		return
	}

	for _, t := range tasks {
		fmt.Fprintln(w, formatTaskLine(t))
	}
}

// TaskDetailCompact renders a single task with detail in compact format.
func TaskDetailCompact(w io.Writer, t *task.Task) {
	line := formatTaskLine(t)
	if t.Estimate != "" {
		line += " est:" + t.Estimate
	}
	fmt.Fprintln(w, line)

	// Timestamps line.
	ts := "  created:" + t.Created.Format("2006-01-02") +
		" updated:" + t.Updated.Format("2006-01-02")
	if t.Started != nil {
		ts += " started:" + t.Started.Format("2006-01-02")
	}
	if t.Completed != nil {
		ts += " completed:" + t.Completed.Format("2006-01-02")
	}
	fmt.Fprintln(w, ts)

	if t.Body != "" {
		for _, bodyLine := range strings.Split(t.Body, "\n") {
			fmt.Fprintln(w, "  "+bodyLine)
		}
	}
}

// OverviewCompact renders a board summary in compact format.
func OverviewCompact(w io.Writer, s board.Overview) {
	fmt.Fprintf(w, "%s (%d tasks)\n", s.BoardName, s.TotalTasks)

	for _, ss := range s.Statuses {
		line := "  " + ss.Status + ": " + strconv.Itoa(ss.Count)
		if ss.WIPLimit > 0 {
			line += "/" + strconv.Itoa(ss.WIPLimit)
		}
		var annotations []string
		if ss.Blocked > 0 {
			annotations = append(annotations, strconv.Itoa(ss.Blocked)+" blocked")
		}
		if ss.Overdue > 0 {
			annotations = append(annotations, strconv.Itoa(ss.Overdue)+" overdue")
		}
		if len(annotations) > 0 {
			line += " (" + strings.Join(annotations, ", ") + ")"
		}
		fmt.Fprintln(w, line)
	}

	if len(s.Priorities) > 0 {
		parts := make([]string, 0, len(s.Priorities))
		for _, pc := range s.Priorities {
			parts = append(parts, pc.Priority+"="+strconv.Itoa(pc.Count))
		}
		fmt.Fprintln(w, "Priority: "+strings.Join(parts, " "))
	}
}

// MetricsCompact renders flow metrics in compact format.
func MetricsCompact(w io.Writer, m board.Metrics) {
	parts := []string{
		"Throughput: " + strconv.Itoa(m.Throughput7d) + "/7d " + strconv.Itoa(m.Throughput30d) + "/30d",
		"Lead: " + compactDuration(m.AvgLeadTimeHours),
		"Cycle: " + compactDuration(m.AvgCycleTimeHours),
		"Efficiency: " + formatOptionalPercent(m.FlowEfficiency),
	}
	fmt.Fprintln(w, strings.Join(parts, " | "))

	for _, a := range m.AgingItems {
		title := a.Title
		const maxTitle = 60
		if len(title) > maxTitle {
			title = title[:maxTitle-3] + "..."
		}
		fmt.Fprintf(w, "Aging: #%d [%s] %s (%s)\n",
			a.ID, a.Status, title, FormatDuration(time.Duration(a.AgeHours*float64(time.Hour))))
	}
}

// ActivityLogCompact renders activity log entries in compact format.
func ActivityLogCompact(w io.Writer, entries []board.LogEntry) {
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No activity log entries found.")
		return
	}

	for _, e := range entries {
		fmt.Fprintf(w, "%s %s #%d %s\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Action, e.TaskID, e.Detail)
	}
}

// formatTaskLine builds the one-line representation of a task.
func formatTaskLine(t *task.Task) string {
	line := "#" + strconv.Itoa(t.ID) + " [" + t.Status + "/" + t.Priority + "] " + t.Title

	if t.Assignee != "" {
		line += " @" + t.Assignee
	}
	if len(t.Tags) > 0 {
		line += " (" + strings.Join(t.Tags, ", ") + ")"
	}
	if t.Due != nil {
		line += " due:" + t.Due.String()
	}

	return line
}

// compactDuration formats an optional hours value as a duration string.
func compactDuration(h *float64) string {
	if h == nil {
		return "--"
	}
	return FormatDuration(time.Duration(*h * float64(time.Hour)))
}
