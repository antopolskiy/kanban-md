package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"go.yaml.in/yaml/v3"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestPropertyCompactSafetyDefaultsAndProjection(t *testing.T) {
	var tk task.Task
	if err := yaml.Unmarshal([]byte("id: 1\ntitle: Example\nstatus: backlog\npriority: medium\ntype: \"a\\n\\u001b[31m\\u009bX\"\nprivate: [secret]\nnullish: null\n"), &tk); err != nil {
		t.Fatal(err)
	}
	var baseline, defaultOpts, selected bytes.Buffer
	TaskCompact(&baseline, []*task.Task{&tk})
	TaskCompactWithOptions(&defaultOpts, []*task.Task{&tk}, TaskViewOptions{CompactFields: []string{"status", "priority"}})
	if baseline.String() != defaultOpts.String() {
		t.Fatal("default compact changed")
	}
	TaskCompactWithOptions(&selected, []*task.Task{&tk}, TaskViewOptions{CompactFields: []string{"status", "property:type"}})
	if strings.Count(selected.String(), "\n") != 1 || strings.ContainsAny(selected.String(), "\x1b\u009b") {
		t.Fatalf("unsafe compact: %q", selected.String())
	}
	for width := 1; width < 40; width++ {
		for _, token := range TaskPropertyTokens(&tk, []string{"type"}, width) {
			if ansi.StringWidth(token) > width {
				t.Fatalf("width %d: %q", width, token)
			}
		}
	}
}

func TestPropertySelectedProjectionPrivacy(t *testing.T) {
	var tk task.Task
	if err := yaml.Unmarshal([]byte("id: 1\ntitle: Example\nstatus: backlog\npriority: medium\ntype: story\nprivate: [secret]\nnullish: null\n"), &tk); err != nil {
		t.Fatal(err)
	}
	values, warnings := SelectedProperties(&tk, []string{"type", "nullish", "missing", "private"})
	if len(warnings) != 1 || strings.Contains(warnings[0], "secret") || len(values) != 2 {
		t.Fatalf("projection=%v warnings=%v", values, warnings)
	}
	data, err := json.Marshal(SelectedTask{Task: &tk, Properties: values})
	if err != nil || !json.Valid(data) || strings.Contains(string(data), "secret") {
		t.Fatalf("JSON %s %v", data, err)
	}
	plain, _ := json.Marshal(tk)
	if strings.Contains(string(plain), "properties") || strings.Contains(string(plain), "nullish") {
		t.Fatal("default JSON leak")
	}
}

func TestPropertyTableColumnsAlign(t *testing.T) {
	var tk task.Task
	if err := yaml.Unmarshal([]byte("id: 1\ntitle: Example\nstatus: backlog\npriority: medium\ntype: story\nrank: 20\n"), &tk); err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	TaskTableWithProperties(&rendered, []*task.Task{&tk}, []string{"type", "rank"})
	lines := strings.Split(ansi.Strip(rendered.String()), "\n")
	if strings.Index(lines[0], "type") != strings.Index(lines[1], `"story"`) || strings.Index(lines[0], "rank") != strings.Index(lines[1], "20") {
		t.Fatalf("columns misaligned:\n%s", rendered.String())
	}
}
