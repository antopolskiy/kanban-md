package cmd

import (
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
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
	listCmd.Flags().Bool("unblocked", false, "show only tasks with all dependencies satisfied (missing dependency IDs are treated as satisfied)")
	listCmd.Flags().Bool("unclaimed", false, "show only unclaimed or expired-claim tasks")
	listCmd.Flags().String("claimed-by", "", "filter by claimant")
	listCmd.Flags().String("class", "", "filter by class of service")
	listCmd.Flags().StringP("search", "s", "", "search tasks by title, body, or tags (case-insensitive)")
	listCmd.Flags().Bool("archived", false, "show only archived tasks")
	listCmd.Flags().String("group-by", "", "group results by field ("+strings.Join(board.ValidGroupByFields(), ", ")+")")
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
	unclaimed, _ := cmd.Flags().GetBool("unclaimed")
	claimedBy, _ := cmd.Flags().GetString("claimed-by")
	class, _ := cmd.Flags().GetString("class")
	search, _ := cmd.Flags().GetString("search")
	groupBy, _ := cmd.Flags().GetString("group-by")
	archived, _ := cmd.Flags().GetBool("archived")

	if groupBy != "" && !slices.Contains(board.ValidGroupByFields(), groupBy) {
		return clierr.Newf(clierr.InvalidGroupBy, "invalid --group-by field %q; valid: %s",
			groupBy, strings.Join(board.ValidGroupByFields(), ", "))
	}

	filter := board.FilterOptions{
		Statuses:     statuses,
		Priorities:   priorities,
		Assignee:     assignee,
		Tag:          tag,
		Search:       search,
		ClaimTimeout: cfg.ClaimTimeoutDuration(),
	}

	// --archived flag: show only archived tasks.
	// Default (no --status, no --archived): exclude archived.
	if archived {
		filter.Statuses = []string{config.ArchivedStatus}
	} else if !cmd.Flags().Changed("status") {
		filter.ExcludeStatuses = []string{config.ArchivedStatus}
	}

	if unclaimed {
		filter.Unclaimed = true
	}
	if claimedBy != "" {
		filter.ClaimedBy = claimedBy
	}
	if class != "" {
		filter.Class = class
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

	tasks, warnings, err := board.List(cfg, opts)
	if err != nil {
		return err
	}
	printWarnings(warnings)

	if groupBy != "" {
		return outputGroupedList(tasks, groupBy, cfg)
	}

	return outputTaskList(tasks)
}

func outputGroupedList(tasks []*task.Task, groupBy string, cfg *config.Config) error {
	grouped := board.GroupBy(tasks, groupBy, cfg)
	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, grouped)
	}
	output.GroupedTable(os.Stdout, grouped)
	return nil
}

func outputTaskList(tasks []*task.Task) error {
	format := outputFormat()
	if format == output.FormatJSON {
		if tasks == nil {
			tasks = []*task.Task{}
		}
		return output.JSON(os.Stdout, tasks)
	}
	if format == output.FormatCompact {
		output.TaskCompact(os.Stdout, tasks)
		return nil
	}

	output.TaskTable(os.Stdout, tasks)
	return nil
}
