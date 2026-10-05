package cmd

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var showCmd = &cobra.Command{
	Use:   "show ID",
	Short: "Show task details",
	Long:  `Displays full details of a single task including its markdown body.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	showCmd.Flags().StringArray("show-property", nil, "include only this selected scalar property on task and children (KEY; repeatable)")
	showCmd.Flags().Bool("archived", false, "include archived child tasks")
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	keys, err := parsePropertyKeys(cmd, "show-property")
	if err != nil {
		return err
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return task.ValidateTaskID(args[0])
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

	allTasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return err
	}
	printWarnings(warnings)

	includeArchived, _ := cmd.Flags().GetBool("archived")
	parent := board.FindParent(allTasks, t)
	children := board.SummarizeChildren(allTasks, t.ID, cfg, includeArchived)
	return outputShownWithOptions(t, parent, children, allTasks, cfg, keys)
}

type shownTaskDetail struct {
	*task.Task
	Children []board.ChildTask `json:"children"`
}

func outputShownTaskDetail(t *task.Task, parent *board.ParentTask, children board.ChildSummary) error {
	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(os.Stdout, shownTaskDetail{Task: t, Children: children.Children})
	}
	if format == output.FormatCompact {
		output.TaskDetailCompactWithChildren(os.Stdout, t, children)
		return nil
	}

	output.TaskDetailWithRelations(os.Stdout, t, parent, children)
	return nil
}

func outputTaskDetail(t *task.Task) error {
	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(os.Stdout, t)
	}
	if format == output.FormatCompact {
		output.TaskDetailCompact(os.Stdout, t)
		return nil
	}

	output.TaskDetail(os.Stdout, t)
	return nil
}
