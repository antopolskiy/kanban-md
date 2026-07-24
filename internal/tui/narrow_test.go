package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestNarrow_AutoThresholdByColumnCount(t *testing.T) {
	b := newMouseTestBoard()
	n := len(b.columns)
	if n == 0 {
		t.Fatal("test board has no columns")
	}
	threshold := minUsableColumnWidth * n

	b.Update(tea.WindowSizeMsg{Width: threshold - 1, Height: 30})
	if !b.narrow() {
		t.Errorf("width %d (< %d) should be narrow with %d columns", threshold-1, threshold, n)
	}
	b.Update(tea.WindowSizeMsg{Width: threshold, Height: 30})
	if b.narrow() {
		t.Errorf("width %d (>= %d) should stay wide with %d columns", threshold, threshold, n)
	}
}

func TestNarrow_ForceOverridesWidth(t *testing.T) {
	b := newMouseTestBoard()
	b.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	if b.narrow() {
		t.Fatal("width 200 should be wide before forcing")
	}
	b.SetForceNarrow(true)
	if !b.narrow() {
		t.Error("SetForceNarrow(true) should force narrow at any width")
	}
}

func TestNarrow_ConfigThresholdOverridesAuto(t *testing.T) {
	b := newMouseTestBoard()
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	b.SetNarrowThreshold(200) // everything below 200 is narrow
	if !b.narrow() {
		t.Error("configured threshold 200 should make width 120 narrow")
	}
	b.SetNarrowThreshold(10) // effectively disables narrow mode
	if b.narrow() {
		t.Error("configured threshold 10 should make width 120 wide")
	}
}

// The active column's full name shows on its own header line even when the
// tab strip must abbreviate it.
func TestNarrow_ActiveHeaderShowsFullName(t *testing.T) {
	b := newMouseTestBoard()
	const longName = "waiting-stakeholder"
	b.columns[0].status = longName
	b.activeCol = 0
	b.Update(tea.WindowSizeMsg{Width: 44, Height: 30})

	view := b.View()
	if !strings.Contains(view, longName) {
		t.Errorf("narrow view should show the full active column name %q even when the tab strip abbreviates it:\n%s", longName, view)
	}
}

func TestNarrow_TabTapSwitchesColumn(t *testing.T) {
	b := newMouseTestBoard()
	b.SetForceNarrow(true)
	b.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	_ = b.View() // populate layout.tabs

	var target *columnTarget
	for i := range b.layout.tabs {
		if b.layout.tabs[i].col != b.activeCol {
			target = &b.layout.tabs[i]
			break
		}
	}
	if target == nil {
		t.Fatalf("expected a non-active tab target, got tabs=%d activeCol=%d", len(b.layout.tabs), b.activeCol)
	}

	want := target.col
	b.Update(tea.MouseMsg{X: target.rect.x0, Y: target.rect.y0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if b.activeCol != want {
		t.Errorf("tapping the tab for column %d left activeCol=%d", want, b.activeCol)
	}
}

func TestNarrow_KeyboardTabSwitchesColumn(t *testing.T) {
	b := newMouseTestBoard()
	b.SetForceNarrow(true)
	b.Update(tea.WindowSizeMsg{Width: 60, Height: 30})

	start := b.activeCol
	b.Update(tea.KeyMsg{Type: tea.KeyTab})
	if b.activeCol == start {
		t.Fatalf("tab key did not advance column from %d", start)
	}
	b.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if b.activeCol != start {
		t.Errorf("shift+tab did not return to column %d, got %d", start, b.activeCol)
	}
}

// Scroll-fit must use the width the active column renders at (full width in
// narrow mode), not the wide-mode columnWidth().
func TestNarrow_ScrollUsesRenderWidth(t *testing.T) {
	b := newMouseTestBoard()
	// Fits one line at render width, wraps at the narrower columnWidth.
	const longTitle = "Medium-length card title here"
	b.columns[0].tasks = nil
	for i := 1; i <= 5; i++ {
		b.columns[0].tasks = append(b.columns[0].tasks, &task.Task{
			ID: 100 + i, Title: longTitle, Status: b.columns[0].status,
			Priority: "medium", Updated: mouseTestTime,
		})
	}
	b.activeCol, b.activeRow = 0, 0
	b.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	if !b.narrow() {
		t.Fatal("expected narrow mode for this scenario")
	}

	col := &b.columns[0]
	// The two widths must disagree on fit or the test can't catch anything.
	atRender := b.visibleCardsForColumn(col, b.width)
	atColWidth := b.visibleCardsForColumn(col, b.columnWidth())
	if atRender <= atColWidth {
		t.Fatalf("scenario not sensitive (render fit %d <= columnWidth fit %d); retune", atRender, atColWidth)
	}
	if atRender < len(col.tasks) {
		t.Fatalf("scenario needs all %d cards to fit at render width (fit %d); retune", len(col.tasks), atRender)
	}

	for range col.tasks {
		b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	if col.scrollOff != 0 {
		t.Errorf("column whose cards all fit at render width scrolled (scrollOff=%d)", col.scrollOff)
	}
}

// The compact ◂/▸ fallback bar must be tappable: left half = prev, right
// half = next.
func TestNarrow_CompactBarTapSwitchesColumn(t *testing.T) {
	b := newMouseTestBoard()
	b.SetForceNarrow(true)
	b.Update(tea.WindowSizeMsg{Width: 14, Height: 20}) // too narrow for tab chips
	_ = b.View()

	var next *columnTarget
	for i := range b.layout.tabs {
		if b.layout.tabs[i].col == b.activeCol+1 {
			next = &b.layout.tabs[i]
		}
	}
	if next == nil {
		t.Fatalf("compact bar produced no next-column tap target: %#v", b.layout.tabs)
	}
	want := next.col
	x := (next.rect.x0 + next.rect.x1) / 2
	b.Update(tea.MouseMsg{X: x, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if b.activeCol != want {
		t.Errorf("tapping compact-bar next zone left activeCol=%d, want %d", b.activeCol, want)
	}
}

// A card tap in narrow mode must select the card under it (guards the
// tab-bar row offset in captureNarrowBoardLayout).
func TestNarrow_CardTapSelects(t *testing.T) {
	b := newMouseTestBoard()
	b.SetForceNarrow(true)
	b.Update(tea.WindowSizeMsg{Width: 60, Height: 30})

	target := targetForTask(t, b, 1) // task #1 lives in the active (backlog) column
	clickTarget(b, target, tea.MouseButtonLeft)
	if got := b.selectedTask(); got == nil || got.ID != 1 {
		t.Errorf("card tap in narrow mode selected %#v, want task #1", got)
	}
}
