package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestPropertyFlagParsingAndConflicts(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("set-property", nil, "")
	cmd.Flags().StringArray("clear-property", nil, "")
	if err := cmd.Flags().Parse([]string{"--set-property", `note="a,b"`, "--set-property", "rank=20"}); err != nil {
		t.Fatal(err)
	}
	plan, err := parsePropertyPlan(cmd)
	if err != nil || len(plan.sets) != 2 || plan.sets[0].Value.JSONLiteral() != `"a,b"` {
		t.Fatalf("plan=%v %v", plan, err)
	}
	if err := cmd.Flags().Set("clear-property", "rank"); err != nil {
		t.Fatal(err)
	}
	if _, err := parsePropertyPlan(cmd); err == nil {
		t.Fatal("conflict accepted")
	}
}
