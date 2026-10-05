package output

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Status colors aligned with TUI column-header palette.
	statusStyles = map[string]lipgloss.Style{
		"backlog":     lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		"todo":        lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		"in-progress": lipgloss.NewStyle().Foreground(lipgloss.Color("33")),
		"review":      lipgloss.NewStyle().Foreground(lipgloss.Color("62")),
		"done":        lipgloss.NewStyle().Foreground(lipgloss.Color("34")),
		"archived":    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	}

	// Priority colors matching TUI priority palette.
	priorityStyles = map[string]lipgloss.Style{
		"critical": lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		"high":     lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		"medium":   lipgloss.NewStyle().Foreground(lipgloss.Color("226")),
		"low":      lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
	}

	tagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
	claimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("44")).Bold(true)
)

// DisableColor strips all styling from table output.
func DisableColor() {
	headerStyle = lipgloss.NewStyle()
	dimStyle = lipgloss.NewStyle()
	statusStyles = map[string]lipgloss.Style{}
	priorityStyles = map[string]lipgloss.Style{}
	tagStyle = lipgloss.NewStyle()
	claimStyle = lipgloss.NewStyle()
}

// TaskTable renders a list of tasks as a formatted table.
func TaskTable(w io.Writer, tasks []*task.Task) {
	TaskTableWithProperties(w, tasks, nil)
}

// TaskTableWithProperties adds explicitly requested properties to task rows.
func TaskTableWithProperties(w io.Writer, tasks []*task.Task, keys []string) {
	if len(tasks) == 0 {
		fmt.Fprintln(os.Stderr, "No tasks found.")
		return
	}

	// Calculate column widths.
	const pad = 2
	idW, statusW, prioW, titleW, claimW, tagsW, dueW := 4, 8, 10, 5, 9, 6, 12
	for _, t := range tasks {
		idW = max(idW, len(strconv.Itoa(t.ID))+pad)
		statusW = max(statusW, len(t.Status)+pad)
		prioW = max(prioW, len(t.Priority)+pad)
		titleW = max(titleW, min(len(t.Title)+pad, 50)) //nolint:mnd // max title column width
		claimW = max(claimW, len(claimDisplay(t))+pad)
		tagsW = max(tagsW, min(len(strings.Join(t.Tags, ","))+pad, 30)) //nolint:mnd // max tags column width
	}

	// Print header.
	propertyWidths := tablePropertyWidths(tasks, keys)
	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		idW, "ID", statusW, "STATUS", prioW, "PRIORITY",
		titleW, "TITLE", claimW, "CLAIMED", tagsW, "TAGS", dueW, "DUE")
	header += tablePropertySuffix(keys, propertyWidths)
	fmt.Fprintln(w, headerStyle.Render(strings.TrimRight(header, " ")))

	// Print rows.
	for _, t := range tasks {
		title := t.Title
		const maxTitle = 48
		if len(title) > maxTitle {
			title = title[:maxTitle-3] + "..."
		}
		claim := claimDisplay(t)
		if claim == "" {
			claim = dimStyle.Render("--")
		} else {
			claim = claimStyle.Render(claim)
		}
		tags := strings.Join(t.Tags, ",")
		if tags == "" {
			tags = dimStyle.Render("--")
		} else {
			tags = tagStyle.Render(tags)
		}
		due := "--"
		if t.Due != nil {
			due = t.Due.String()
		} else {
			due = dimStyle.Render(due)
		}
		if len(keys) > 0 {
			due = padRight(due, dueW)
		}

		row := fmt.Sprintf("%-*d %s %s %s %s %s %s",
			idW, t.ID,
			padRight(styledValue(t.Status, statusStyles), statusW),
			padRight(styledValue(t.Priority, priorityStyles), prioW),
			padRight(title, titleW),
			padRight(claim, claimW),
			padRight(tags, tagsW),
			due)
		row += tablePropertySuffix(tablePropertyCells(t, keys), propertyWidths)
		fmt.Fprintln(w, strings.TrimRight(row, " "))
	}
}

// TaskDetail renders a single task with full detail.
func TaskDetail(w io.Writer, t *task.Task) {
	taskDetail(w, t, nil, board.ChildSummary{})
}

// TaskDetailWithChildren renders a single task and a read-only direct-child roll-up.
func TaskDetailWithChildren(w io.Writer, t *task.Task, children board.ChildSummary) {
	taskDetail(w, t, nil, children)
}

// TaskDetailWithRelations renders a task with its resolved direct parent and children.
func TaskDetailWithRelations(
	w io.Writer,
	t *task.Task,
	parent *board.ParentTask,
	children board.ChildSummary,
) {
	taskDetail(w, t, parent, children)
}

func taskDetail(w io.Writer, t *task.Task, parent *board.ParentTask, children board.ChildSummary) {
	TaskDetailWithProperties(w, t, parent, children, nil)
}

// TaskDetailWithProperties displays chosen values without selecting JSON output.
func TaskDetailWithProperties(w io.Writer, t *task.Task, parent *board.ParentTask, children board.ChildSummary, keys []string) {
	titleLine := fmt.Sprintf("Task #%d: %s", t.ID, t.Title)
	fmt.Fprintln(w, lipgloss.NewStyle().Bold(true).Render(titleLine))
	fmt.Fprintln(w, strings.Repeat("─", len(titleLine)))

	printField(w, "Status", styledValue(t.Status, statusStyles))
	printField(w, "Priority", styledValue(t.Priority, priorityStyles))
	for _, key := range keys {
		printField(w, key, strings.TrimPrefix(TaskPropertyTokens(t, []string{key}, 0)[0], key+"="))
	}
	if t.Class != "" {
		printField(w, "Class", t.Class)
	}
	printField(w, "Assignee", stringOrDash(t.Assignee))
	if len(t.Tags) > 0 {
		printField(w, "Tags", tagStyle.Render(strings.Join(t.Tags, ", ")))
	} else {
		printField(w, "Tags", dimStyle.Render("--"))
	}
	if t.Due != nil {
		printField(w, "Due", t.Due.String())
	} else {
		printField(w, "Due", dimStyle.Render("--"))
	}
	printField(w, "Estimate", stringOrDash(t.Estimate))
	printField(w, "Created", t.Created.Format("2006-01-02 15:04"))
	printField(w, "Updated", t.Updated.Format("2006-01-02 15:04"))
	if t.Started != nil {
		printField(w, "Started", t.Started.Format("2006-01-02 15:04"))
	}
	if t.Completed != nil {
		printField(w, "Completed", t.Completed.Format("2006-01-02 15:04"))
		printField(w, "Lead time", FormatDuration(t.Completed.Sub(t.Created)))
		if t.Started != nil {
			printField(w, "Cycle time", FormatDuration(t.Completed.Sub(*t.Started)))
		}
	}

	if t.ClaimedBy != "" {
		claimStr := claimStyle.Render(t.ClaimedBy)
		if t.ClaimedAt != nil {
			claimStr += " (since " + t.ClaimedAt.Format("2006-01-02 15:04") + ")"
		}
		printField(w, "Claimed by", claimStr)
	}

	if t.Parent != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, parentRelationLine(*t.Parent, parent))
	}

	if children.Total() > 0 {
		fmt.Fprintln(w)
		heading := fmt.Sprintf("Children (%d/%d done)", children.Done, children.Total())
		fmt.Fprintln(w, lipgloss.NewStyle().Bold(true).Render(heading))
		for i, child := range children.Children {
			branch := "├─"
			if i == len(children.Children)-1 {
				branch = "└─"
			}
			fmt.Fprintf(w, "%s #%d [%s] %s%s\n", branch, child.ID, child.Status, child.Title, childPropertySuffix(child))
		}
	}

	if t.Body != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.Body)
	}
}

func parentRelationLine(parentID int, parent *board.ParentTask) string {
	if parent == nil {
		return fmt.Sprintf("↑ Parent  #%d", parentID)
	}
	return fmt.Sprintf("↑ Parent  #%d [%s] %s", parent.ID, parent.Status, parent.Title)
}

// OverviewTable renders a board summary as a formatted dashboard.
func OverviewTable(w io.Writer, s board.Overview) {
	fmt.Fprintln(w, lipgloss.NewStyle().Bold(true).Render(s.BoardName))
	fmt.Fprintf(w, "Total: %d tasks\n\n", s.TotalTasks)

	header := fmt.Sprintf("%-16s %6s %8s %8s %8s", "STATUS", "COUNT", "WIP", "BLOCKED", "OVERDUE")
	fmt.Fprintln(w, headerStyle.Render(header))

	for _, ss := range s.Statuses {
		wip := dimStyle.Render("--")
		if ss.WIPLimit > 0 {
			wip = strconv.Itoa(ss.Count) + "/" + strconv.Itoa(ss.WIPLimit)
		}
		const statusColW = 16
		fmt.Fprintf(w, "%s %6d %s %8d %8d\n",
			padRight(styledValue(ss.Status, statusStyles), statusColW),
			ss.Count, padRight(wip, 8), ss.Blocked, ss.Overdue) //nolint:mnd // column width
	}

	fmt.Fprintln(w)
	prioHeader := fmt.Sprintf("%-16s %6s", "PRIORITY", "COUNT")
	fmt.Fprintln(w, headerStyle.Render(prioHeader))

	for _, pc := range s.Priorities {
		const prioColW = 16
		fmt.Fprintf(w, "%s %6d\n",
			padRight(styledValue(pc.Priority, priorityStyles), prioColW), pc.Count)
	}

	if len(s.Classes) > 0 {
		fmt.Fprintln(w)
		classHeader := fmt.Sprintf("%-16s %6s", "CLASS", "COUNT")
		fmt.Fprintln(w, headerStyle.Render(classHeader))
		for _, cc := range s.Classes {
			fmt.Fprintf(w, "%-16s %6d\n", cc.Class, cc.Count)
		}
	}
}

// MetricsTable renders flow metrics as a formatted dashboard.
func MetricsTable(w io.Writer, m board.Metrics) {
	fmt.Fprintln(w, lipgloss.NewStyle().Bold(true).Render("Flow Metrics"))
	fmt.Fprintln(w)

	printField(w, "Throughput 7d", strconv.Itoa(m.Throughput7d)+" tasks")
	printField(w, "Throughput 30d", strconv.Itoa(m.Throughput30d)+" tasks")
	printField(w, "Avg lead time", formatOptionalHours(m.AvgLeadTimeHours))
	printField(w, "Avg cycle time", formatOptionalHours(m.AvgCycleTimeHours))
	printField(w, "Flow efficiency", formatOptionalPercent(m.FlowEfficiency))

	if len(m.AgingItems) > 0 {
		fmt.Fprintln(w)
		agingHeader := fmt.Sprintf("%-6s %-16s %-40s %10s", "ID", "STATUS", "TITLE", "AGE")
		fmt.Fprintln(w, headerStyle.Render(agingHeader))
		for _, a := range m.AgingItems {
			title := a.Title
			const maxTitle = 38
			if len(title) > maxTitle {
				title = title[:maxTitle-3] + "..."
			}
			const agingStatusW = 16
			fmt.Fprintf(w, "%-6d %s %-40s %10s\n",
				a.ID, padRight(styledValue(a.Status, statusStyles), agingStatusW),
				title, FormatDuration(time.Duration(a.AgeHours*float64(time.Hour))))
		}
	}
}

func formatOptionalHours(h *float64) string {
	if h == nil {
		return dimStyle.Render("--")
	}
	return FormatDuration(time.Duration(*h * float64(time.Hour)))
}

func formatOptionalPercent(f *float64) string {
	if f == nil {
		return dimStyle.Render("--")
	}
	const percentMultiplier = 100
	return fmt.Sprintf("%.1f%%", *f*percentMultiplier)
}

// ActivityLogTable renders activity log entries as a formatted table.
func ActivityLogTable(w io.Writer, entries []board.LogEntry) {
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No activity log entries found.")
		return
	}

	header := fmt.Sprintf("%-20s %-10s %6s  %s", "TIMESTAMP", "ACTION", "TASK", "DETAIL")
	fmt.Fprintln(w, headerStyle.Render(header))

	for _, e := range entries {
		fmt.Fprintf(w, "%-20s %-10s %6d  %s\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Action, e.TaskID, e.Detail)
	}
}

// GroupedTable renders a grouped board view with per-group status breakdowns.
func GroupedTable(w io.Writer, gs board.GroupedSummary) {
	if len(gs.Groups) == 0 {
		fmt.Fprintln(os.Stderr, "No groups found.")
		return
	}

	for i, g := range gs.Groups {
		if i > 0 {
			fmt.Fprintln(w)
		}
		title := fmt.Sprintf("%s (%d tasks)", g.Key, g.Total)
		fmt.Fprintln(w, lipgloss.NewStyle().Bold(true).Render(title))

		for _, ss := range g.Statuses {
			if ss.Count == 0 {
				continue
			}
			const groupStatusW = 16
			fmt.Fprintf(w, "  %s %d\n",
				padRight(styledValue(ss.Status, statusStyles), groupStatusW), ss.Count)
		}
	}
}

// Messagef prints a simple formatted message line.
func Messagef(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format+"\n", args...)
}

func printField(w io.Writer, label, value string) {
	fmt.Fprintf(w, "  %-12s %s\n", label+":", value)
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

// padRight pads s with spaces to the given visible width, accounting for ANSI
// escape codes that are invisible but consume bytes.
func padRight(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func stringOrDash(s string) string {
	if s == "" {
		return dimStyle.Render("--")
	}
	return s
}

// claimDisplay returns "@agent" if the task is claimed, or "" otherwise.
func claimDisplay(t *task.Task) string {
	if t.ClaimedBy != "" {
		return "@" + t.ClaimedBy
	}
	return ""
}

// styledValue renders s using a matching style from the map, or returns s unchanged.
func styledValue(s string, styles map[string]lipgloss.Style) string {
	if st, ok := styles[s]; ok {
		return st.Render(s)
	}
	return s
}
