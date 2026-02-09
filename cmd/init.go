package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
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
	initCmd.Flags().StringSlice("wip-limit", nil, "WIP limit per status (format: status:N, repeatable)")
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
		return clierr.Newf(clierr.BoardAlreadyExists, "board already initialized in %s", absDir).
			WithDetails(map[string]any{"dir": absDir})
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

	if wipLimits, _ := cmd.Flags().GetStringSlice("wip-limit"); len(wipLimits) > 0 {
		parsed, err := parseWIPLimits(wipLimits)
		if err != nil {
			return err
		}
		cfg.WIPLimits = parsed
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
		return output.JSON(os.Stdout, map[string]string{
			"status":  "initialized",
			"dir":     absDir,
			"name":    name,
			"config":  cfg.ConfigPath(),
			"tasks":   tasksDir,
			"columns": strings.Join(cfg.Statuses, ","),
		})
	}

	output.Messagef(os.Stdout, "Initialized board %q in %s", name, absDir)
	output.Messagef(os.Stdout, "  Config:  %s", cfg.ConfigPath())
	output.Messagef(os.Stdout, "  Tasks:   %s", tasksDir)
	output.Messagef(os.Stdout, "  Columns: %s", strings.Join(cfg.Statuses, ", "))
	output.Messagef(os.Stdout, "  Hint:    Install agent skills with: kanban-md skill install")
	return nil
}

// parseWIPLimits parses "status:N" pairs into a map.
func parseWIPLimits(pairs []string) (map[string]int, error) {
	limits := make(map[string]int, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2) //nolint:mnd // key:value pair
		if len(parts) != 2 {                  //nolint:mnd // key:value pair
			return nil, fmt.Errorf("invalid WIP limit %q (expected status:N)", pair)
		}
		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid WIP limit value %q in %q", parts[1], pair)
		}
		limits[parts[0]] = n
	}
	return limits, nil
}
