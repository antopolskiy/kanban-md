package tui

import (
	"slices"

	"github.com/charmbracelet/x/ansi"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// propertyCardFields replaces only the configurable scalar part of card details.
// Default priority styling is unchanged.
func propertyCardFields(t *task.Task, cfg *config.Config, width int) []string {
	var details []string
	fields := cfg.CardFields()
	remaining := max(1, width)
	for i, field := range fields {
		if remaining < 1 {
			break
		}
		quota := max(1, (remaining-len(fields)+i+1)/(len(fields)-i))
		if key, selected, err := property.SelectorKey(field); selected && err == nil {
			for _, token := range output.TaskPropertyTokens(t, []string{key}, quota) {
				details = append(details, dimStyle.Render(token))
				remaining -= ansi.StringWidth(token) + 1
			}
			continue
		}
		switch field {
		case "priority":
			style, ok := priorityStyles[t.Priority]
			if !ok {
				style = dimStyle
			}
			value := t.Priority
			if len(fields) > 1 {
				value = ansi.Truncate(value, quota, "...")
			}
			details = append(details, style.Render(value))
			remaining -= ansi.StringWidth(value) + 1
		case "status":
			value := ansi.Truncate(t.Status, quota, "...")
			details = append(details, dimStyle.Render(value))
			remaining -= ansi.StringWidth(value) + 1
		case "class":
			value := ansi.Truncate(t.Class, quota, "...")
			details = append(details, dimStyle.Render(value))
			remaining -= ansi.StringWidth(value) + 1
		}
	}
	return details
}

// propertyDetailLines exposes configured human fields without JSON selection.
func propertyDetailLines(t *task.Task, cfg *config.Config) []string {
	return output.TaskPropertyTokens(t, propertyViewKeys(cfg), 0)
}

// propertyViewKeys explains configured card fields and direct-child order in
// human details, independently of explicit JSON selection.
func propertyViewKeys(cfg *config.Config) []string {
	var keys []string
	for _, selector := range append(cfg.CardFields(), cfg.Children.DetailSort) {
		if key, selected, err := property.SelectorKey(selector); selected && err == nil && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys
}
