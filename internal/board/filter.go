// Package board provides board-level operations on task collections.
package board

import "github.com/antopolskiy/kanban-md/internal/task"

// FilterOptions defines which tasks to include.
type FilterOptions struct {
	Statuses   []string
	Priorities []string
	Assignee   string
	Tag        string
	Blocked    *bool // nil=no filter, true=only blocked, false=only not-blocked
	ParentID   *int  // nil=no filter, non-nil=only tasks with this parent
}

// Filter returns tasks matching all specified criteria (AND logic).
func Filter(tasks []*task.Task, opts FilterOptions) []*task.Task {
	var result []*task.Task
	for _, t := range tasks {
		if matchesFilter(t, opts) {
			result = append(result, t)
		}
	}
	return result
}

func matchesFilter(t *task.Task, opts FilterOptions) bool {
	if len(opts.Statuses) > 0 && !containsStr(opts.Statuses, t.Status) {
		return false
	}
	if len(opts.Priorities) > 0 && !containsStr(opts.Priorities, t.Priority) {
		return false
	}
	if opts.Assignee != "" && t.Assignee != opts.Assignee {
		return false
	}
	if opts.Tag != "" && !containsStr(t.Tags, opts.Tag) {
		return false
	}
	if opts.Blocked != nil && t.Blocked != *opts.Blocked {
		return false
	}
	if opts.ParentID != nil && (t.Parent == nil || *t.Parent != *opts.ParentID) {
		return false
	}
	return true
}

// FilterUnblocked returns tasks whose dependencies are all at a terminal status.
// Tasks with no dependencies are always included. The terminalStatus parameter
// is typically the last status in the board's configured statuses.
func FilterUnblocked(tasks []*task.Task, terminalStatus string) []*task.Task {
	// Build a map of task ID → status for dependency lookups.
	statusByID := make(map[int]string, len(tasks))
	for _, t := range tasks {
		statusByID[t.ID] = t.Status
	}

	var result []*task.Task
	for _, t := range tasks {
		if allDepsSatisfied(t.DependsOn, statusByID, terminalStatus) {
			result = append(result, t)
		}
	}
	return result
}

func allDepsSatisfied(deps []int, statusByID map[int]string, terminalStatus string) bool {
	for _, depID := range deps {
		s, ok := statusByID[depID]
		if !ok {
			// Dependency not found (deleted?) — treat as unsatisfied.
			return false
		}
		if s != terminalStatus {
			return false
		}
	}
	return true
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
