package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var editCmd = &cobra.Command{
	Use:   "edit ID[,ID,...]",
	Short: "Edit a task",
	Long: `Modifies fields of an existing task. Only specified fields are changed.
Multiple IDs can be provided as a comma-separated list.`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	editCmd.Flags().String("title", "", "new title")
	editCmd.Flags().String("status", "", "new status")
	editCmd.Flags().String("priority", "", "new priority")
	editCmd.Flags().String("assignee", "", "new assignee")
	editCmd.Flags().StringSlice("add-tag", nil, "add tags")
	editCmd.Flags().StringSlice("remove-tag", nil, "remove tags")
	editCmd.Flags().String("due", "", "new due date (YYYY-MM-DD)")
	editCmd.Flags().Bool("clear-due", false, "clear due date")
	editCmd.Flags().String("estimate", "", "new time estimate")
	editCmd.Flags().String("body", "", "new body text")
	editCmd.Flags().String("started", "", "set started date (YYYY-MM-DD)")
	editCmd.Flags().Bool("clear-started", false, "clear started timestamp")
	editCmd.Flags().String("completed", "", "set completed date (YYYY-MM-DD)")
	editCmd.Flags().Bool("clear-completed", false, "clear completed timestamp")
	editCmd.Flags().Int("parent", 0, "set parent task ID")
	editCmd.Flags().Bool("clear-parent", false, "clear parent")
	editCmd.Flags().IntSlice("add-dep", nil, "add dependency task IDs")
	editCmd.Flags().IntSlice("remove-dep", nil, "remove dependency task IDs")
	editCmd.Flags().String("block", "", "mark task as blocked with reason")
	editCmd.Flags().Bool("unblock", false, "clear blocked state")
	editCmd.Flags().BoolP("force", "f", false, "override WIP limits when changing status")
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	ids, err := parseIDs(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")

	// Single ID: preserve exact current behavior.
	if len(ids) == 1 {
		return editSingleTask(cfg, ids[0], cmd, force)
	}

	// Batch mode.
	return runBatch(ids, func(id int) error {
		return editSingleCore(cfg, id, cmd, force)
	})
}

// editSingleTask handles a single task edit with full output.
func editSingleTask(cfg *config.Config, id int, cmd *cobra.Command, force bool) error {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	oldTitle := t.Title
	oldStatus := t.Status
	wasBlocked := t.Blocked
	changed, err := applyEditFlags(cmd, t, cfg)
	if err != nil {
		return err
	}
	if !changed {
		return clierr.New(clierr.NoChanges, "no changes specified")
	}

	// Validate dependency references.
	if err = validateEditDeps(cfg, t); err != nil {
		return err
	}

	// Check WIP limit if status changed.
	if t.Status != oldStatus {
		if err = enforceWIPLimit(cfg, oldStatus, t.Status, force); err != nil {
			return err
		}
	}

	t.Updated = time.Now()

	var newPath string
	newPath, err = writeAndRename(path, t, oldTitle)
	if err != nil {
		return err
	}

	logEditActivity(cfg, t, wasBlocked)

	if outputFormat() == output.FormatJSON {
		t.File = newPath
		return output.JSON(os.Stdout, t)
	}

	output.Messagef(os.Stdout, "Updated task #%d: %s", t.ID, t.Title)
	return nil
}

// editSingleCore performs the core edit logic without output (for batch mode).
func editSingleCore(cfg *config.Config, id int, cmd *cobra.Command, force bool) error {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	oldTitle := t.Title
	oldStatus := t.Status
	wasBlocked := t.Blocked
	changed, err := applyEditFlags(cmd, t, cfg)
	if err != nil {
		return err
	}
	if !changed {
		return clierr.New(clierr.NoChanges, "no changes specified")
	}

	if err = validateEditDeps(cfg, t); err != nil {
		return err
	}

	if t.Status != oldStatus {
		if err = enforceWIPLimit(cfg, oldStatus, t.Status, force); err != nil {
			return err
		}
	}

	t.Updated = time.Now()

	if _, err = writeAndRename(path, t, oldTitle); err != nil {
		return err
	}

	logEditActivity(cfg, t, wasBlocked)
	return nil
}

// writeAndRename writes the task and renames the file if the title changed.
func writeAndRename(path string, t *task.Task, oldTitle string) (string, error) {
	newPath := path
	if t.Title != oldTitle {
		slug := task.GenerateSlug(t.Title)
		filename := task.GenerateFilename(t.ID, slug)
		newPath = filepath.Join(filepath.Dir(path), filename)
	}

	if err := task.Write(newPath, t); err != nil {
		return "", fmt.Errorf("writing task: %w", err)
	}

	if newPath != path {
		if err := os.Remove(path); err != nil {
			return "", fmt.Errorf("removing old file: %w", err)
		}
	}
	return newPath, nil
}

// logEditActivity logs the edit and any block/unblock transitions.
func logEditActivity(cfg *config.Config, t *task.Task, wasBlocked bool) {
	logActivity(cfg, "edit", t.ID, t.Title)
	if !wasBlocked && t.Blocked {
		logActivity(cfg, "block", t.ID, t.BlockReason)
	}
	if wasBlocked && !t.Blocked {
		logActivity(cfg, "unblock", t.ID, t.Title)
	}
}

func applyEditFlags(cmd *cobra.Command, t *task.Task, cfg *config.Config) (bool, error) {
	changed, err := applySimpleEditFlags(cmd, t, cfg)
	if err != nil {
		return false, err
	}

	// Apply grouped flag helpers, each returning (bool, error).
	for _, fn := range []func(*cobra.Command, *task.Task) (bool, error){
		applyTimestampFlags,
		applyTagDueFlags,
		applyDepFlags,
		applyBlockFlags,
	} {
		c, fnErr := fn(cmd, t)
		if fnErr != nil {
			return false, fnErr
		}
		if c {
			changed = true
		}
	}

	return changed, nil
}

func applySimpleEditFlags(cmd *cobra.Command, t *task.Task, cfg *config.Config) (bool, error) {
	changed := false

	if v, _ := cmd.Flags().GetString("title"); v != "" {
		t.Title = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		if err := task.ValidateStatus(v, cfg.Statuses); err != nil {
			return false, err
		}
		t.Status = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("priority"); v != "" {
		if err := task.ValidatePriority(v, cfg.Priorities); err != nil {
			return false, err
		}
		t.Priority = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("assignee"); v != "" {
		t.Assignee = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("estimate"); v != "" {
		t.Estimate = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("body"); v != "" {
		t.Body = v
		changed = true
	}

	return changed, nil
}

func applyTimestampFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	changed := false

	startedSet := cmd.Flags().Changed("started")
	clearStarted, _ := cmd.Flags().GetBool("clear-started")
	completedSet := cmd.Flags().Changed("completed")
	clearCompleted, _ := cmd.Flags().GetBool("clear-completed")

	if startedSet && clearStarted {
		return false, clierr.New(clierr.StatusConflict, "cannot use --started and --clear-started together")
	}
	if completedSet && clearCompleted {
		return false, clierr.New(clierr.StatusConflict, "cannot use --completed and --clear-completed together")
	}

	if startedSet {
		v, _ := cmd.Flags().GetString("started")
		d, err := date.Parse(v)
		if err != nil {
			return false, task.ValidateDate("started", v, err)
		}
		ts := d.Time
		t.Started = &ts
		changed = true
	}
	if clearStarted {
		t.Started = nil
		changed = true
	}
	if completedSet {
		v, _ := cmd.Flags().GetString("completed")
		d, err := date.Parse(v)
		if err != nil {
			return false, task.ValidateDate("completed", v, err)
		}
		ts := d.Time
		t.Completed = &ts
		changed = true
	}
	if clearCompleted {
		t.Completed = nil
		changed = true
	}

	return changed, nil
}

func applyTagDueFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	changed := false

	if v, _ := cmd.Flags().GetStringSlice("add-tag"); len(v) > 0 {
		t.Tags = appendUnique(t.Tags, v...)
		changed = true
	}
	if v, _ := cmd.Flags().GetStringSlice("remove-tag"); len(v) > 0 {
		t.Tags = removeAll(t.Tags, v...)
		changed = true
	}
	if v, _ := cmd.Flags().GetString("due"); v != "" {
		d, err := date.Parse(v)
		if err != nil {
			return false, task.FormatDueDate(v, err)
		}
		t.Due = &d
		changed = true
	}
	if clearDue, _ := cmd.Flags().GetBool("clear-due"); clearDue {
		t.Due = nil
		changed = true
	}

	return changed, nil
}

func applyDepFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	changed := false

	parentSet := cmd.Flags().Changed("parent")
	clearParent, _ := cmd.Flags().GetBool("clear-parent")

	if parentSet && clearParent {
		return false, clierr.New(clierr.StatusConflict, "cannot use --parent and --clear-parent together")
	}
	if parentSet {
		v, _ := cmd.Flags().GetInt("parent")
		t.Parent = &v
		changed = true
	}
	if clearParent {
		t.Parent = nil
		changed = true
	}

	if v, _ := cmd.Flags().GetIntSlice("add-dep"); len(v) > 0 {
		t.DependsOn = appendUniqueInts(t.DependsOn, v...)
		changed = true
	}
	if v, _ := cmd.Flags().GetIntSlice("remove-dep"); len(v) > 0 {
		t.DependsOn = removeInts(t.DependsOn, v...)
		changed = true
	}

	return changed, nil
}

func appendUniqueInts(slice []int, items ...int) []int {
	seen := make(map[int]bool, len(slice))
	for _, v := range slice {
		seen[v] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}

func removeInts(slice []int, items ...int) []int {
	remove := make(map[int]bool, len(items))
	for _, item := range items {
		remove[item] = true
	}
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if !remove[v] {
			result = append(result, v)
		}
	}
	return result
}

func applyBlockFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	blockReason, _ := cmd.Flags().GetString("block")
	unblock, _ := cmd.Flags().GetBool("unblock")
	blockSet := cmd.Flags().Changed("block")

	if blockSet && unblock {
		return false, clierr.New(clierr.StatusConflict, "cannot use --block and --unblock together")
	}
	if blockSet {
		if blockReason == "" {
			return false, clierr.New(clierr.InvalidInput, "block reason is required (use --block REASON)")
		}
		t.Blocked = true
		t.BlockReason = blockReason
		return true, nil
	}
	if unblock {
		t.Blocked = false
		t.BlockReason = ""
		return true, nil
	}
	return false, nil
}

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}

func removeAll(slice []string, items ...string) []string {
	remove := make(map[string]bool, len(items))
	for _, item := range items {
		remove[item] = true
	}
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !remove[s] {
			result = append(result, s)
		}
	}
	return result
}

func validateEditDeps(cfg *config.Config, t *task.Task) error {
	if t.Parent != nil {
		if err := validateDepIDs(cfg.TasksPath(), t.ID, []int{*t.Parent}); err != nil {
			return fmt.Errorf("invalid parent: %w", err)
		}
	}
	if len(t.DependsOn) > 0 {
		if err := validateDepIDs(cfg.TasksPath(), t.ID, t.DependsOn); err != nil {
			return err
		}
	}
	return nil
}
