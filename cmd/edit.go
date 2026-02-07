package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var editCmd = &cobra.Command{
	Use:   "edit ID",
	Short: "Edit a task",
	Long:  `Modifies fields of an existing task. Only specified fields are changed.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
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
	editCmd.Flags().String("block", "", "mark task as blocked with reason")
	editCmd.Flags().Bool("unblock", false, "clear blocked state")
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid task ID %q: %w", args[0], err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	oldTitle := t.Title
	changed, err := applyEditFlags(cmd, t, cfg)
	if err != nil {
		return err
	}
	if !changed {
		return errors.New("no changes specified")
	}

	t.Updated = time.Now()

	// If title changed, rename the file.
	newPath := path
	if t.Title != oldTitle {
		slug := task.GenerateSlug(t.Title)
		filename := task.GenerateFilename(t.ID, slug)
		newPath = filepath.Join(filepath.Dir(path), filename)
	}

	if err := task.Write(newPath, t); err != nil {
		return fmt.Errorf("writing task: %w", err)
	}

	// Remove old file if renamed.
	if newPath != path {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing old file: %w", err)
		}
	}

	if outputFormat() == output.FormatJSON {
		t.File = newPath
		return output.JSON(t)
	}

	output.Messagef("Updated task #%d: %s", t.ID, t.Title)
	return nil
}

func applyEditFlags(cmd *cobra.Command, t *task.Task, cfg *config.Config) (bool, error) {
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
	tagDueChanged, err := applyTagDueFlags(cmd, t)
	if err != nil {
		return false, err
	}
	if tagDueChanged {
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

	blockChanged, err := applyBlockFlags(cmd, t)
	if err != nil {
		return false, err
	}
	if blockChanged {
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
			return false, fmt.Errorf("invalid due date: %w", err)
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

func applyBlockFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	blockReason, _ := cmd.Flags().GetString("block")
	unblock, _ := cmd.Flags().GetBool("unblock")
	blockSet := cmd.Flags().Changed("block")

	if blockSet && unblock {
		return false, errors.New("cannot use --block and --unblock together")
	}
	if blockSet {
		if blockReason == "" {
			return false, errors.New("block reason is required (use --block REASON)")
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
