package board

import (
	"fmt"
	"time"

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
// Uses lenient parsing: malformed task files are skipped and returned as warnings.
func List(cfg *config.Config, opts ListOptions) ([]*task.Task, []task.ReadWarning, error) {
	tasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return nil, nil, err
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

	return tasks, warnings, nil
}

// FindDependents returns human-readable messages for tasks that reference the
// given ID as a parent or dependency. Used to warn before deleting a task.
func FindDependents(tasksDir string, id int) []string {
	allTasks, _, err := task.ReadAllLenient(tasksDir)
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

// StatusSummary holds metrics for a single status column.
type StatusSummary struct {
	Status   string `json:"status"`
	Count    int    `json:"count"`
	WIPLimit int    `json:"wip_limit,omitempty"`
	Blocked  int    `json:"blocked"`
	Overdue  int    `json:"overdue"`
}

// PriorityCount holds a count for a priority level.
type PriorityCount struct {
	Priority string `json:"priority"`
	Count    int    `json:"count"`
}

// ClassCount holds a count for a class of service.
type ClassCount struct {
	Class string `json:"class"`
	Count int    `json:"count"`
}

// Overview is the aggregate board overview.
type Overview struct {
	BoardName  string          `json:"board_name"`
	TotalTasks int             `json:"total_tasks"`
	Statuses   []StatusSummary `json:"statuses"`
	Priorities []PriorityCount `json:"priorities"`
	Classes    []ClassCount    `json:"classes,omitempty"`
}

// Summary computes a board summary from all tasks.
func Summary(cfg *config.Config, tasks []*task.Task) Overview {
	now := time.Now()

	statusMap := make(map[string]*StatusSummary, len(cfg.Statuses))
	for _, s := range cfg.Statuses {
		statusMap[s] = &StatusSummary{
			Status:   s,
			WIPLimit: cfg.WIPLimit(s),
		}
	}

	prioMap := make(map[string]int, len(cfg.Priorities))
	classMap := make(map[string]int)

	for _, t := range tasks {
		if ss, ok := statusMap[t.Status]; ok {
			ss.Count++
			if t.Blocked {
				ss.Blocked++
			}
			if t.Due != nil && t.Due.Before(now) && !cfg.IsTerminalStatus(t.Status) {
				ss.Overdue++
			}
		}
		prioMap[t.Priority]++
		cls := t.Class
		if cls == "" {
			cls = classStandard
		}
		classMap[cls]++
	}

	statuses := make([]StatusSummary, 0, len(cfg.Statuses))
	for _, s := range cfg.Statuses {
		statuses = append(statuses, *statusMap[s])
	}

	priorities := make([]PriorityCount, 0, len(cfg.Priorities))
	for _, p := range cfg.Priorities {
		priorities = append(priorities, PriorityCount{Priority: p, Count: prioMap[p]})
	}

	var classes []ClassCount
	if len(cfg.Classes) > 0 {
		classes = make([]ClassCount, 0, len(cfg.Classes))
		for _, cl := range cfg.Classes {
			classes = append(classes, ClassCount{Class: cl.Name, Count: classMap[cl.Name]})
		}
	}

	return Overview{
		BoardName:  cfg.Board.Name,
		TotalTasks: len(tasks),
		Statuses:   statuses,
		Priorities: priorities,
		Classes:    classes,
	}
}

// CountByStatus returns the number of tasks in each status.
func CountByStatus(tasks []*task.Task) map[string]int {
	counts := make(map[string]int)
	for _, t := range tasks {
		counts[t.Status]++
	}
	return counts
}
