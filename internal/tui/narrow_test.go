package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

func TestNarrow_ActiveHeaderPrioritizesNameOverMetadata(t *testing.T) {
	b := newMouseTestBoard()
	const statusName = "abcdefghijklmnopq" // 17 cells; fits in width 20 after header padding
	b.columns[0].status = statusName
	b.activeCol = 0
	b.SetForceNarrow(true)
	b.Update(tea.WindowSizeMsg{Width: 20, Height: 30})

	rendered, _ := b.renderColumn(0, b.columns[0], b.width)
	header := strings.SplitN(rendered, "\n", 2)[0]
	if !strings.Contains(header, statusName) {
		t.Fatalf("active header should preserve fitting status name %q: %q", statusName, header)
	}
	if strings.Contains(header, "(2)") {
		t.Fatalf("active header should omit metadata that would truncate the status name: %q", header)
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

func TestNarrow_CompactBarShowsNavigationAndPosition(t *testing.T) {
	tests := []struct {
		name     string
		active   int
		wantPrev bool
		wantNext bool
	}{
		{name: "first", active: 0, wantNext: true},
		{name: "middle", active: 2, wantPrev: true, wantNext: true},
		{name: "last", active: 4, wantPrev: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newMouseTestBoard()
			b.activeCol = tt.active
			b.SetForceNarrow(true)
			b.Update(tea.WindowSizeMsg{Width: 14, Height: 20})

			rendered, _ := b.renderNarrowCompactBar()
			if got := strings.Contains(rendered, "◂"); got != tt.wantPrev {
				t.Errorf("previous cue visible=%v, want %v: %q", got, tt.wantPrev, rendered)
			}
			if got := strings.Contains(rendered, "▸"); got != tt.wantNext {
				t.Errorf("next cue visible=%v, want %v: %q", got, tt.wantNext, rendered)
			}
			position := fmt.Sprintf("%d/%d", tt.active+1, len(b.columns))
			if !strings.Contains(rendered, position) {
				t.Errorf("compact bar should preserve position %q: %q", position, rendered)
			}
		})
	}
}

func TestCompactNarrowHeader_TinyWidthsDoNotOverflow(t *testing.T) {
	for width := 1; width <= 8; width++ {
		got := compactNarrowHeader("backlog (2)", true, true, 3, 5, width)
		if gotWidth := lipgloss.Width(got); gotWidth != width {
			t.Errorf("width %d rendered %d cells: %q", width, gotWidth, got)
		}
	}

	if got := compactNarrowHeader("backlog (2)", true, true, 3, 5, 2); got != "◂▸" {
		t.Errorf("two-cell header should prioritize both navigation cues, got %q", got)
	}
	if got := compactNarrowHeader("backlog (2)", true, true, 3, 5, 6); got != "◂3/5 ▸" {
		t.Errorf("six-cell header should show navigation and position, got %q", got)
	}
}

func TestNarrow_CompactBarTinyWidthsAlignCuesAndTargets(t *testing.T) {
	for width := 1; width <= 10; width++ {
		for _, active := range []int{0, 2, 4} {
			name := fmt.Sprintf("width_%d/column_%d", width, active)
			t.Run(name, func(t *testing.T) {
				b := newMouseTestBoard()
				b.activeCol = active
				b.SetForceNarrow(true)
				b.Update(tea.WindowSizeMsg{Width: width, Height: 20})

				rendered, _ := b.renderNarrowCompactBar()
				if got := lipgloss.Width(rendered); got > width {
					t.Fatalf("compact bar rendered %d cells in terminal width %d: %q", got, width, rendered)
				}

				visibleCues := 0
				for _, cue := range []struct {
					rune rune
					col  int
				}{
					{rune: '◂', col: active - 1},
					{rune: '▸', col: active + 1},
				} {
					x, visible := displayCellForRune(rendered, cue.rune)
					if !visible {
						continue
					}
					visibleCues++

					b.activeCol = active
					_ = b.View()
					var target *columnTarget
					for i := range b.layout.tabs {
						if b.layout.tabs[i].col == cue.col {
							target = &b.layout.tabs[i]
							break
						}
					}
					if target == nil {
						t.Fatalf("visible cue %q has no target: %q %#v", cue.rune, rendered, b.layout.tabs)
					}
					if !target.rect.contains(x, 0) {
						t.Errorf("visible cue %q at cell %d is outside target %#v: %q", cue.rune, x, target.rect, rendered)
					}

					b.Update(tea.MouseMsg{
						X: x, Y: 0,
						Button: tea.MouseButtonLeft,
						Action: tea.MouseActionPress,
					})
					if b.activeCol != cue.col {
						t.Errorf("clicking visible cue %q selected column %d, want %d", cue.rune, b.activeCol, cue.col)
					}
				}
				if visibleCues == 0 && len(b.columns) > 1 {
					t.Errorf("compact bar exposes no navigation cue: %q", rendered)
				}
			})
		}
	}
}

func displayCellForRune(s string, want rune) (int, bool) {
	index := strings.IndexRune(s, want)
	if index < 0 {
		return 0, false
	}
	return lipgloss.Width(s[:index]), true
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
