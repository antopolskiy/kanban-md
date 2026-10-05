package board

import (
	"sort"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

type scalarGroup struct {
	value property.Scalar
	tasks []*task.Task
}

const unclassifiedPropertyGroup = "unclassified"

func groupByProperty(tasks []*task.Task, key, selector string, cfg *config.Config) GroupedSummary {
	groups := map[string]*scalarGroup{}
	for _, t := range tasks {
		value, state := t.PropertyScalar(key)
		identity := unclassifiedPropertyGroup
		if state == task.PropertySupported {
			identity = value.Identity()
		}
		if groups[identity] == nil {
			groups[identity] = &scalarGroup{value: value}
		}
		groups[identity].tasks = append(groups[identity].tasks, t)
	}
	order := map[string]int{}
	for i, value := range cfg.GroupOrders[selector] {
		order[value.Identity()] = i
	}
	keys := make([]string, 0, len(groups))
	for identity := range groups {
		keys = append(keys, identity)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == unclassifiedPropertyGroup || keys[j] == unclassifiedPropertyGroup {
			return keys[i] != unclassifiedPropertyGroup
		}
		a, hasA := order[keys[i]]
		b, hasB := order[keys[j]]
		if hasA != hasB {
			return hasA
		}
		if hasA {
			return a < b
		}
		return compareGroupScalars(groups[keys[i]].value, groups[keys[j]].value) < 0
	})
	result := GroupedSummary{Groups: make([]GroupSummary, 0, len(keys))}
	for _, identity := range keys {
		group := groups[identity]
		label := "(unclassified)"
		if identity != unclassifiedPropertyGroup {
			label = group.value.DisplayLiteral()
		}
		result.Groups = append(result.Groups, GroupSummary{Key: label, Statuses: groupStatusSummary(group.tasks, cfg), Total: len(group.tasks)})
	}
	return result
}

func compareGroupScalars(a, b property.Scalar) int {
	return a.Compare(b)
}
