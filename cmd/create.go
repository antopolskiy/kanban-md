package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/filelock"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var createCmd = &cobra.Command{
	Use:     "create [TITLE]",
	Aliases: []string{"add"},
	Short:   "Create a new task",
	Long: `Creates a new task file with the given title and optional fields.

Title can be provided as a positional argument or via --title flag.
Body/description can be provided via --body or --description flag.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().String("title", "", "task title (alternative to positional argument)")
	createCmd.Flags().String("status", "", "task status (default from config)")
	createCmd.Flags().String("priority", "", "task priority (default from config)")
	createCmd.Flags().String("assignee", "", "task assignee")
	createCmd.Flags().StringSlice("tags", nil, "comma-separated tags")
	createCmd.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "tag":
			name = "tags"
		case "description":
			name = "body"
		}
		return pflag.NormalizedName(name)
	})
	createCmd.Flags().String("due", "", "due date (YYYY-MM-DD)")
	createCmd.Flags().String("estimate", "", "time estimate (e.g. 4h, 2d)")
	createCmd.Flags().Int("parent", 0, "parent task ID")
	createCmd.Flags().IntSlice("depends-on", nil, "dependency task IDs (comma-separated)")
	createCmd.Flags().String("body", "", "task body/description (markdown)")
	createCmd.Flags().String("class", "", "class of service (expedite, fixed-date, standard, intangible)")
	createCmd.Flags().String("claim", "", "claim task for an agent (use 'agent-name' to generate)")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Acquire an exclusive lock to prevent concurrent creates from
	// reading the same next_id and generating duplicate task IDs.
	dir, err := resolveDir()
	if err != nil {
		return err
	}
	unlock, err := filelock.Lock(filepath.Join(dir, ".lock"))
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer unlock() //nolint:errcheck // best-effort unlock on exit

	cfg, err := config.Load(dir)
	if err != nil {
		return err
	}

	title, err := resolveCreateTitle(cmd, args)
	if err != nil {
		return err
	}
	now := time.Now()

	t := &task.Task{
		ID:       cfg.NextID,
		Title:    title,
		Status:   cfg.Defaults.Status,
		Priority: cfg.Defaults.Priority,
		Class:    cfg.Defaults.Class,
		Created:  now,
		Updated:  now,
	}

	if err := applyCreateFlags(cmd, t, cfg); err != nil {
		return err
	}

	// Validate dependency references.
	if err := validateDeps(cfg, t); err != nil {
		return err
	}

	// Check WIP limit for the target status (class-aware).
	if t.Class != "" && len(cfg.Classes) > 0 {
		if err := enforceWIPLimitForClass(cfg, t, "", t.Status); err != nil {
			return err
		}
	} else {
		if err := enforceWIPLimit(cfg, "", t.Status); err != nil {
			return err
		}
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

	logActivity(cfg, "create", t.ID, t.Title)

	return outputCreateResult(t, path)
}

func outputCreateResult(t *task.Task, path string) error {
	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, t)
	}

	output.Messagef(os.Stdout, "Created task #%d: %s", t.ID, t.Title)
	output.Messagef(os.Stdout, "  File: %s", path)
	output.Messagef(os.Stdout, "  Status: %s | Priority: %s", t.Status, t.Priority)
	if t.Assignee != "" {
		output.Messagef(os.Stdout, "  Assignee: %s", t.Assignee)
	}
	if len(t.Tags) > 0 {
		output.Messagef(os.Stdout, "  Tags: %s", strings.Join(t.Tags, ", "))
	}
	return nil
}

// resolveCreateTitle returns the task title from either the positional arg or --title flag.
func resolveCreateTitle(cmd *cobra.Command, args []string) (string, error) {
	flagTitle, _ := cmd.Flags().GetString("title")
	hasPositional := len(args) > 0
	hasFlag := flagTitle != ""

	switch {
	case hasPositional && hasFlag:
		return "", clierr.New(clierr.InvalidInput,
			"title provided both as argument and --title flag; use one or the other")
	case hasPositional:
		return args[0], nil
	case hasFlag:
		return flagTitle, nil
	default:
		return "", errors.New("title is required: provide it as an argument or with --title")
	}
}

func applyCreateFlags(cmd *cobra.Command, t *task.Task, cfg *config.Config) error {
	if err := applyCreateValidatedFlags(cmd, t, cfg); err != nil {
		return err
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
			return task.FormatDueDate(v, err)
		}
		t.Due = &d
	}
	if v, _ := cmd.Flags().GetString("estimate"); v != "" {
		t.Estimate = v
	}
	if cmd.Flags().Changed("parent") {
		v, _ := cmd.Flags().GetInt("parent")
		t.Parent = &v
	}
	if v, _ := cmd.Flags().GetIntSlice("depends-on"); len(v) > 0 {
		t.DependsOn = v
	}
	if v, _ := cmd.Flags().GetString("body"); v != "" {
		t.Body = v
	}
	if v, _ := cmd.Flags().GetString("claim"); v != "" {
		now := time.Now()
		t.ClaimedBy = v
		t.ClaimedAt = &now
	}
	return nil
}

// applyCreateValidatedFlags handles flags that require validation against config.
func applyCreateValidatedFlags(cmd *cobra.Command, t *task.Task, cfg *config.Config) error {
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		if err := task.ValidateStatus(v, cfg.StatusNames()); err != nil {
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
	if v, _ := cmd.Flags().GetString("class"); v != "" {
		if err := task.ValidateClass(v, cfg.ClassNames()); err != nil {
			return err
		}
		t.Class = v
	}
	return nil
}
