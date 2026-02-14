package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/agentname"
)

var agentNameCmd = &cobra.Command{
	Use:   "agent-name",
	Short: "Generate a unique agent name for claims",
	Long: `Generate a random two-word name suitable for use with --claim.

The name is printed on stdout with no decoration, so it can be used directly:

  kanban-md pick --claim $(kanban-md agent-name) --status todo --move in-progress

Uses the system dictionary (/usr/share/dict/words) when available,
with an embedded word list as fallback.`,
	Args: cobra.NoArgs,
	RunE: runAgentName,
}

func init() {
	rootCmd.AddCommand(agentNameCmd)
}

func runAgentName(_ *cobra.Command, _ []string) error {
	name, err := agentname.Generate()
	if err != nil {
		return fmt.Errorf("generating agent name: %w", err)
	}
	fmt.Println(name)
	return nil
}
