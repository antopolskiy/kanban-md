package board

import (
	"fmt"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ListOptions controls how tasks are listed.
type ListOptions struct {
	Filter    FilterOptions
	SortBy    string
	Reverse   bool
	Limit     int
	Unblocked bool // only tasks with all dependencies at terminal status
}

// List loads all tasks, applies filters and sorting.
func List(cfg *config.Config, opts ListOptions) ([]*task.Task, error) {
	tasks, err := task.ReadAll(cfg.TasksPath())
	if err != nil {
		return nil, err
	}

	tasks = Filter(tasks, opts.Filter)

	if opts.Unblocked && len(cfg.Statuses) > 0 {
		terminalStatus := cfg.Statuses[len(cfg.Statuses)-1]
		tasks = FilterUnblocked(tasks, terminalStatus)
	}

	sortField := opts.SortBy
	if sortField == "" {
		sortField = "id"
	}
	Sort(tasks, sortField, opts.Reverse, cfg)

	if opts.Limit > 0 && len(tasks) > opts.Limit {
		tasks = tasks[:opts.Limit]
	}

	return tasks, nil
}

// FindDependents returns human-readable messages for tasks that reference the
// given ID as a parent or dependency. Used to warn before deleting a task.
func FindDependents(tasksDir string, id int) []string {
	allTasks, err := task.ReadAll(tasksDir)
	if err != nil {
		return nil
	}

	var msgs []string
	for _, t := range allTasks {
		if t.Parent != nil && *t.Parent == id {
			msgs = append(msgs, fmt.Sprintf("task #%d (%s) has this as parent", t.ID, t.Title))
		}
		for _, dep := range t.DependsOn {
			if dep == id {
				msgs = append(msgs, fmt.Sprintf("task #%d (%s) depends on this task", t.ID, t.Title))
				break
			}
		}
	}
	return msgs
}

// CountByStatus returns the number of tasks in each status.
func CountByStatus(tasks []*task.Task) map[string]int {
	counts := make(map[string]int)
	for _, t := range tasks {
		counts[t.Status]++
	}
	return counts
}
