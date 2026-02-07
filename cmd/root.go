// Package cmd implements the kanban-md CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
)

// version is set at build time via ldflags.
var version = "dev"

// Global flags.
var (
	flagJSON    bool
	flagTable   bool
	flagDir     string
	flagNoColor bool
)

var rootCmd = &cobra.Command{
	Use:   "kanban-md",
	Short: "A file-based Kanban tool powered by Markdown",
	Long: `kanban-md is a CLI tool for managing Kanban boards using plain Markdown files.
Tasks are stored as individual files with YAML frontmatter, making them
easy to read, edit, and version-control. Designed for AI agents and humans alike.`,
	Version: version,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		if flagNoColor || os.Getenv("NO_COLOR") != "" {
			output.DisableColor()
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolVar(&flagTable, "table", false, "output as table")
	rootCmd.PersistentFlags().StringVar(&flagDir, "dir", "", "path to kanban directory")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable color output")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadConfig finds and loads the kanban config.
func loadConfig() (*config.Config, error) {
	if flagDir != "" {
		return config.Load(flagDir)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	dir, err := config.FindDir(cwd)
	if err != nil {
		return nil, err
	}

	return config.Load(dir)
}

// outputFormat returns the detected output format from flags/env/TTY.
func outputFormat() output.Format {
	return output.Detect(flagJSON, flagTable)
}
