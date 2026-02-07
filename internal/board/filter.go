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
