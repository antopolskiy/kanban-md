package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show flow metrics",
	Long:  `Displays flow metrics: throughput, average lead/cycle time, flow efficiency, and aging work items.`,
	RunE:  runMetrics,
}

func init() {
	metricsCmd.Flags().String("since", "", "only include tasks completed after this date (YYYY-MM-DD)")
	rootCmd.AddCommand(metricsCmd)
}

func runMetrics(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	tasks, err := task.ReadAll(cfg.TasksPath())
	if err != nil {
		return err
	}
	if tasks == nil {
		tasks = []*task.Task{}
	}

	sinceStr, _ := cmd.Flags().GetString("since")
	if sinceStr != "" {
		d, parseErr := date.Parse(sinceStr)
		if parseErr != nil {
			return parseErr
		}
		sinceTime := d.Time
		filtered := make([]*task.Task, 0, len(tasks))
		for _, t := range tasks {
			if t.Completed == nil || t.Completed.After(sinceTime) {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	now := time.Now()
	m := board.ComputeMetrics(cfg, tasks, now)

	if outputFormat() == output.FormatJSON {
		return output.JSON(m)
	}

	output.MetricsTable(m)
	return nil
}
