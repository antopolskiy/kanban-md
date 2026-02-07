package board

import (
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ListOptions controls how tasks are listed.
type ListOptions struct {
	Filter  FilterOptions
	SortBy  string
	Reverse bool
	Limit   int
}

// List loads all tasks, applies filters and sorting.
func List(cfg *config.Config, opts ListOptions) ([]*task.Task, error) {
	tasks, err := task.ReadAll(cfg.TasksPath())
	if err != nil {
		return nil, err
	}

	tasks = Filter(tasks, opts.Filter)

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
