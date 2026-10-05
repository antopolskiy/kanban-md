package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func relationMouseBoard(t *testing.T) *Board {
	t.Helper()
	tasks := workflowTasks()
	tasks[2].Title = "A story with a deliberately long wrapped relation title for testing"
	b, _ := newWorkflowBoard(t, tasks)
	b.SetMouseEnabled(true)
	b.Update(tea.WindowSizeMsg{Width: 38, Height: 40})
	b.selectTaskByID(2)
	b.handleEnter()
	_ = b.View()
	return b
}

func relationMouseTargets(b *Board, id int) []relationTarget {
	_ = b.View()
	var targets []relationTarget
	for _, target := range b.layout.relations {
		if target.taskID == id {
			targets = append(targets, target)
		}
	}
	return targets
}

func relationClick(b *Board, x, y int) {
	b.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	b.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonNone, Action: tea.MouseActionRelease})
}

func TestWorkflowMouseEveryWrappedRelationLineOpens(t *testing.T) {
	b := relationMouseBoard(t)
	targets := relationMouseTargets(b, 3)
	if len(targets) < 2 {
		t.Fatal("fixture needs a wrapped relation")
	}
	for _, target := range targets {
		relationClick(b, target.rect.x0, target.rect.y0)
		requireDetailID(t, b, 3)
		workflowKey(b, "esc")
		requireDetailID(t, b, 2)
		_ = b.View()
	}
}

func TestWorkflowMouseRelationBackAndFocus(t *testing.T) {
	b := relationMouseBoard(t)
	targets := relationMouseTargets(b, 3)
	relationClick(b, targets[0].rect.x0, targets[0].rect.y0)
	requireDetailID(t, b, 3)
	_ = b.View()
	back := b.layout.back.rect
	relationClick(b, back.x0, back.y0)
	requireDetailID(t, b, 2)
	if b.detailFocusID != 3 {
		t.Fatal("mouse Back should restore the clicked relation focus")
	}
	_ = b.View()
	back = b.layout.back.rect
	relationClick(b, back.x0, back.y0)
	if b.view != viewBoard || b.selectedTask().ID != 2 {
		t.Fatal("mouse Back should finish at the original board card")
	}
}

func TestWorkflowMouseRejectsCancelledAndStaleTargets(t *testing.T) {
	for _, action := range []string{"different row", "blank release", "motion", "keyboard", "resize", "reload", "wheel", "modifier"} {
		t.Run(action, func(t *testing.T) {
			b := relationMouseBoard(t)
			target := relationMouseTargets(b, 3)[0]
			b.Update(tea.MouseMsg{
				X: target.rect.x0, Y: target.rect.y0,
				Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
			})
			x, y := target.rect.x0, target.rect.y0
			switch action {
			case "different row":
				other := relationMouseTargets(b, 1)[0]
				x, y = other.rect.x0, other.rect.y0
			case "blank release":
				x = b.width - 1
			case "motion":
				b.Update(tea.MouseMsg{X: b.width - 1, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
			case "keyboard":
				workflowKey(b, "j")
			case "resize":
				b.Update(tea.WindowSizeMsg{Width: 39, Height: 39})
			case "reload":
				b.Update(ReloadMsg{})
			case "wheel":
				b.Update(tea.MouseMsg{X: 1, Y: 1, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
			case "modifier":
				b.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, Ctrl: true})
			}
			b.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
			requireDetailID(t, b, 2)
			if b.pointer.pressed {
				t.Fatal("rejected release should clear the gesture")
			}
		})
	}
}

func TestWorkflowMouseTargetsExcludeContextBlankAndClippedRows(t *testing.T) {
	b := relationMouseBoard(t)
	b.selectTaskByID(6)
	b.closeDetail()
	b.selectTaskByID(6)
	b.handleEnter()
	_ = b.View()
	if len(relationMouseTargets(b, 5)) != 0 || len(relationMouseTargets(b, 6)) != 0 {
		t.Fatal("archived and current context must not be clickable")
	}
	if len(relationMouseTargets(b, 1)) == 0 {
		t.Fatal("active ancestor above archived context should be clickable")
	}
	for _, target := range b.layout.relations {
		if b.relationAt(b.width-1, target.rect.y0) != nil {
			t.Fatal("blank right-hand cells should not activate a relation")
		}
	}
	b.Update(tea.WindowSizeMsg{Width: 38, Height: 10})
	_ = b.View()
	if len(b.layout.relations) != 0 {
		t.Fatal("off-screen relations should have no initial targets")
	}
	workflowKey(b, "tab")
	_ = b.View()
	if len(b.layout.relations) == 0 {
		t.Fatal("Tab should reveal and expose the active ancestor")
	}
	for _, target := range b.layout.relations {
		if target.rect.y0 < 0 || target.rect.y1 > b.height-detailChrome || target.rect.x1 > b.width {
			t.Fatalf("target escapes viewport: %#v", target)
		}
	}
	if b.relationAt(0, b.height-detailChrome) != nil {
		t.Fatal("fixed bottom chrome must not contain relation targets")
	}
}

func TestWorkflowRelationsMissingSelfCycleAndEmpty(t *testing.T) {
	for _, tt := range []struct {
		id     int
		parent int
		marker string
	}{
		{7, 99, "missing parent #99"}, {7, 7, "self parent #7"}, {1, 3, "cycle to #1"},
	} {
		t.Run(tt.marker, func(t *testing.T) {
			tasks := workflowTasks()
			tasks[tt.id-1].Parent = parentID(tt.parent)
			b, _ := newWorkflowBoard(t, tasks)
			b.selectTaskByID(tt.id)
			b.handleEnter()
			if got := b.View(); !strings.Contains(got, tt.marker) {
				t.Fatalf("missing ancestry marker %q:\n%s", tt.marker, got)
			}
		})
	}
	b, _ := newWorkflowBoard(t, workflowTasks())
	b.selectTaskByID(7)
	b.handleEnter()
	for _, key := range []string{"tab", "shift+tab", "enter"} {
		workflowKey(b, key)
	}
	requireDetailID(t, b, 7)
	if b.detailFocusID != 0 || len(b.detailHistory) != 0 {
		t.Fatal("empty relations must keep focus/history inactive")
	}
}
