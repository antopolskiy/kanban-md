package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

type propertyPlan struct {
	sets   []property.Assignment
	clears []string
}

type listPropertyOptions struct {
	tests     []property.Assignment
	keys      []string
	group     string
	tableOpts output.TableOptions
}

func parseListPropertyOptions(cmd *cobra.Command) (listPropertyOptions, error) {
	tests, err := parsePropertyAssignments(cmd, "property")
	if err != nil {
		return listPropertyOptions{}, err
	}
	keys, err := parsePropertyKeys(cmd, "show-property")
	if err != nil {
		return listPropertyOptions{}, err
	}
	group, _ := cmd.Flags().GetString("group-by")
	if err = validatePropertyGroup(group, keys); err != nil {
		return listPropertyOptions{}, err
	}
	tableOpts, err := parseTableOptions(cmd, group)
	if err != nil {
		return listPropertyOptions{}, err
	}
	return listPropertyOptions{tests: tests, keys: keys, group: group, tableOpts: tableOpts}, nil
}

func propertyFlagValues(cmd *cobra.Command, name string) []string {
	if cmd.Flags().Lookup(name) == nil {
		return nil
	}
	values, _ := cmd.Flags().GetStringArray(name)
	return values
}

func parsePropertyAssignments(cmd *cobra.Command, name string) ([]property.Assignment, error) {
	var values []property.Assignment
	seen := map[string]bool{}
	for _, raw := range propertyFlagValues(cmd, name) {
		value, err := property.ParseAssignment(raw)
		if err != nil {
			return nil, clierr.New(clierr.InvalidInput, err.Error())
		}
		if seen[value.Key] {
			return nil, clierr.Newf(clierr.InvalidInput, "duplicate --%s key %q", name, value.Key)
		}
		seen[value.Key] = true
		values = append(values, value)
	}
	return values, nil
}

func parsePropertyKeys(cmd *cobra.Command, name string) ([]string, error) {
	keys := propertyFlagValues(cmd, name)
	seen := map[string]bool{}
	for _, key := range keys {
		if err := property.ValidateKey(key); err != nil {
			return nil, clierr.New(clierr.InvalidInput, err.Error())
		}
		if seen[key] {
			return nil, clierr.Newf(clierr.InvalidInput, "duplicate --%s key %q", name, key)
		}
		seen[key] = true
	}
	return keys, nil
}

func parsePropertyPlan(cmd *cobra.Command) (propertyPlan, error) {
	sets, err := parsePropertyAssignments(cmd, "set-property")
	if err != nil {
		return propertyPlan{}, err
	}
	clears, err := parsePropertyKeys(cmd, "clear-property")
	if err != nil {
		return propertyPlan{}, err
	}
	for _, set := range sets {
		if slices.Contains(clears, set.Key) {
			return propertyPlan{}, clierr.Newf(clierr.StatusConflict, "cannot set and clear property %q together", set.Key)
		}
	}
	return propertyPlan{sets: sets, clears: clears}, nil
}

func (plan propertyPlan) apply(t *task.Task) (bool, error) {
	changed := false
	for _, set := range plan.sets {
		c, err := t.SetPropertyScalar(set.Key, set.Value)
		if err != nil {
			return false, err
		}
		changed = changed || c
	}
	for _, key := range plan.clears {
		c, err := t.ClearProperty(key)
		if err != nil {
			return false, err
		}
		changed = changed || c
	}
	return changed, nil
}

func humanPropertyKeys(requested, fields []string, childSort string) []string {
	keys := append([]string{}, requested...)
	for _, selector := range append(append([]string{}, fields...), childSort) {
		if key, selected, err := property.SelectorKey(selector); selected && err == nil && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys
}

func outputTaskListWithOptions(tasks []*task.Task, cfg *config.Config, keys []string, tableOpts output.TableOptions) error {
	if outputFormat() == output.FormatJSON {
		if len(keys) == 0 {
			return outputTaskList(tasks)
		}
		projected := make([]output.SelectedTask, 0, len(tasks))
		for _, t := range tasks {
			values, warnings := output.SelectedProperties(t, keys)
			printPropertyWarnings(warnings)
			projected = append(projected, output.SelectedTask{Task: t, Properties: values})
		}
		return output.JSON(os.Stdout, projected)
	}
	if outputFormat() == output.FormatCompact {
		output.TaskCompactWithOptions(os.Stdout, tasks, output.TaskViewOptions{CompactFields: cfg.CompactFields(), PropertyKeys: keys})
		return nil
	}
	output.TaskTableWithProperties(os.Stdout, tasks, keys, tableOpts)
	return nil
}

type selectedChild struct {
	board.ChildTask
	Properties map[string]property.Scalar `json:"properties"`
}
type selectedShown struct {
	output.SelectedTask
	Children []selectedChild `json:"children"`
}

func outputShownWithOptions(t *task.Task, parent *board.ParentTask, children board.ChildSummary, allTasks []*task.Task, cfg *config.Config, keys []string) error {
	if outputFormat() == output.FormatJSON {
		if len(keys) == 0 {
			return outputShownTaskDetail(t, parent, children)
		}
		values, warnings := output.SelectedProperties(t, keys)
		printPropertyWarnings(warnings)
		result := selectedShown{SelectedTask: output.SelectedTask{Task: t, Properties: values}, Children: make([]selectedChild, 0, len(children.Children))}
		lookup := map[int]*task.Task{}
		for _, candidate := range allTasks {
			lookup[candidate.ID] = candidate
		}
		for _, child := range children.Children {
			values := map[string]property.Scalar{}
			if candidate := lookup[child.ID]; candidate != nil {
				var warnings []string
				values, warnings = output.SelectedProperties(candidate, keys)
				printPropertyWarnings(warnings)
			}
			result.Children = append(result.Children, selectedChild{ChildTask: child, Properties: values})
		}
		return output.JSON(os.Stdout, result)
	}
	humanKeys := humanPropertyKeys(keys, cfg.CompactFields(), cfg.Children.DetailSort)
	board.SelectChildProperties(&children, allTasks, humanKeys)
	if outputFormat() == output.FormatCompact {
		output.TaskDetailCompactWithOptions(os.Stdout, t, children, output.TaskViewOptions{CompactFields: cfg.CompactFields(), PropertyKeys: humanKeys})
		return nil
	}
	output.TaskDetailWithProperties(os.Stdout, t, parent, children, humanKeys)
	return nil
}

func validatePropertyGroup(group string, keys []string) error {
	if group != "" && !board.ValidGroupBy(group) {
		return clierr.Newf(clierr.InvalidGroupBy, "invalid --group-by field %q; valid: %s, property:KEY", group, strings.Join(board.ValidGroupByFields(), ", "))
	}
	if group != "" && len(keys) > 0 {
		return clierr.New(clierr.InvalidInput, "--show-property requires an ungrouped task list")
	}
	return nil
}

func parseTableOptions(cmd *cobra.Command, group string) (output.TableOptions, error) {
	columnsRaw, _ := cmd.Flags().GetString("columns")
	widthChanged := cmd.Flags().Changed("title-width")
	columnsActive := columnsRaw != ""
	if !columnsActive && !widthChanged {
		return output.TableOptions{}, nil
	}

	fmtMode := outputFormat()
	if fmtMode == output.FormatJSON {
		return output.TableOptions{}, clierr.New(clierr.InvalidInput, "--columns and --title-width are only valid with table output")
	}
	if fmtMode == output.FormatCompact {
		return output.TableOptions{}, clierr.New(clierr.InvalidInput, "--columns and --title-width are only valid with table output")
	}
	if group != "" {
		return output.TableOptions{}, clierr.New(clierr.InvalidInput, "--columns and --title-width cannot be used with --group-by")
	}

	var opts output.TableOptions

	if columnsActive {
		raw, _ := cmd.Flags().GetString("columns")
		parts := strings.Split(raw, ",")
		seen := make(map[string]bool, len(parts))
		for _, name := range parts {
			if name == "" {
				return output.TableOptions{}, clierr.Newf(clierr.InvalidInput,
					"--columns: empty column name; valid: %s", strings.Join(output.ValidTableColumns, ", "))
			}
			if !slices.Contains(output.ValidTableColumns, name) {
				return output.TableOptions{}, clierr.Newf(clierr.InvalidInput,
					"--columns: unknown column %q; valid: %s", name, strings.Join(output.ValidTableColumns, ", "))
			}
			if seen[name] {
				return output.TableOptions{}, clierr.Newf(clierr.InvalidInput,
					"--columns: duplicate column %q", name)
			}
			seen[name] = true
		}
		opts.Columns = parts
	}

	if widthChanged {
		w, _ := cmd.Flags().GetInt("title-width")
		if w <= 0 {
			return output.TableOptions{}, clierr.Newf(clierr.InvalidInput,
				"--title-width must be a positive integer, got %d", w)
		}
		opts.TitleWidth = w
	}

	return opts, nil
}

func printPropertyWarnings(warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "Warning: "+warning)
	}
}
