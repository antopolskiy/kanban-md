package board

import (
	"sort"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// PickOptions controls how the pick algorithm selects a task.
type PickOptions struct {
	Statuses     []string      // status columns to pick from (empty = all non-terminal)
	ClaimTimeout time.Duration // claim expiration for filtering
	Tags         []string      // optional tag filter (OR logic: task must have at least one)
}

// Pick finds the highest-priority unclaimed, unblocked task matching criteria.
// Returns nil if no task matches.
func Pick(cfg *config.Config, tasks []*task.Task, opts PickOptions) *task.Task {
	candidates := pickCandidates(cfg, tasks, opts)
	candidates = filterPickDeps(cfg, tasks, candidates)

	if len(candidates) == 0 {
		return nil
	}

	sortPickCandidates(candidates, cfg)
	return candidates[0]
}

// pickCandidates filters tasks by status, claim, block, and tag.
func pickCandidates(cfg *config.Config, tasks []*task.Task, opts PickOptions) []*task.Task {
	statuses := opts.Statuses
	if len(statuses) == 0 {
		statuses = cfg.ActiveStatuses()
	}

	var candidates []*task.Task
	for _, t := range tasks {
		if !containsStr(statuses, t.Status) {
			continue
		}
		if !IsUnclaimed(t, opts.ClaimTimeout) {
			continue
		}
		if t.Blocked {
			continue
		}
		if len(opts.Tags) > 0 && !hasAnyTag(t.Tags, opts.Tags) {
			continue
		}
		candidates = append(candidates, t)
	}
	return candidates
}

// filterPickDeps removes tasks with unmet dependencies using the full task set.
func filterPickDeps(cfg *config.Config, allTasks, candidates []*task.Task) []*task.Task {
	if len(cfg.StatusNames()) == 0 {
		return candidates
	}
	statusByID := make(map[int]string, len(allTasks))
	for _, t := range allTasks {
		statusByID[t.ID] = t.Status
	}
	var unblocked []*task.Task
	for _, c := range candidates {
		if allDepsSatisfied(c.DependsOn, statusByID, cfg) {
			unblocked = append(unblocked, c)
		}
	}
	return unblocked
}

// sortPickCandidates sorts by class priority then task priority.
func sortPickCandidates(candidates []*task.Task, cfg *config.Config) {
	sort.SliceStable(candidates, func(i, j int) bool {
		ci := classOrder(candidates[i], cfg)
		cj := classOrder(candidates[j], cfg)
		if ci != cj {
			return ci < cj
		}
		if candidates[i].Class == "fixed-date" && candidates[j].Class == "fixed-date" {
			if compareDue(candidates[i], candidates[j]) {
				return true
			}
			if compareDue(candidates[j], candidates[i]) {
				return false
			}
		}
		return cfg.PriorityIndex(candidates[i].Priority) > cfg.PriorityIndex(candidates[j].Priority)
	})
}

// hasAnyTag returns true if the task has at least one of the given tags.
func hasAnyTag(taskTags, filterTags []string) bool {
	for _, ft := range filterTags {
		if containsStr(taskTags, ft) {
			return true
		}
	}
	return false
}

// classOrder returns a sort key for a task's class. Lower is higher priority.
func classOrder(t *task.Task, cfg *config.Config) int {
	if t.Class == "" {
		return cfg.ClassIndex(classStandard)
	}
	idx := cfg.ClassIndex(t.Class)
	if idx < 0 {
		return cfg.ClassIndex(classStandard)
	}
	return idx
}
