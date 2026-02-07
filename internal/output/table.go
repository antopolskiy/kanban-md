package output

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/antopolskiy/kanban-md/internal/task"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// DisableColor strips all styling from table output.
func DisableColor() {
	headerStyle = lipgloss.NewStyle()
	dimStyle = lipgloss.NewStyle()
}

// TaskTable renders a list of tasks as a formatted table.
func TaskTable(tasks []*task.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(os.Stderr, "No tasks found.")
		return
	}

	// Calculate column widths.
	const pad = 2
	idW, statusW, prioW, titleW, assignW, dueW := 4, 8, 10, 5, 10, 12
	for _, t := range tasks {
		idW = max(idW, len(strconv.Itoa(t.ID))+pad)
		statusW = max(statusW, len(t.Status)+pad)
		prioW = max(prioW, len(t.Priority)+pad)
		titleW = max(titleW, min(len(t.Title)+pad, 50)) //nolint:mnd // max title column width
		assignW = max(assignW, len(t.Assignee)+pad)
	}

	// Print header.
	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
		idW, "ID", statusW, "STATUS", prioW, "PRIORITY",
		titleW, "TITLE", assignW, "ASSIGNEE", dueW, "DUE")
	fmt.Fprintln(os.Stdout, headerStyle.Render(header))

	// Print rows.
	for _, t := range tasks {
		title := t.Title
		const maxTitle = 48
		if len(title) > maxTitle {
			title = title[:maxTitle-3] + "..."
		}
		assignee := t.Assignee
		if assignee == "" {
			assignee = dimStyle.Render("--")
		}
		due := "--"
		if t.Due != nil {
			due = t.Due.String()
		} else {
			due = dimStyle.Render(due)
		}

		fmt.Fprintf(os.Stdout, "%-*d %-*s %-*s %-*s %-*s %-*s\n",
			idW, t.ID, statusW, t.Status, prioW, t.Priority,
			titleW, title, assignW, assignee, dueW, due)
	}
}

// TaskDetail renders a single task with full detail.
func TaskDetail(t *task.Task) {
	titleLine := fmt.Sprintf("Task #%d: %s", t.ID, t.Title)
	fmt.Fprintln(os.Stdout, lipgloss.NewStyle().Bold(true).Render(titleLine))
	fmt.Fprintln(os.Stdout, strings.Repeat("─", len(titleLine)))

	printField("Status", t.Status)
	printField("Priority", t.Priority)
	printField("Assignee", stringOrDash(t.Assignee))
	if len(t.Tags) > 0 {
		printField("Tags", strings.Join(t.Tags, ", "))
	} else {
		printField("Tags", dimStyle.Render("--"))
	}
	if t.Due != nil {
		printField("Due", t.Due.String())
	} else {
		printField("Due", dimStyle.Render("--"))
	}
	printField("Estimate", stringOrDash(t.Estimate))
	printField("Created", t.Created.Format("2006-01-02 15:04"))
	printField("Updated", t.Updated.Format("2006-01-02 15:04"))
	if t.Started != nil {
		printField("Started", t.Started.Format("2006-01-02 15:04"))
	}
	if t.Completed != nil {
		printField("Completed", t.Completed.Format("2006-01-02 15:04"))
		printField("Lead time", FormatDuration(t.Completed.Sub(t.Created)))
		if t.Started != nil {
			printField("Cycle time", FormatDuration(t.Completed.Sub(*t.Started)))
		}
	}

	if t.Body != "" {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, t.Body)
	}
}

// Messagef prints a simple formatted message line.
func Messagef(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

func printField(label, value string) {
	fmt.Fprintf(os.Stdout, "  %-12s %s\n", label+":", value)
}

// FormatDuration renders a duration as human-readable "Xd Yh" or "Xh Ym".
func FormatDuration(d time.Duration) string {
	const hoursPerDay = 24
	days := int(d.Hours()) / hoursPerDay
	hours := int(d.Hours()) % hoursPerDay
	if days > 0 {
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h"
	}
	minutes := int(d.Minutes()) % 60 //nolint:mnd // 60 minutes per hour
	return strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m"
}

func stringOrDash(s string) string {
	if s == "" {
		return dimStyle.Render("--")
	}
	return s
}
