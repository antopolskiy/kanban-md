package cmd

import "testing"

func TestRootCommand(t *testing.T) {
	if rootCmd.Use != "kanban-md" {
		t.Errorf("rootCmd.Use = %v, want kanban-md", rootCmd.Use)
	}
}
