package output

import (
	"testing"
)

func TestDetectJSON(t *testing.T) {
	if got := Detect(true, false); got != FormatJSON {
		t.Errorf("Detect(json=true) = %d, want FormatJSON", got)
	}
}

func TestDetectTable(t *testing.T) {
	if got := Detect(false, true); got != FormatTable {
		t.Errorf("Detect(table=true) = %d, want FormatTable", got)
	}
}

func TestDetectEnvJSON(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "json")

	if got := Detect(false, false); got != FormatJSON {
		t.Errorf("Detect with KANBAN_OUTPUT=json = %d, want FormatJSON", got)
	}
}

func TestDetectEnvTable(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "table")

	if got := Detect(false, false); got != FormatTable {
		t.Errorf("Detect with KANBAN_OUTPUT=table = %d, want FormatTable", got)
	}
}

func TestDetectFlagOverridesEnv(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "table")

	if got := Detect(true, false); got != FormatJSON {
		t.Errorf("Detect(json=true) with KANBAN_OUTPUT=table = %d, want FormatJSON (flag wins)", got)
	}
}
