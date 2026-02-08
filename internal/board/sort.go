package board

import (
	"sort"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// Sort sorts tasks by the given field. For status and priority,
// the config order is used (not alphabetical).
func Sort(tasks []*task.Task, field string, reverse bool, cfg *config.Config) {
	sort.SliceStable(tasks, func(i, j int) bool {
		less := compareTasks(tasks[i], tasks[j], field, cfg)
		if reverse {
			return !less
		}
		return less
	})
}

func compareTasks(a, b *task.Task, field string, cfg *config.Config) bool {
	switch field {
	case "id":
		return a.ID < b.ID
	case fieldStatus:
		return cfg.StatusIndex(a.Status) < cfg.StatusIndex(b.Status)
	case fieldPriority:
		return cfg.PriorityIndex(a.Priority) < cfg.PriorityIndex(b.Priority)
	case "created":
		return a.Created.Before(b.Created)
	case "updated":
		return a.Updated.Before(b.Updated)
	case "due":
		return compareDue(a, b)
	default:
		return a.ID < b.ID
	}
}

func compareDue(a, b *task.Task) bool {
	if a.Due == nil && b.Due == nil {
		return false
	}
	if a.Due == nil {
		return false // nil sorts last
	}
	if b.Due == nil {
		return true
	}
	return a.Due.Before(b.Due.Time)
}
