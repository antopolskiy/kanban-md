// Package cmd implements the kanban-md CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "kanban-md",
	Short: "A file-based Kanban tool powered by Markdown",
	Long: `kanban-md is a CLI tool for managing Kanban boards using plain Markdown files.
Tasks are stored as individual files with YAML frontmatter, making them
easy to read, edit, and version-control. Designed for AI agents and humans alike.`,
	Version: version,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
