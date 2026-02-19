package board

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// Sentinel markers for in-place file updates.
const (
	contextBeginMarker = "<!-- BEGIN kanban-md context -->"
	contextEndMarker   = "<!-- END kanban-md context -->"
)

// ContextOptions controls which sections to include.
type ContextOptions struct {
	Sections []string // empty = all sections
	Days     int      // lookback for recently completed (default 7)
}

// ContextData holds all context information for rendering.
type ContextData struct {
	BoardName string           `json:"board_name"`
	Summary   ContextSummary   `json:"summary"`
	Sections  []ContextSection `json:"sections"`
}

// ContextSummary holds aggregate board statistics.
type ContextSummary struct {
	TotalTasks int    `json:"total_tasks"`
	Active     int    `json:"active"`
	Blocked    int    `json:"blocked"`
	Overdue    int    `json:"overdue"`
	WIPWarning string `json:"wip_warning,omitempty"`
}

// ContextSection represents a named group of context items.
type ContextSection struct {
	Name  string        `json:"name"`
	Items []ContextItem `json:"items"`
}

// ContextItem represents a single task in the context output.
type ContextItem struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Assignee string `json:"assignee,omitempty"`
	Note     string `json:"note,omitempty"`
}

// sectionName constants for filtering and display.
const (
	sectionInProgress        = "in-progress"
	sectionBlocked           = "blocked"
	sectionOverdue           = "overdue"
	sectionRecentlyCompleted = "recently-completed"
)

// allSectionNames returns the ordered list of section names.
func allSectionNames() []string {
	return []string{
		sectionInProgress,
		sectionBlocked,
		sectionOverdue,
		sectionRecentlyCompleted,
	}
}

const defaultDays = 7

// GenerateContext builds context data from config and tasks.
func GenerateContext(cfg *config.Config, tasks []*task.Task, opts ContextOptions, now time.Time) ContextData {
	days := opts.Days
	if days <= 0 {
		days = defaultDays
	}

	data := ContextData{
		BoardName: cfg.Board.Name,
		Summary:   computeSummary(cfg, tasks, now),
	}

	// Build sections.
	wantedSections := allSectionNames()
	if len(opts.Sections) > 0 {
		wantedSections = opts.Sections
	}

	for _, name := range wantedSections {
		items := buildSection(cfg, tasks, name, now, days)
		if len(items) > 0 {
			data.Sections = append(data.Sections, ContextSection{Name: name, Items: items})
		}
	}

	return data
}

func computeSummary(cfg *config.Config, tasks []*task.Task, now time.Time) ContextSummary {
	var active, blocked, overdue int
	for _, t := range tasks {
		// Active = not in first status (backlog) and not in terminal status (done).
		if !isFirstStatus(cfg, t.Status) && !cfg.IsTerminalStatus(t.Status) {
			active++
		}
		if t.Blocked {
			blocked++
		}
		if t.Due != nil && t.Due.Before(now) && !cfg.IsTerminalStatus(t.Status) {
			overdue++
		}
	}

	summary := ContextSummary{
		TotalTasks: len(tasks),
		Active:     active,
		Blocked:    blocked,
		Overdue:    overdue,
	}

	// Check WIP warnings.
	statusCounts := CountByStatus(tasks)
	var wipWarnings []string
	for _, s := range cfg.StatusNames() {
		limit := cfg.WIPLimit(s)
		if limit > 0 && statusCounts[s] >= limit {
			wipWarnings = append(wipWarnings,
				s+" ("+strconv.Itoa(statusCounts[s])+"/"+strconv.Itoa(limit)+")")
		}
	}
	if len(wipWarnings) > 0 {
		summary.WIPWarning = "WIP limit reached: " + strings.Join(wipWarnings, ", ")
	}

	return summary
}

func buildSection(cfg *config.Config, tasks []*task.Task, name string, now time.Time, days int) []ContextItem {
	switch name {
	case sectionInProgress:
		return buildInProgressSection(cfg, tasks)
	case sectionBlocked:
		return buildBlockedSection(cfg, tasks)
	case sectionOverdue:
		return buildOverdueSection(cfg, tasks, now)
	case sectionRecentlyCompleted:
		return buildRecentlyCompletedSection(cfg, tasks, now, days)
	default:
		return nil
	}
}

func buildInProgressSection(cfg *config.Config, tasks []*task.Task) []ContextItem {
	var items []ContextItem
	for _, t := range tasks {
		if !isFirstStatus(cfg, t.Status) && !cfg.IsTerminalStatus(t.Status) && !t.Blocked {
			items = append(items, taskToItem(t, ""))
		}
	}
	sortByPriority(items, cfg)
	return items
}

func buildBlockedSection(_ *config.Config, tasks []*task.Task) []ContextItem {
	var items []ContextItem
	for _, t := range tasks {
		if t.Blocked {
			items = append(items, taskToItem(t, t.BlockReason))
		}
	}
	return items
}

func buildOverdueSection(cfg *config.Config, tasks []*task.Task, now time.Time) []ContextItem {
	var items []ContextItem
	for _, t := range tasks {
		if t.Due != nil && t.Due.Before(now) && !cfg.IsTerminalStatus(t.Status) {
			items = append(items, taskToItem(t, "due "+t.Due.String()))
		}
	}
	return items
}

func buildRecentlyCompletedSection(cfg *config.Config, tasks []*task.Task, now time.Time, days int) []ContextItem {
	cutoff := now.AddDate(0, 0, -days)
	var items []ContextItem
	for _, t := range tasks {
		if cfg.IsTerminalStatus(t.Status) && t.Completed != nil && t.Completed.After(cutoff) {
			items = append(items, taskToItem(t, "completed "+t.Completed.Format("2006-01-02")))
		}
	}
	return items
}

func taskToItem(t *task.Task, note string) ContextItem {
	return ContextItem{
		ID:       t.ID,
		Title:    t.Title,
		Status:   t.Status,
		Priority: t.Priority,
		Assignee: t.Assignee,
		Note:     note,
	}
}

func sortByPriority(items []ContextItem, cfg *config.Config) {
	sort.Slice(items, func(i, j int) bool {
		return cfg.PriorityIndex(items[i].Priority) > cfg.PriorityIndex(items[j].Priority)
	})
}

func isFirstStatus(cfg *config.Config, status string) bool {
	names := cfg.StatusNames()
	return len(names) > 0 && names[0] == status
}

// RenderContextMarkdown renders context data as markdown wrapped in sentinel markers.
func RenderContextMarkdown(data ContextData) string {
	var b strings.Builder

	b.WriteString(contextBeginMarker)
	b.WriteString("\n")
	b.WriteString("## Board: ")
	b.WriteString(data.BoardName)
	b.WriteString("\n\n")

	// Summary.
	fmt.Fprintf(&b, "**%d tasks** | %d active | %d blocked | %d overdue\n",
		data.Summary.TotalTasks, data.Summary.Active,
		data.Summary.Blocked, data.Summary.Overdue)
	if data.Summary.WIPWarning != "" {
		b.WriteString("\n> ")
		b.WriteString(data.Summary.WIPWarning)
		b.WriteString("\n")
	}

	// Sections.
	for _, sec := range data.Sections {
		b.WriteString("\n### ")
		b.WriteString(sectionTitle(sec.Name))
		b.WriteString("\n\n")
		for _, item := range sec.Items {
			fmt.Fprintf(&b, "- **#%d** %s", item.ID, item.Title)
			parts := []string{item.Priority}
			if item.Assignee != "" {
				parts = append(parts, "@"+item.Assignee)
			}
			b.WriteString(" (")
			b.WriteString(strings.Join(parts, ", "))
			b.WriteString(")")
			if item.Note != "" {
				b.WriteString(" — ")
				b.WriteString(item.Note)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(contextEndMarker)
	b.WriteString("\n")
	return b.String()
}

func sectionTitle(name string) string {
	switch name {
	case sectionInProgress:
		return "In Progress"
	case sectionBlocked:
		return "Blocked"
	case sectionOverdue:
		return "Overdue"
	case sectionRecentlyCompleted:
		return "Recently Completed"
	default:
		return name
	}
}

// WriteContextToFile writes context content to a file, replacing existing
// sentinel-marked blocks or appending if none found.
func WriteContextToFile(path, content string) error {
	const fileMode = 0o600

	existing, err := os.ReadFile(path) //nolint:gosec // user-provided path
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte(content), fileMode)
		}
		return fmt.Errorf("reading file: %w", err)
	}

	text := string(existing)
	beginIdx := strings.Index(text, contextBeginMarker)
	endIdx := strings.Index(text, contextEndMarker)

	if beginIdx >= 0 && endIdx >= 0 {
		// Replace existing block (include the end marker and trailing newline).
		endOfBlock := endIdx + len(contextEndMarker)
		if endOfBlock < len(text) && text[endOfBlock] == '\n' {
			endOfBlock++
		}
		updated := text[:beginIdx] + content + text[endOfBlock:]
		return os.WriteFile(path, []byte(updated), fileMode)
	}

	// No markers found — append.
	separator := "\n"
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		separator = "\n\n"
	}
	updated := text + separator + content
	return os.WriteFile(path, []byte(updated), fileMode)
}
