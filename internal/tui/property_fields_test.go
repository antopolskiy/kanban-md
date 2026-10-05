package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"go.yaml.in/yaml/v3"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestPropertyFieldHelpers(t *testing.T) {
	cfg := config.NewDefault("test")
	tk := &task.Task{Priority: "medium"}
	if got := propertyCardFields(tk, cfg, 40); len(got) != 1 || got[0] != priorityStyles["medium"].Render("medium") {
		t.Fatal("default priority badge changed")
	}
	if err := yaml.Unmarshal([]byte("id: 1\ntitle: Example\nstatus: backlog\npriority: medium\ntype: \"a\\n\\u001b[31m\"\nrank: 20\n"), tk); err != nil {
		t.Fatal(err)
	}
	cfg.TUI.CardFields = config.FieldList{"property:type"}
	cfg.Children.DetailSort = "property:rank"
	for width := 1; width < 40; width++ {
		for _, token := range propertyCardFields(tk, cfg, width) {
			plain := ansi.Strip(token)
			if ansi.StringWidth(plain) > width || strings.ContainsAny(plain, "\n\x1b") {
				t.Fatalf("unsafe badge %q", plain)
			}
		}
	}
	if lines := propertyDetailLines(tk, cfg); len(lines) != 2 || !strings.Contains(lines[1], "rank=20") {
		t.Fatalf("detail=%v", lines)
	}
	cfg.TUI.CardFields = config.FieldList{"property:type", "property:rank"}
	for width := 1; width < 40; width++ {
		line := strings.Join(propertyCardFields(tk, cfg, width), " ")
		if ansi.StringWidth(line) > width {
			t.Fatalf("combined badge width %d: %q", width, line)
		}
	}
}
