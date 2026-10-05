package output

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// TaskViewOptions selects human fields independently of JSON projection.
type TaskViewOptions struct {
	CompactFields []string
	PropertyKeys  []string
}

const (
	propertyBadgeCells   = 24
	compactPropertyCells = 40
)

func selectedValues(t *task.Task, keys []string) map[string]property.Scalar {
	values := map[string]property.Scalar{}
	for _, key := range keys {
		value, state := t.PropertyScalar(key)
		if state != task.PropertyMissing {
			values[key] = value
		}
	}
	return values
}

// PropertyToken escapes a selected value before optional display-width truncation.
func PropertyToken(key string, value property.Scalar, present bool, width int) string {
	literal := "--"
	if present {
		literal = value.DisplayLiteral()
		if value.Kind() == property.Invalid {
			literal = "?"
		}
	}
	if width > 0 {
		literal = ansi.Truncate(literal, propertyBadgeCells, "...")
	}
	token := key + "=" + literal
	if width > 0 {
		token = ansi.Truncate(token, width, "...")
	}
	return token
}

// PropertyTokens renders selected literals in caller-selected key order.
func PropertyTokens(keys []string, values map[string]property.Scalar, width int) []string {
	tokens := make([]string, 0, len(keys))
	for _, key := range keys {
		value, present := values[key]
		tokens = append(tokens, PropertyToken(key, value, present, width))
	}
	return tokens
}

// TaskPropertyTokens is shared by compact/TUI selected rendering.
func TaskPropertyTokens(t *task.Task, keys []string, width int) []string {
	return PropertyTokens(keys, selectedValues(t, keys), width)
}

func compactChip(t *task.Task, fields []string) string {
	if fields == nil || slices.Equal(fields, []string{"status", "priority"}) {
		return "[" + t.Status + "/" + t.Priority + "]"
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if key, selected, err := property.SelectorKey(field); selected && err == nil {
			parts = append(parts, TaskPropertyTokens(t, []string{key}, compactPropertyCells)...)
			continue
		}
		switch field {
		case "status":
			parts = append(parts, "status:"+t.Status)
		case "priority":
			parts = append(parts, "priority:"+t.Priority)
		case "class":
			parts = append(parts, "class:"+t.Class)
		}
	}
	return "[" + strings.Join(parts, "/") + "]"
}

func extraPropertyTokens(t *task.Task, opts TaskViewOptions) []string {
	var keys []string
	for _, key := range opts.PropertyKeys {
		if !slices.Contains(opts.CompactFields, "property:"+key) {
			keys = append(keys, key)
		}
	}
	return TaskPropertyTokens(t, keys, compactPropertyCells)
}

func childPropertySuffix(child board.ChildTask) string {
	if len(child.PropertyKeys) == 0 {
		return ""
	}
	return " (" + strings.Join(PropertyTokens(child.PropertyKeys, child.Properties, 0), " ") + ")"
}

// SelectedProperties exports only supported explicitly named scalar values.
func SelectedProperties(t *task.Task, keys []string) (map[string]property.Scalar, []string) {
	values := map[string]property.Scalar{}
	var warnings []string
	for _, key := range keys {
		value, state := t.PropertyScalar(key)
		if state == task.PropertySupported {
			values[key] = value
		}
		if state == task.PropertyUnsupported {
			warnings = append(warnings, fmt.Sprintf("task #%d property %q is not a supported scalar; omitted from properties", t.ID, key))
		}
	}
	return values, warnings
}

func tablePropertyCells(t *task.Task, keys []string) []string {
	cells := TaskPropertyTokens(t, keys, 0)
	for i, key := range keys {
		cells[i] = strings.TrimPrefix(cells[i], key+"=")
	}
	return cells
}

func tablePropertyWidths(tasks []*task.Task, keys []string) []int {
	const propertyColumnPad = 2
	widths := make([]int, len(keys))
	for i, key := range keys {
		widths[i] = ansi.StringWidth(key) + propertyColumnPad
	}
	for _, t := range tasks {
		for i, cell := range tablePropertyCells(t, keys) {
			widths[i] = max(widths[i], ansi.StringWidth(cell)+propertyColumnPad)
		}
	}
	return widths
}

func tablePropertySuffix(cells []string, widths []int) string {
	var suffix strings.Builder
	for i, cell := range cells {
		suffix.WriteByte(' ')
		suffix.WriteString(padRight(cell, widths[i]))
	}
	return suffix.String()
}

// SelectedTask preserves canonical fields and adds an explicit projection only.
type SelectedTask struct {
	*task.Task
	Properties map[string]property.Scalar `json:"properties"`
}
