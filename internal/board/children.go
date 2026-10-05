package board

import (
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// ChildTask is the stable, read-only child representation used by task-detail views.
type ChildTask struct {
	ID           int                        `json:"id"`
	Title        string                     `json:"title"`
	Status       string                     `json:"status"`
	PropertyKeys []string                   `json:"-"`
	Properties   map[string]property.Scalar `json:"-"`
}

// ParentTask is the resolved, read-only parent representation used by task-detail views.
type ParentTask struct {
	ID     int
	Title  string
	Status string
}

// ChildSummary contains direct children and their terminal-status roll-up.
type ChildSummary struct {
	Children []ChildTask
	Done     int
}

// Total returns the number of direct children in the summary.
func (s ChildSummary) Total() int {
	return len(s.Children)
}

// FindParent resolves a task's direct parent, including an archived parent.
// Missing and defensive self-references return nil so renderers can fall back
// to the parent ID stored on the task.
func FindParent(tasks []*task.Task, current *task.Task) *ParentTask {
	if current.Parent == nil || *current.Parent == current.ID {
		return nil
	}
	for _, candidate := range tasks {
		if candidate.ID == *current.Parent {
			return &ParentTask{ID: candidate.ID, Title: candidate.Title, Status: candidate.Status}
		}
	}
	return nil
}

// SummarizeChildren returns direct children in configured numeric-property order,
// or ascending task-ID order by default. Unsuitable values and ties use task ID.
// Archived children are omitted unless includeArchived is true. Self-references
// are ignored defensively; normal CLI mutations reject them before writing.
func SummarizeChildren(
	tasks []*task.Task,
	parentID int,
	cfg *config.Config,
	includeArchived bool,
) ChildSummary {
	summary := ChildSummary{Children: make([]ChildTask, 0)}
	var candidates []*task.Task
	for _, candidate := range tasks {
		if candidate.ID == parentID || candidate.Parent == nil || *candidate.Parent != parentID {
			continue
		}
		if !includeArchived && cfg.IsArchivedStatus(candidate.Status) {
			continue
		}

		candidates = append(candidates, candidate)
	}
	sortChildProperties(candidates, cfg.Children.DetailSort)
	for _, candidate := range candidates {
		child := ChildTask{
			ID:     candidate.ID,
			Title:  candidate.Title,
			Status: candidate.Status,
		}
		if key, selected, err := property.SelectorKey(cfg.Children.DetailSort); selected && err == nil {
			child.PropertyKeys = []string{key}
			child.Properties = childPropertyValues(candidate, child.PropertyKeys)
		}
		summary.Children = append(summary.Children, child)
		if cfg.IsTerminalStatus(candidate.Status) {
			summary.Done++
		}
	}

	return summary
}
