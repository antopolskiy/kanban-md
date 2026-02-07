package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var createCmd = &cobra.Command{
	Use:     "create TITLE",
	Aliases: []string{"add"},
	Short:   "Create a new task",
	Long:    `Creates a new task file with the given title and optional fields.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runCreate,
}

func init() {
	createCmd.Flags().String("status", "", "task status (default from config)")
	createCmd.Flags().String("priority", "", "task priority (default from config)")
	createCmd.Flags().String("assignee", "", "task assignee")
	createCmd.Flags().StringSlice("tags", nil, "comma-separated tags")
	createCmd.Flags().String("due", "", "due date (YYYY-MM-DD)")
	createCmd.Flags().String("estimate", "", "time estimate (e.g. 4h, 2d)")
	createCmd.Flags().String("body", "", "task body (markdown)")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	title := args[0]
	now := time.Now()

	t := &task.Task{
		ID:       cfg.NextID,
		Title:    title,
		Status:   cfg.Defaults.Status,
		Priority: cfg.Defaults.Priority,
		Created:  now,
		Updated:  now,
	}

	if err := applyCreateFlags(cmd, t, cfg); err != nil {
		return err
	}

	// Generate filename and write.
	slug := task.GenerateSlug(title)
	filename := task.GenerateFilename(t.ID, slug)
	path := filepath.Join(cfg.TasksPath(), filename)
	t.File = path

	if err := task.Write(path, t); err != nil {
		return fmt.Errorf("writing task: %w", err)
	}

	// Increment next_id and save config.
	cfg.NextID++
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Output.
	if outputFormat() == output.FormatJSON {
		return output.JSON(t)
	}

	output.Messagef("Created task #%d: %s", t.ID, t.Title)
	output.Messagef("  File: %s", path)
	output.Messagef("  Status: %s | Priority: %s", t.Status, t.Priority)
	if t.Assignee != "" {
		output.Messagef("  Assignee: %s", t.Assignee)
	}
	if len(t.Tags) > 0 {
		output.Messagef("  Tags: %s", strings.Join(t.Tags, ", "))
	}
	return nil
}

func applyCreateFlags(cmd *cobra.Command, t *task.Task, cfg *config.Config) error {
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		if err := task.ValidateStatus(v, cfg.Statuses); err != nil {
			return err
		}
		t.Status = v
	}
	if v, _ := cmd.Flags().GetString("priority"); v != "" {
		if err := task.ValidatePriority(v, cfg.Priorities); err != nil {
			return err
		}
		t.Priority = v
	}
	if v, _ := cmd.Flags().GetString("assignee"); v != "" {
		t.Assignee = v
	}
	if v, _ := cmd.Flags().GetStringSlice("tags"); len(v) > 0 {
		t.Tags = v
	}
	if v, _ := cmd.Flags().GetString("due"); v != "" {
		d, err := date.Parse(v)
		if err != nil {
			return fmt.Errorf("invalid due date: %w", err)
		}
		t.Due = &d
	}
	if v, _ := cmd.Flags().GetString("estimate"); v != "" {
		t.Estimate = v
	}
	if v, _ := cmd.Flags().GetString("body"); v != "" {
		t.Body = v
	}
	return nil
}
