// Package board provides board-level operations on task collections.
package board

import (
	"strings"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// FilterOptions defines which tasks to include.
type FilterOptions struct {
	Statuses        []string
	ExcludeStatuses []string // statuses to exclude from results
	Priorities      []string
	Assignee        string
	Tag             string
	Search          string        // case-insensitive substring match across title, body, and tags
	Blocked         *bool         // nil=no filter, true=only blocked, false=only not-blocked
	ParentID        *int          // nil=no filter, non-nil=only tasks with this parent
	Unclaimed       bool          // only unclaimed or expired-claim tasks
	ClaimedBy       string        // filter to specific claimant
	ClaimTimeout    time.Duration // claim expiration for unclaimed filter
	Class           string        // filter by class of service
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
	if !matchesCoreFilter(t, opts) {
		return false
	}
	return matchesExtendedFilter(t, opts)
}

func matchesCoreFilter(t *task.Task, opts FilterOptions) bool {
	if !matchesStatus(t.Status, opts.Statuses, opts.ExcludeStatuses) {
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

func matchesStatus(status string, include, exclude []string) bool {
	if len(include) > 0 && !containsStr(include, status) {
		return false
	}
	if len(exclude) > 0 && containsStr(exclude, status) {
		return false
	}
	return true
}

// matchesSearch performs case-insensitive substring matching across title, body, and tags.
func matchesSearch(t *task.Task, query string) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(t.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(t.Body), q) {
		return true
	}
	for _, tag := range t.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}

func matchesExtendedFilter(t *task.Task, opts FilterOptions) bool {
	if opts.Search != "" && !matchesSearch(t, opts.Search) {
		return false
	}
	if opts.Unclaimed && !IsUnclaimed(t, opts.ClaimTimeout) {
		return false
	}
	if opts.ClaimedBy != "" && t.ClaimedBy != opts.ClaimedBy {
		return false
	}
	if opts.Class != "" && t.Class != opts.Class {
		return false
	}
	return true
}

// IsUnclaimed returns true if the task has no active claim (unclaimed or expired).
func IsUnclaimed(t *task.Task, timeout time.Duration) bool {
	if t.ClaimedBy == "" {
		return true
	}
	if timeout > 0 && t.ClaimedAt != nil {
		return time.Since(*t.ClaimedAt) > timeout
	}
	return false
}

// FilterUnblocked returns tasks whose dependencies are all at a terminal status.
// Tasks with no dependencies are always included.
func FilterUnblocked(tasks []*task.Task, cfg *config.Config) []*task.Task {
	return FilterUnblockedWithLookup(tasks, tasks, cfg)
}

// FilterUnblockedWithLookup returns tasks from candidates whose dependencies
// are all at a terminal status. lookupTasks is used to build the status map
// for dependency resolution and may include tasks not in candidates (e.g.
// archived tasks needed for dep lookups).
func FilterUnblockedWithLookup(candidates, lookupTasks []*task.Task, cfg *config.Config) []*task.Task {
	// Build a map of task ID → status for dependency lookups.
	statusByID := make(map[int]string, len(lookupTasks))
	for _, t := range lookupTasks {
		statusByID[t.ID] = t.Status
	}

	var result []*task.Task
	for _, t := range candidates {
		if allDepsSatisfied(t.DependsOn, statusByID, cfg) {
			result = append(result, t)
		}
	}
	return result
}

func allDepsSatisfied(deps []int, statusByID map[int]string, cfg *config.Config) bool {
	for _, depID := range deps {
		s, ok := statusByID[depID]
		if !ok {
			// Missing dependency IDs can occur after legacy hard-deletes.
			// Treat as satisfied so dependents are recoverable via edit/cleanup.
			continue
		}
		if !cfg.IsTerminalStatus(s) {
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
