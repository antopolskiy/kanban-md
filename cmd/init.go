package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new kanban board",
	Long:  `Creates a kanban directory with config.yml and tasks/ subdirectory.`,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().String("name", "", "board name (defaults to current directory name)")
	initCmd.Flags().StringSlice("statuses", nil, "comma-separated list of statuses")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, _ []string) error {
	dir := flagDir
	if dir == "" {
		dir = config.DefaultDir
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Check if already initialized.
	if _, err := os.Stat(filepath.Join(absDir, config.ConfigFileName)); err == nil {
		return fmt.Errorf("board already initialized in %s", absDir)
	}

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		name = filepath.Base(cwd)
	}

	cfg := config.NewDefault(name)
	cfg.SetDir(absDir)

	if statuses, _ := cmd.Flags().GetStringSlice("statuses"); len(statuses) > 0 {
		cfg.Statuses = statuses
		cfg.Defaults.Status = statuses[0]
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	// Create directories.
	tasksDir := cfg.TasksPath()
	const dirMode = 0o750
	if err := os.MkdirAll(tasksDir, dirMode); err != nil {
		return fmt.Errorf("creating tasks directory: %w", err)
	}

	// Write config.
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Output result.
	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(map[string]string{
			"status":  "initialized",
			"dir":     absDir,
			"name":    name,
			"config":  cfg.ConfigPath(),
			"tasks":   tasksDir,
			"columns": strings.Join(cfg.Statuses, ","),
		})
	}

	output.Messagef("Initialized board %q in %s", name, absDir)
	output.Messagef("  Config:  %s", cfg.ConfigPath())
	output.Messagef("  Tasks:   %s", tasksDir)
	output.Messagef("  Columns: %s", strings.Join(cfg.Statuses, ", "))
	return nil
}
