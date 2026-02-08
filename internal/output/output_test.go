package output

import (
	"testing"
)

func TestDetectJSON(t *testing.T) {
	if got := Detect(true, false, false); got != FormatJSON {
		t.Errorf("Detect(json=true) = %d, want FormatJSON", got)
	}
}

func TestDetectTable(t *testing.T) {
	if got := Detect(false, true, false); got != FormatTable {
		t.Errorf("Detect(table=true) = %d, want FormatTable", got)
	}
}

func TestDetectCompactFlag(t *testing.T) {
	if got := Detect(false, false, true); got != FormatCompact {
		t.Errorf("Detect(compact=true) = %d, want FormatCompact", got)
	}
}

func TestDetectEnvJSON(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "json")

	if got := Detect(false, false, false); got != FormatJSON {
		t.Errorf("Detect with KANBAN_OUTPUT=json = %d, want FormatJSON", got)
	}
}

func TestDetectEnvTable(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "table")

	if got := Detect(false, false, false); got != FormatTable {
		t.Errorf("Detect with KANBAN_OUTPUT=table = %d, want FormatTable", got)
	}
}

func TestDetectEnvCompact(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "compact")

	if got := Detect(false, false, false); got != FormatCompact {
		t.Errorf("Detect with KANBAN_OUTPUT=compact = %d, want FormatCompact", got)
	}
}

func TestDetectEnvOneline(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "oneline")

	if got := Detect(false, false, false); got != FormatCompact {
		t.Errorf("Detect with KANBAN_OUTPUT=oneline = %d, want FormatCompact", got)
	}
}

func TestDetectFlagOverridesEnv(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "table")

	if got := Detect(true, false, false); got != FormatJSON {
		t.Errorf("Detect(json=true) with KANBAN_OUTPUT=table = %d, want FormatJSON (flag wins)", got)
	}
}

func TestDetectJSONFlagOverridesCompact(t *testing.T) {
	if got := Detect(true, false, true); got != FormatJSON {
		t.Errorf("Detect(json=true, compact=true) = %d, want FormatJSON (json wins)", got)
	}
}

func TestDetectDefaultIsTable(t *testing.T) {
	t.Setenv("KANBAN_OUTPUT", "")

	if got := Detect(false, false, false); got != FormatTable {
		t.Errorf("Detect(default) = %d, want FormatTable", got)
	}
}
