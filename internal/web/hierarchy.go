package web

import (
	"fmt"
	"strconv"

	"github.com/antopolskiy/kanban-md/internal/task"
)

const unknownDepth = -1

// hierarchy copies the TUI depth rules: roots are 0, children increase by one,
// and a cycle is unknown. Archived ancestors still count.
type hierarchy struct {
	byID   map[int]*task.Task
	depths map[int]int
	has    bool
}

func newHierarchy(tasks []*task.Task) *hierarchy {
	h := &hierarchy{
		byID:   make(map[int]*task.Task, len(tasks)),
		depths: make(map[int]int, len(tasks)),
	}
	for _, t := range tasks {
		h.byID[t.ID] = t
		h.has = h.has || t.Parent != nil
	}
	for _, t := range tasks {
		h.resolve(t.ID)
	}
	return h
}

func (h *hierarchy) resolve(id int) {
	if _, known := h.depths[id]; known {
		return
	}
	var path []int
	seen := make(map[int]bool)
	depth := unknownDepth
	for {
		if d, known := h.depths[id]; known {
			depth = d
			break
		}
		if seen[id] {
			break
		}
		seen[id] = true
		path = append(path, id)
		current := h.byID[id]
		if current == nil || current.Parent == nil || *current.Parent == id || h.byID[*current.Parent] == nil {
			depth = 0
			h.depths[id] = depth
			path = path[:len(path)-1]
			break
		}
		id = *current.Parent
	}
	for i := len(path) - 1; i >= 0; i-- {
		if depth != unknownDepth {
			depth++
		}
		h.depths[path[i]] = depth
	}
}

type ancestorPath struct {
	tasks  []*task.Task
	marker string
}

func (h *hierarchy) ancestors(current *task.Task) ancestorPath {
	var result ancestorPath
	seen := map[int]bool{current.ID: true}
	for current.Parent != nil {
		id := *current.Parent
		switch {
		case id == current.ID:
			result.marker = fmt.Sprintf("self parent #%d", id)
		case seen[id]:
			result.marker = fmt.Sprintf("cycle to #%d", id)
		case h.byID[id] == nil:
			result.marker = fmt.Sprintf("missing parent #%d", id)
		default:
			seen[id] = true
			current = h.byID[id]
			result.tasks = append(result.tasks, current)
			continue
		}
		break
	}
	for i, j := 0, len(result.tasks)-1; i < j; i, j = i+1, j-1 {
		result.tasks[i], result.tasks[j] = result.tasks[j], result.tasks[i]
	}
	return result
}

func depthLabel(depth int) string {
	if depth == unknownDepth {
		return "?"
	}
	return strconv.Itoa(depth)
}

func levelMatch(depth int, level string) bool {
	switch level {
	case "", levelAll:
		return true
	case levelUnknown:
		return depth == unknownDepth
	default:
		want, err := strconv.Atoi(level)
		if err != nil {
			return true
		}
		return depth == want
	}
}
