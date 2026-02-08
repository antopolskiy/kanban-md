package board

import (
	"sort"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const (
	fieldPriority = "priority"
	fieldStatus   = "status"
	classStandard = "standard"
)

// GroupedSummary holds tasks grouped by a field.
type GroupedSummary struct {
	Groups []GroupSummary `json:"groups"`
}

// GroupSummary is one group within a grouped view.
type GroupSummary struct {
	Key      string          `json:"key"`
	Statuses []StatusSummary `json:"statuses"`
	Total    int             `json:"total"`
}

// GroupBy groups tasks by the specified field and returns summaries per group.
func GroupBy(tasks []*task.Task, field string, cfg *config.Config) GroupedSummary {
	groups := make(map[string][]*task.Task)

	for _, t := range tasks {
		keys := extractGroupKeys(t, field)
		for _, key := range keys {
			groups[key] = append(groups[key], t)
		}
	}

	sortedKeys := sortGroupKeys(groups, field, cfg)

	result := GroupedSummary{
		Groups: make([]GroupSummary, 0, len(sortedKeys)),
	}
	for _, key := range sortedKeys {
		groupTasks := groups[key]
		statuses := groupStatusSummary(groupTasks, cfg)
		result.Groups = append(result.Groups, GroupSummary{
			Key:      key,
			Statuses: statuses,
			Total:    len(groupTasks),
		})
	}
	return result
}

func extractGroupKeys(t *task.Task, field string) []string {
	switch field {
	case "assignee":
		if t.Assignee == "" {
			return []string{"(unassigned)"}
		}
		return []string{t.Assignee}
	case "tag":
		if len(t.Tags) == 0 {
			return []string{"(untagged)"}
		}
		return t.Tags
	case "class":
		cls := t.Class
		if cls == "" {
			cls = classStandard
		}
		return []string{cls}
	case fieldPriority:
		return []string{t.Priority}
	case fieldStatus:
		return []string{t.Status}
	default:
		return []string{"(all)"}
	}
}

func sortGroupKeys(groups map[string][]*task.Task, field string, cfg *config.Config) []string {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}

	switch field {
	case fieldStatus:
		sort.SliceStable(keys, func(i, j int) bool {
			return cfg.StatusIndex(keys[i]) < cfg.StatusIndex(keys[j])
		})
	case fieldPriority:
		sort.SliceStable(keys, func(i, j int) bool {
			return cfg.PriorityIndex(keys[i]) < cfg.PriorityIndex(keys[j])
		})
	case "class":
		sort.SliceStable(keys, func(i, j int) bool {
			return cfg.ClassIndex(keys[i]) < cfg.ClassIndex(keys[j])
		})
	default:
		sort.Strings(keys)
	}
	return keys
}

func groupStatusSummary(tasks []*task.Task, cfg *config.Config) []StatusSummary {
	counts := make(map[string]int)
	for _, t := range tasks {
		counts[t.Status]++
	}
	statuses := make([]StatusSummary, 0, len(cfg.Statuses))
	for _, s := range cfg.Statuses {
		statuses = append(statuses, StatusSummary{
			Status:   s,
			Count:    counts[s],
			WIPLimit: cfg.WIPLimit(s),
		})
	}
	return statuses
}

// ValidGroupByFields returns the list of valid --group-by field names.
func ValidGroupByFields() []string {
	return []string{"assignee", "tag", "class", "priority", "status"}
}
