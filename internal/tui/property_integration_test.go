package tui

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func setIntegrationProperty(t *testing.T, tk *task.Task, key, literal string) {
	t.Helper()
	value, err := property.ParseLiteral(literal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tk.SetPropertyScalar(key, value); err != nil {
		t.Fatal(err)
	}
}

func propertyIntegrationBoard(t *testing.T) (*Board, *config.Config) {
	t.Helper()
	tasks := []*task.Task{
		{ID: 1, Title: "Parent", Status: "backlog", Priority: "medium"},
		{ID: 2, Title: "Later child", Status: "todo", Priority: "high", Parent: parentID(1)},
		{ID: 3, Title: "Earlier child", Status: "todo", Priority: "medium", Parent: parentID(1)},
		{ID: 4, Title: "Text order child", Status: "done", Priority: "low", Parent: parentID(1)},
	}
	setIntegrationProperty(t, tasks[0], "kind", "milestone")
	setIntegrationProperty(t, tasks[1], "kind", `"a\n\u001b[31m\u009bX"`)
	setIntegrationProperty(t, tasks[1], "reading_order", "20")
	setIntegrationProperty(t, tasks[2], "kind", "story")
	setIntegrationProperty(t, tasks[2], "reading_order", "10")
	setIntegrationProperty(t, tasks[3], "reading_order", `"1"`)
	b, cfg := newWorkflowBoard(t, tasks)
	cfg.TUI.CardFields = config.FieldList{"property:kind"}
	cfg.Children.DetailSort = "property:reading_order"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	return b, cfg
}

func TestPropertyIntegrationCardAndSharedRelationOrder(t *testing.T) {
	b, cfg := propertyIntegrationBoard(t)
	before := workflowFileBytes(t, cfg)
	card := ansi.Strip(strings.Join(b.cardContentLines(b.activeTask(3), 40), "\n"))
	if !strings.Contains(card, `kind="story"`) {
		t.Fatalf("configured property missing from actual card: %s", card)
	}
	b.selectTaskByID(1)
	b.handleEnter()
	doc := b.detailDocument(b.detailTask)
	var ids []int
	for _, span := range doc.relations {
		ids = append(ids, span.taskID)
	}
	if !slices.Equal(ids, []int{3, 2, 4}) {
		t.Fatalf("relation order=%v; want shared numeric order [3 2 4]", ids)
	}
	text := ansi.Strip(strings.Join(doc.lines, "\n"))
	for _, token := range []string{`kind="milestone"`, `reading_order=--`, `kind="story"`, `reading_order=10`, `reading_order="1"`} {
		if !strings.Contains(text, token) {
			t.Fatalf("detail missing selected value %s:\n%s", token, text)
		}
	}
	if !strings.Contains(text, `\u001b`) || !strings.Contains(text, `\u009b`) {
		t.Fatal("detail must render terminal-control property values as literal escapes")
	}
	if strings.ContainsAny(text, "\x1b\u009b") {
		t.Fatalf("foreign property injected terminal controls: %q", text)
	}
	workflowKey(b, "tab")
	workflowKey(b, "enter")
	requireDetailID(t, b, 3)
	workflowKey(b, "esc")
	workflowKey(b, "q")
	after := workflowFileBytes(t, cfg)
	if len(before) != len(after) {
		t.Fatal("property browsing changed the file set")
	}
	for path, data := range before {
		if !bytes.Equal(data, after[path]) {
			t.Fatalf("property browsing changed %s", path)
		}
	}
}

func TestPropertyIntegrationLosslessDetailWrappingAndMouseReload(t *testing.T) {
	b, cfg := propertyIntegrationBoard(t)
	longValue := strings.Repeat("x", 95) + "END"
	changeWorkflowTask(t, cfg, 1, func(tk *task.Task) { setIntegrationProperty(t, tk, "kind", `"`+longValue+`"`) })
	b.Update(ReloadMsg{})
	b.SetMouseEnabled(true)
	b.Update(tea.WindowSizeMsg{Width: 32, Height: 40})
	b.selectTaskByID(1)
	b.handleEnter()
	doc := b.detailDocument(b.detailTask)
	var selectedLines []string
	for _, line := range doc.lines {
		plain := ansi.Strip(line)
		if ansi.StringWidth(plain) > b.width {
			t.Fatalf("detail line exceeds viewport: %q", plain)
		}
		selectedLines = append(selectedLines, plain)
	}
	if !strings.Contains(strings.Join(selectedLines, ""), `kind="`+longValue+`"`) {
		t.Fatal("long selected property was truncated in detail")
	}
	workflowKey(b, "tab")
	if b.detailFocusID != 3 {
		t.Fatal("first child should have lowest numeric order")
	}
	targets := relationMouseTargets(b, 3)
	if len(targets) == 0 {
		t.Fatal("configured detail should expose real relation targets")
	}
	target := targets[0]
	b.Update(tea.MouseMsg{X: target.rect.x0, Y: target.rect.y0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	changeWorkflowTask(t, cfg, 2, func(tk *task.Task) { setIntegrationProperty(t, tk, "reading_order", "5") })
	b.Update(ReloadMsg{})
	b.Update(tea.MouseMsg{X: target.rect.x0, Y: target.rect.y0, Button: tea.MouseButtonNone, Action: tea.MouseActionRelease})
	requireDetailID(t, b, 1)
	if b.detailFocusID != 3 {
		t.Fatal("numeric reorder must preserve focused task ID")
	}
	doc = b.detailDocument(b.detailTask)
	if doc.relations[0].taskID != 2 {
		t.Fatal("numeric reorder was not reflected in detail relations")
	}
	reorderedTargets := relationMouseTargets(b, 2)
	if len(reorderedTargets) < 2 {
		t.Fatal("fixture must expose multiple wrapped selected-property relation rows")
	}
	for _, target := range reorderedTargets {
		relationClick(b, target.rect.x0, target.rect.y0)
		requireDetailID(t, b, 2)
		workflowKey(b, "esc")
		_ = b.View()
	}
}

func TestPropertyIntegrationColoredCardClipping(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(previousProfile) })
	b, _ := newWorkflowBoard(t, []*task.Task{{
		ID: 1, Title: "Example", Status: "todo", Priority: "medium", Tags: []string{"backend"},
	}})
	b.cfg.TUI.CardFields = config.FieldList{"property:kind"}
	tk := b.activeTask(1)
	setIntegrationProperty(t, tk, "kind", "chapter")
	var plainWithoutColor string
	for _, profile := range []termenv.Profile{termenv.Ascii, termenv.ANSI256} {
		lipgloss.SetColorProfile(profile)
		lines := b.cardContentLines(tk, 24)
		line := lines[len(lines)-1]
		plain := ansi.Strip(line)
		if !strings.Contains(plain, `kind="chapter"`) {
			t.Fatalf("fitting property lost when clipping profile %v: %q", profile, line)
		}
		if ansi.StringWidth(line) > 20 || !strings.HasSuffix(plain, "...") {
			t.Fatalf("later metadata must clip within 20 cells: %q", line)
		}
		if profile == termenv.Ascii {
			plainWithoutColor = plain
			continue
		}
		if plain != plainWithoutColor || strings.ContainsRune(plain, '\x1b') {
			t.Fatalf("ANSI formatting changed visible content: got %q, want %q", plain, plainWithoutColor)
		}
		if !strings.Contains(line, "\x1b[0m") {
			t.Fatalf("clipping lost the style reset: %q", line)
		}
	}
}
