package tui

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/charmbracelet/lipgloss"

	"github.com/antopolskiy/kanban-md/internal/task"
)

const unknownDepth = -1

// taskHierarchy is read-only, derived view state rebuilt after loading tasks.
type taskHierarchy struct {
	byID         map[int]*task.Task
	depths       map[int]int
	hasHierarchy bool
}

func newTaskHierarchy(tasks []*task.Task) *taskHierarchy {
	h := &taskHierarchy{
		byID:   make(map[int]*task.Task, len(tasks)),
		depths: make(map[int]int, len(tasks)),
	}
	for _, t := range tasks {
		h.byID[t.ID] = t
		h.hasHierarchy = h.hasHierarchy || t.Parent != nil
	}
	for _, t := range tasks {
		h.resolveDepth(t.ID)
	}
	return h
}

// resolveDepth walks iteratively so deep hand-edited chains cannot exhaust the
// stack. A completed depth is memoized once for every task in the walk.
func (h *taskHierarchy) resolveDepth(id int) {
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
		if current.Parent == nil || *current.Parent == id || h.byID[*current.Parent] == nil {
			// Missing and self parents remain effective roots defensively.
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
	tasks  []*task.Task // oldest known ancestor first; excludes the current task
	marker string
}

func (h *taskHierarchy) ancestors(current *task.Task) ancestorPath {
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

func (b *Board) hierarchyIndex() *taskHierarchy {
	if b.hierarchy != nil {
		return b.hierarchy
	}
	// Small in-memory models used by package tests may not call loadTasks.
	tasks := b.allTasks
	if tasks == nil {
		tasks = b.unfilteredTasks
	}
	return newTaskHierarchy(tasks)
}

func (b *Board) depthChoices() []int {
	seen := make(map[int]bool)
	h := b.hierarchyIndex()
	for _, t := range b.unfilteredTasks {
		seen[h.depths[t.ID]] = true
	}
	var choices []int
	for depth := range seen {
		if depth != unknownDepth {
			choices = append(choices, depth)
		}
	}
	sort.Ints(choices)
	if seen[unknownDepth] {
		choices = append(choices, unknownDepth)
	}
	return choices
}

func (b *Board) cycleDepth() {
	choices := b.depthChoices()
	next := 0
	if b.depthFilter != nil {
		for i, depth := range choices {
			if depth == *b.depthFilter {
				next = i + 1
				break
			}
		}
	}
	b.depthFilter = nil
	if next < len(choices) {
		depth := choices[next]
		b.depthFilter = &depth
	}
	b.reloadKeepingSelection()
}

func (b *Board) reconcileDepthFilter() {
	if b.depthFilter == nil {
		return
	}
	for _, depth := range b.depthChoices() {
		if depth == *b.depthFilter {
			return
		}
	}
	b.depthFilter = nil
}

func depthLabel(depth int) string {
	if depth == unknownDepth {
		return "?"
	}
	return strconv.Itoa(depth)
}

func (b *Board) levelState() string {
	if b.depthFilter != nil {
		return "level[" + depthLabel(*b.depthFilter) + "]"
	}
	return "level[all]"
}

func (b *Board) showLevelState() bool {
	return b.depthFilter != nil || b.hierarchy != nil && b.hierarchy.hasHierarchy
}

func (b *Board) cardTitleLines(t *task.Task, cardWidth int) []string {
	idText := "#" + strconv.Itoa(t.ID)
	if b.hierarchy != nil && b.hierarchy.hasHierarchy {
		idText += " L" + depthLabel(b.hierarchy.depths[t.ID])
	}
	idLen := lipgloss.Width(idText)
	firstWidth := max(cardWidth-idLen-1, 1)
	switch {
	case idLen >= cardWidth:
		return []string{dimStyle.Render(truncate(idText, cardWidth))}
	case b.cfg.TitleLines() == 1:
		return []string{dimStyle.Render(idText) + " " + truncate(t.Title, firstWidth)}
	default:
		wrapped := wrapTitle2(t.Title, firstWidth, cardWidth, b.cfg.TitleLines())
		wrapped[0] = dimStyle.Render(idText) + " " + wrapped[0]
		return wrapped
	}
}
