package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List tasks",
	Long:    `Lists tasks with optional filtering, sorting, and output format control.`,
	RunE:    runList,
}

func init() {
	listCmd.Flags().StringSlice("status", nil, "filter by status (comma-separated)")
	listCmd.Flags().StringSlice("priority", nil, "filter by priority (comma-separated)")
	listCmd.Flags().String("assignee", "", "filter by assignee")
	listCmd.Flags().String("tag", "", "filter by tag")
	listCmd.Flags().String("sort", "id", "sort field (id, status, priority, created, updated, due)")
	listCmd.Flags().BoolP("reverse", "r", false, "reverse sort order")
	listCmd.Flags().IntP("limit", "n", 0, "limit number of results")
	listCmd.Flags().Bool("blocked", false, "show only blocked tasks")
	listCmd.Flags().Bool("not-blocked", false, "show only non-blocked tasks")
	listCmd.Flags().Int("parent", 0, "filter by parent task ID")
	listCmd.Flags().Bool("unblocked", false, "show only tasks with all dependencies satisfied")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	statuses, _ := cmd.Flags().GetStringSlice("status")
	priorities, _ := cmd.Flags().GetStringSlice("priority")
	assignee, _ := cmd.Flags().GetString("assignee")
	tag, _ := cmd.Flags().GetString("tag")
	sortBy, _ := cmd.Flags().GetString("sort")
	reverse, _ := cmd.Flags().GetBool("reverse")
	limit, _ := cmd.Flags().GetInt("limit")
	blocked, _ := cmd.Flags().GetBool("blocked")
	notBlocked, _ := cmd.Flags().GetBool("not-blocked")
	parentID, _ := cmd.Flags().GetInt("parent")
	unblocked, _ := cmd.Flags().GetBool("unblocked")

	filter := board.FilterOptions{
		Statuses:   statuses,
		Priorities: priorities,
		Assignee:   assignee,
		Tag:        tag,
	}

	if blocked {
		v := true
		filter.Blocked = &v
	} else if notBlocked {
		v := false
		filter.Blocked = &v
	}

	if cmd.Flags().Changed("parent") {
		filter.ParentID = &parentID
	}

	opts := board.ListOptions{
		Filter:    filter,
		SortBy:    sortBy,
		Reverse:   reverse,
		Limit:     limit,
		Unblocked: unblocked,
	}

	tasks, err := board.List(cfg, opts)
	if err != nil {
		return err
	}

	format := outputFormat()
	if format == output.FormatJSON {
		if tasks == nil {
			tasks = []*task.Task{}
		}
		return output.JSON(os.Stdout, tasks)
	}

	output.TaskTable(os.Stdout, tasks)
	return nil
}
