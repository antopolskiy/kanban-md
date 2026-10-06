package output

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ValidTableColumns is the ordered set of column names accepted by TableOptions.
var ValidTableColumns = []string{"id", "status", "priority", "title", "claimed", "tags", "due"}

// TableOptions controls column selection and title width for TaskTableWithProperties.
// Zero value means "use defaults": all columns, standard title width cap.
type TableOptions struct {
	// Columns lists column names to display in the specified order.
	// nil or empty means all columns in default order.
	Columns []string
	// TitleWidth is the maximum display-cell width for the title column.
	// 0 means use the default cap.
	TitleWidth int
}

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
	TaskTableWithProperties(w, tasks, nil, TableOptions{})
}

// TaskTableWithProperties adds explicitly requested properties to task rows
// and applies column selection and title-width constraints from opts.
// tableColWidths holds the computed display widths for each fixed column.
type tableColWidths struct {
	id, status, priority, title, claimed, tags, due int
}

const (
	tableDefaultMaxTitle = 50
	tableDefaultMaxTags  = 30
	tableColPad          = 2

	// Minimum column widths (header label width + padding).
	colMinID       = 4
	colMinStatus   = 8
	colMinPriority = 10
	colMinTitle    = 5
	colMinClaimed  = 9
	colMinTags     = 6
	colMinDue      = 12
)

func computeColWidths(tasks []*task.Task, cols []string, maxTitle int) tableColWidths {
	w := tableColWidths{id: colMinID, status: colMinStatus, priority: colMinPriority, title: colMinTitle, claimed: colMinClaimed, tags: colMinTags, due: colMinDue}
	for _, t := range tasks {
		if slices.Contains(cols, "id") {
			w.id = max(w.id, len(strconv.Itoa(t.ID))+tableColPad)
		}
		if slices.Contains(cols, "status") {
			w.status = max(w.status, ansi.StringWidth(t.Status)+tableColPad)
		}
		if slices.Contains(cols, "priority") {
			w.priority = max(w.priority, ansi.StringWidth(t.Priority)+tableColPad)
		}
		if slices.Contains(cols, "title") {
			w.title = max(w.title, min(ansi.StringWidth(t.Title)+tableColPad, maxTitle))
		}
		if slices.Contains(cols, "claimed") {
			w.claimed = max(w.claimed, ansi.StringWidth(claimDisplay(t))+tableColPad)
		}
		if slices.Contains(cols, "tags") {
			w.tags = max(w.tags, min(ansi.StringWidth(strings.Join(t.Tags, ","))+tableColPad, tableDefaultMaxTags))
		}
	}
	return w
}

func buildHeaderParts(cols []string, w tableColWidths) []string {
	headers := map[string]string{
		"id":       fmt.Sprintf("%-*s", w.id, "ID"),
		"status":   fmt.Sprintf("%-*s", w.status, "STATUS"),
		"priority": fmt.Sprintf("%-*s", w.priority, "PRIORITY"),
		"title":    fmt.Sprintf("%-*s", w.title, "TITLE"),
		"claimed":  fmt.Sprintf("%-*s", w.claimed, "CLAIMED"),
		"tags":     fmt.Sprintf("%-*s", w.tags, "TAGS"),
		"due":      fmt.Sprintf("%-*s", w.due, "DUE"),
	}
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		if h, ok := headers[col]; ok {
			parts = append(parts, h)
		}
	}
	return parts
}

func buildRowParts(t *task.Task, cols []string, w tableColWidths, title, claim, tags, due string, hasExtraKeys bool) []string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		switch col {
		case "id":
			parts = append(parts, fmt.Sprintf("%-*d", w.id, t.ID))
		case "status":
			parts = append(parts, padRight(styledValue(t.Status, statusStyles), w.status))
		case "priority":
			parts = append(parts, padRight(styledValue(t.Priority, priorityStyles), w.priority))
		case "title":
			parts = append(parts, padRight(title, w.title))
		case "claimed":
			parts = append(parts, padRight(claim, w.claimed))
		case "tags":
			parts = append(parts, padRight(tags, w.tags))
		case "due":
			if hasExtraKeys {
				parts = append(parts, padRight(due, w.due))
			} else {
				parts = append(parts, due)
			}
		}
	}
	return parts
}

// TaskTableWithProperties adds explicitly requested properties to task rows
// and applies column selection and title-width constraints from opts.
func TaskTableWithProperties(w io.Writer, tasks []*task.Task, keys []string, opts TableOptions) {
	if len(tasks) == 0 {
		fmt.Fprintln(os.Stderr, "No tasks found.")
		return
	}

	cols := opts.Columns
	if len(cols) == 0 {
		cols = ValidTableColumns
	}

	maxTitle := tableDefaultMaxTitle
	if opts.TitleWidth > 0 {
		maxTitle = opts.TitleWidth
	}

	cw := computeColWidths(tasks, cols, maxTitle)

	// Print header.
	propertyWidths := tablePropertyWidths(tasks, keys)
	header := strings.Join(buildHeaderParts(cols, cw), " ")
	header += tablePropertySuffix(keys, propertyWidths)
	fmt.Fprintln(w, headerStyle.Render(strings.TrimRight(header, " ")))

	// Print rows.
	for _, t := range tasks {
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
		// Truncate title using display-cell width to handle wide Unicode and ANSI.
		title := ansi.Truncate(t.Title, maxTitle-tableColPad, "...")

		row := strings.Join(buildRowParts(t, cols, cw, title, claim, tags, due, len(keys) > 0), " ")
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
