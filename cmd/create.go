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

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
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
	createCmd.Flags().StringArray("set-property", nil, "set a scalar property (KEY=LITERAL; repeatable, use JSON quotes for strings)")
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
	if _, err := parsePropertyPlan(cmd); err != nil {
		return err
	}
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

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	title, err := resolveCreateTitle(cmd, args)
	if err != nil {
		return err
	}

	params, err := buildCreateParams(cmd, title)
	if err != nil {
		return err
	}

	result, err := board.Create(cfg, params, time.Now())
	if err != nil {
		return err
	}

	return outputCreateResult(result.Task, result.Path)
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

// buildCreateParams converts CLI flags into board.CreateParams.
func buildCreateParams(cmd *cobra.Command, title string) (board.CreateParams, error) {
	p := board.CreateParams{Title: title}
	plan, err := parsePropertyPlan(cmd)
	if err != nil {
		return p, err
	}
	p.Properties = plan.sets

	p.Status, _ = cmd.Flags().GetString("status")
	p.Priority, _ = cmd.Flags().GetString("priority")
	p.Class, _ = cmd.Flags().GetString("class")
	p.Assignee, _ = cmd.Flags().GetString("assignee")
	p.Tags, _ = cmd.Flags().GetStringSlice("tags")
	p.Body, _ = cmd.Flags().GetString("body")
	p.Estimate, _ = cmd.Flags().GetString("estimate")
	p.Claimant, _ = cmd.Flags().GetString("claim")

	if v, _ := cmd.Flags().GetString("due"); v != "" {
		d, err := date.Parse(v)
		if err != nil {
			return p, task.FormatDueDate(v, err)
		}
		p.Due = &d
	}
	if cmd.Flags().Changed("parent") {
		v, _ := cmd.Flags().GetInt("parent")
		p.Parent = &v
	}
	if v, _ := cmd.Flags().GetIntSlice("depends-on"); len(v) > 0 {
		p.DependsOn = v
	}

	return p, nil
}
