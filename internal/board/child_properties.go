package board

import (
	"sort"

	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func sortChildProperties(children []*task.Task, selector string) {
	key, selected, err := property.SelectorKey(selector)
	values := make(map[*task.Task]property.Scalar, len(children))
	if selected && err == nil {
		for _, child := range children {
			value, state := child.PropertyScalar(key)
			if state == task.PropertySupported && value.Kind() == property.Number {
				values[child] = value
			}
		}
	}
	sort.Slice(children, func(i, j int) bool {
		a, hasA := values[children[i]]
		b, hasB := values[children[j]]
		if hasA != hasB {
			return hasA
		}
		if hasA {
			if comparison, _ := a.CompareNumber(b); comparison != 0 {
				return comparison < 0
			}
		}
		return children[i].ID < children[j].ID
	})
}

func childPropertyValues(t *task.Task, keys []string) map[string]property.Scalar {
	values := map[string]property.Scalar{}
	for _, key := range keys {
		value, state := t.PropertyScalar(key)
		if state != task.PropertyMissing {
			values[key] = value
		}
	}
	return values
}

// SelectChildProperties adds human-view fields without changing default child JSON.
func SelectChildProperties(summary *ChildSummary, tasks []*task.Task, keys []string) {
	lookup := map[int]*task.Task{}
	for _, t := range tasks {
		lookup[t.ID] = t
	}
	for i := range summary.Children {
		child := &summary.Children[i]
		for _, key := range keys {
			found := false
			for _, existing := range child.PropertyKeys {
				if existing == key {
					found = true
					break
				}
			}
			if !found {
				child.PropertyKeys = append(child.PropertyKeys, key)
			}
		}
		if t := lookup[child.ID]; t != nil {
			child.Properties = childPropertyValues(t, child.PropertyKeys)
		}
	}
}
