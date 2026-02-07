package cmd

import (
	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/output"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show activity log",
	Long:  `Displays the activity log of board mutations (create, move, edit, delete, block, unblock).`,
	RunE:  runLog,
}

func init() {
	logCmd.Flags().String("since", "", "show entries after this date (YYYY-MM-DD)")
	logCmd.Flags().Int("limit", 0, "maximum number of entries to show (most recent)")
	logCmd.Flags().String("action", "", "filter by action type (create, move, edit, delete, block, unblock)")
	logCmd.Flags().Int("task", 0, "filter by task ID")
	rootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	opts := board.LogFilterOptions{}

	if v, _ := cmd.Flags().GetString("since"); v != "" {
		d, parseErr := date.Parse(v)
		if parseErr != nil {
			return parseErr
		}
		opts.Since = d.Time
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	if v, _ := cmd.Flags().GetString("action"); v != "" {
		opts.Action = v
	}
	if v, _ := cmd.Flags().GetInt("task"); v > 0 {
		opts.TaskID = v
	}

	entries, err := board.ReadLog(cfg.Dir(), opts)
	if err != nil {
		return err
	}

	if outputFormat() == output.FormatJSON {
		if entries == nil {
			entries = []board.LogEntry{}
		}
		return output.JSON(entries)
	}

	output.ActivityLogTable(entries)
	return nil
}
