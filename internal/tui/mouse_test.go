package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var mouseTestTime = time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // fixed test clock

func newMouseTestBoard() *Board {
	cfg := config.NewDefault("Mouse Test")
	tasks := []*task.Task{
		{ID: 1, Title: "Task A", Status: "backlog", Priority: "high", Updated: mouseTestTime},
		{ID: 2, Title: "Task B with a longer title", Status: "backlog", Priority: "medium", Updated: mouseTestTime},
		{ID: 3, Title: "Task C", Status: "in-progress", Priority: "high", Updated: mouseTestTime},
		{ID: 4, Title: "Task D", Status: "done", Priority: "low", Updated: mouseTestTime},
	}
	b := &Board{
		cfg:          cfg,
		tasks:        tasks,
		columns:      columnsForTasks(cfg.BoardStatuses(), tasks),
		width:        120,
		height:       40,
		now:          func() time.Time { return mouseTestTime.Add(time.Hour) },
		mouseNow:     func() time.Time { return mouseTestTime },
		mouseEnabled: true,
		sortField:    "priority",
		sortReverse:  true,
	}
	_ = b.View()
	return b
}

func columnsForTasks(statuses []string, tasks []*task.Task) []column {
	columns := make([]column, len(statuses))
	for i, status := range statuses {
		columns[i].status = status
		for _, tk := range tasks {
			if tk.Status == status {
				columns[i].tasks = append(columns[i].tasks, tk)
			}
		}
	}
	return columns
}

func clickTarget(b *Board, target cardTarget, releaseButton tea.MouseButton) {
	x := target.rect.x0
	y := target.rect.y0
	_, _ = b.Update(tea.MouseMsg{
		X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: x, Y: y, Button: releaseButton, Action: tea.MouseActionRelease,
	})
}

func targetForTask(t *testing.T, b *Board, taskID int) cardTarget {
	t.Helper()
	_ = b.View()
	for _, target := range b.layout.cards {
		if target.taskID == taskID {
			return target
		}
	}
	t.Fatalf("task #%d has no visible mouse target: %#v", taskID, b.layout.cards)
	return cardTarget{}
}

func TestMouseClickSelectsAndDoubleClickOpensDetail(t *testing.T) {
	b := newMouseTestBoard()
	now := mouseTestTime
	b.SetMouseNow(func() time.Time { return now })
	target := targetForTask(t, b, 3)

	clickTarget(b, target, tea.MouseButtonLeft)
	if b.view != viewBoard {
		t.Fatalf("single click changed view to %v", b.view)
	}
	if got := b.selectedTask(); got == nil || got.ID != 3 {
		t.Fatalf("single click selected %#v, want task #3", got)
	}

	now = now.Add(doubleClickWindow)
	target = targetForTask(t, b, 3)
	clickTarget(b, target, tea.MouseButtonLeft)
	if b.view != viewDetail || b.detailTask == nil || b.detailTask.ID != 3 {
		t.Fatalf("double-click did not open task #3 detail: view=%v task=%#v", b.view, b.detailTask)
	}
}

func TestMouseDoubleClickExpiresAfter500Milliseconds(t *testing.T) {
	b := newMouseTestBoard()
	now := mouseTestTime
	b.SetMouseNow(func() time.Time { return now })
	target := targetForTask(t, b, 2)

	clickTarget(b, target, tea.MouseButtonLeft)
	now = now.Add(doubleClickWindow + time.Millisecond)
	target = targetForTask(t, b, 2)
	clickTarget(b, target, tea.MouseButtonLeft)

	if b.view != viewBoard {
		t.Fatalf("expired double-click opened view %v", b.view)
	}
	if b.pointer.lastClickID != 2 {
		t.Fatalf("expired click should start a new sequence, lastClickID=%d", b.pointer.lastClickID)
	}
}

func TestMouseOutsideClickBreaksDoubleClickSequence(t *testing.T) {
	b := newMouseTestBoard()
	target := targetForTask(t, b, 2)
	clickTarget(b, target, tea.MouseButtonLeft)

	_, _ = b.Update(tea.MouseMsg{
		X: 1, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: 1, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
	})

	target = targetForTask(t, b, 2)
	clickTarget(b, target, tea.MouseButtonLeft)
	if b.view != viewBoard {
		t.Fatalf("click after an outside click opened view %v", b.view)
	}
}

func TestMouseAcceptsSGRAndX10Releases(t *testing.T) {
	for _, tt := range []struct {
		name          string
		releaseButton tea.MouseButton
	}{
		{name: "SGR left-button release", releaseButton: tea.MouseButtonLeft},
		{name: "X10 buttonless release", releaseButton: tea.MouseButtonNone},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := newMouseTestBoard()
			clickTarget(b, targetForTask(t, b, 2), tt.releaseButton)
			if got := b.selectedTask(); got == nil || got.ID != 2 {
				t.Fatalf("selected task = %#v, want #2", got)
			}
		})
	}
}

func TestMouseBackIsSingleClick(t *testing.T) {
	b := newMouseTestBoard()
	b.selectTask(3)
	b.handleEnter()
	_ = b.View()
	if b.layout.back == nil {
		t.Fatal("mouse detail view has no Back target")
	}

	target := b.layout.back.rect
	_, _ = b.Update(tea.MouseMsg{
		X: target.x0, Y: target.y0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: target.x0, Y: target.y0, Button: tea.MouseButtonNone, Action: tea.MouseActionRelease,
	})

	if b.view != viewBoard || b.detailTask != nil {
		t.Fatalf("Back click left view=%v task=%#v", b.view, b.detailTask)
	}
	if got := b.selectedTask(); got == nil || got.ID != 3 {
		t.Fatalf("Back click lost keyboard selection: %#v", got)
	}
}

func TestMouseWheelRoutesByHoveredColumnAndClamps(t *testing.T) {
	b := newMouseTestBoard()
	backlog := b.layout.columns[0]

	_, _ = b.Update(tea.MouseMsg{
		X: backlog.rect.x0, Y: backlog.rect.y0,
		Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
	})
	if b.activeCol != 0 || b.activeRow != 1 {
		t.Fatalf("wheel down selected col=%d row=%d, want 0/1", b.activeCol, b.activeRow)
	}

	_, _ = b.Update(tea.MouseMsg{
		X: backlog.rect.x0, Y: backlog.rect.y0,
		Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
	})
	if b.activeRow != 1 {
		t.Fatalf("wheel moved beyond bottom boundary to row %d", b.activeRow)
	}

	_, _ = b.Update(tea.MouseMsg{
		X: backlog.rect.x0, Y: backlog.rect.y0,
		Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress,
	})
	if b.activeRow != 0 {
		t.Fatalf("wheel up selected row=%d, want 0", b.activeRow)
	}

	inProgress := b.layout.columns[2]
	_, _ = b.Update(tea.MouseMsg{
		X: inProgress.rect.x0, Y: inProgress.rect.y0,
		Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
	})
	if b.activeCol != 2 || b.activeRow != 0 {
		t.Fatalf("hovered wheel selected col=%d row=%d, want 2/0", b.activeCol, b.activeRow)
	}
}

func TestMouseWheelScrollsDetailThreeLinesAndClamps(t *testing.T) {
	b := newMouseTestBoard()
	b.tasks[0].Body = strings.Repeat("body line\n", 60)
	b.columns[0].tasks[0] = b.tasks[0]
	b.detailTask = b.tasks[0]
	b.view = viewDetail
	_ = b.View()

	_, _ = b.Update(tea.MouseMsg{
		X: 10, Y: 10, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
	})
	if b.detailScrollOff != detailWheelStep {
		t.Fatalf("detail offset=%d, want %d", b.detailScrollOff, detailWheelStep)
	}

	for range 100 {
		_, _ = b.Update(tea.MouseMsg{
			X: 10, Y: 10, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
		})
	}
	maxOff := len(detailLines(b.detailTask, b.width)) - (b.height - 2)
	if b.detailScrollOff != maxOff {
		t.Fatalf("detail bottom offset=%d, want %d", b.detailScrollOff, maxOff)
	}

	for range 100 {
		_, _ = b.Update(tea.MouseMsg{
			X: 10, Y: 10, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress,
		})
	}
	if b.detailScrollOff != 0 {
		t.Fatalf("detail top offset=%d, want 0", b.detailScrollOff)
	}
}

func TestMouseIgnoresButtonsModifiersAndInactiveViews(t *testing.T) {
	tests := []tea.MouseMsg{
		{X: 1, Y: 1, Button: tea.MouseButtonRight, Action: tea.MouseActionPress},
		{X: 1, Y: 1, Button: tea.MouseButtonMiddle, Action: tea.MouseActionPress},
		{X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, Shift: true},
		{X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, Alt: true},
		{X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, Ctrl: true},
		{X: 1, Y: 1, Button: tea.MouseButtonWheelLeft, Action: tea.MouseActionPress},
	}
	for _, msg := range tests {
		b := newMouseTestBoard()
		_, _ = b.Update(msg)
		if got := b.selectedTask(); got == nil || got.ID != 1 {
			t.Fatalf("message %s changed selection to %#v", msg.String(), got)
		}
		if b.pointer.pressed {
			t.Fatalf("message %s started a gesture", msg.String())
		}
	}

	for _, inactiveView := range []view{
		viewMove, viewConfirmDelete, viewHelp, viewCreate, viewDebug, viewSearch,
	} {
		b := newMouseTestBoard()
		b.view = inactiveView
		_ = b.View()
		_, _ = b.Update(tea.MouseMsg{
			X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
		})
		if b.pointer.pressed {
			t.Fatalf("view %v accepted mouse input", inactiveView)
		}
	}
}

func TestMousePendingClickInvalidatedByKeyboardResizeAndReload(t *testing.T) {
	tests := []struct {
		name       string
		invalidate tea.Msg
	}{
		{name: "keyboard", invalidate: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}},
		{name: "sort", invalidate: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}},
		{name: "search", invalidate: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}},
		{name: "view change", invalidate: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}},
		{name: "resize", invalidate: tea.WindowSizeMsg{Width: 100, Height: 30}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newMouseTestBoard()
			clickTarget(b, targetForTask(t, b, 1), tea.MouseButtonLeft)
			if b.pointer.lastClickID != 1 {
				t.Fatal("first click was not recorded")
			}

			_, _ = b.Update(tt.invalidate)
			if b.pointer.lastClickID != 0 || b.pointer.pressed {
				t.Fatalf("%s did not clear pointer state: %#v", tt.name, b.pointer)
			}
		})
	}
}

func TestMousePendingClickInvalidatedByWheelAndReload(t *testing.T) {
	b := newMouseFilesystemBoard(t)
	clickTarget(b, targetForTask(t, b, 1), tea.MouseButtonLeft)
	col := b.layout.columns[0]
	_, _ = b.Update(tea.MouseMsg{
		X: col.rect.x0, Y: col.rect.y0,
		Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
	})
	if b.pointer.lastClickID != 0 {
		t.Fatalf("wheel did not clear pending click: %#v", b.pointer)
	}

	clickTarget(b, targetForTask(t, b, 1), tea.MouseButtonLeft)
	_, _ = b.Update(ReloadMsg{})
	if b.pointer.lastClickID != 0 || b.pointer.pressed {
		t.Fatalf("reload did not clear pointer state: %#v", b.pointer)
	}
}

func newMouseFilesystemBoard(t *testing.T) *Board {
	t.Helper()
	kanbanDir := filepath.Join(t.TempDir(), "kanban")
	if err := os.MkdirAll(filepath.Join(kanbanDir, "tasks"), 0o750); err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefault("Mouse Filesystem Test")
	cfg.SetDir(kanbanDir)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	tk := &task.Task{
		ID: 1, Title: "Task A", Status: "backlog", Priority: "high", Updated: mouseTestTime,
	}
	path := filepath.Join(cfg.TasksPath(), task.GenerateFilename(tk.ID, tk.Title))
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}
	b := NewBoard(cfg)
	b.SetMouseEnabled(true)
	b.SetMouseNow(func() time.Time { return mouseTestTime })
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	_ = b.View()
	return b
}

func TestMouseReleaseAfterMotionOutsideCardCancels(t *testing.T) {
	b := newMouseTestBoard()
	target := targetForTask(t, b, 2)
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0, Y: target.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x1 + 1, Y: target.rect.y1 + 1,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0, Y: target.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
	})
	if got := b.selectedTask(); got == nil || got.ID != 1 {
		t.Fatalf("canceled gesture selected %#v, want original #1", got)
	}
}

func TestMouseEdgeBranches(t *testing.T) {
	b := newMouseTestBoard()
	target := targetForTask(t, b, 1)

	// Presses outside cards and releases without a matching press are ignored.
	_, _ = b.Update(tea.MouseMsg{
		X: 1, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0, Y: target.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
	})
	if b.pointer.pressed {
		t.Fatal("outside press started a gesture")
	}

	// Motion within the card keeps the gesture; an unrelated release cancels it.
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0, Y: target.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0 + 1, Y: target.rect.y0 + 1,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
	})
	if !b.pointer.pressed {
		t.Fatal("motion within target canceled gesture")
	}
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0 + 1, Y: target.rect.y0 + 1,
		Button: tea.MouseButtonRight, Action: tea.MouseActionRelease,
	})
	if b.pointer.pressed {
		t.Fatal("different-button release did not cancel gesture")
	}

	// A stale layout generation cannot complete a click.
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0, Y: target.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	b.layoutGeneration++
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0, Y: target.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
	})
	if b.pointer.lastClickID != 0 {
		t.Fatal("stale layout completed a click")
	}

	if b.cardAt(-1, -1) != nil || b.columnAt(-1, -1) != nil {
		t.Fatal("off-screen coordinate returned a target")
	}
	if b.selectTask(999) {
		t.Fatal("missing task was selected")
	}

	// Disabled boards ignore mouse messages entirely.
	b.SetMouseEnabled(false)
	_, _ = b.Update(tea.MouseMsg{
		X: target.rect.x0, Y: target.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	if b.pointer.pressed {
		t.Fatal("disabled mouse mode accepted a press")
	}
}

func TestMouseDetailEdgeBranches(t *testing.T) {
	b := newMouseTestBoard()
	b.detailTask = b.tasks[0]
	b.view = viewDetail
	_ = b.View()

	// Non-left presses, outside presses, outside motion, and unrelated releases
	// must not activate Back.
	_, _ = b.Update(tea.MouseMsg{
		X: 0, Y: b.height - 1, Button: tea.MouseButtonRight, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: 10, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	if b.pointer.pressed {
		t.Fatal("detail content started a Back gesture")
	}
	back := b.layout.back.rect
	_, _ = b.Update(tea.MouseMsg{
		X: back.x0, Y: back.y0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: back.x1 + 1, Y: back.y0, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: back.x0, Y: back.y0, Button: tea.MouseButtonRight, Action: tea.MouseActionRelease,
	})
	if b.view != viewDetail {
		t.Fatal("canceled Back gesture changed the view")
	}

	b.detailTask = nil
	b.detailScrollOff = 99
	b.clampDetailScroll()
	if b.detailScrollOff != 0 {
		t.Fatalf("nil detail task offset=%d, want 0", b.detailScrollOff)
	}

	b.detailTask = b.tasks[0]
	b.height = 1
	b.detailScrollOff = 99
	b.clampDetailScroll()
	if b.detailScrollOff != 0 {
		t.Fatalf("short detail viewport offset=%d, want 0", b.detailScrollOff)
	}
}

func TestMouseLayoutUnavailableUntilRendered(t *testing.T) {
	b := newMouseTestBoard()
	b.invalidatePointerState()
	if b.cardAt(1, 1) != nil || b.columnAt(1, 1) != nil {
		t.Fatal("invalidated layout returned targets")
	}

	b.view = viewHelp
	_ = b.View()
	if b.cardAt(1, 1) != nil || b.columnAt(1, 1) != nil {
		t.Fatal("non-board view returned board targets")
	}

	b.view = viewBoard
	b.width = 0
	b.layout = layoutSnapshot{}
	b.captureBoardLayout(10, 10, nil)
	if len(b.layout.columns) != 0 {
		t.Fatal("zero-width board captured columns")
	}
}

func TestSetHideEmptyColumnsInvalidatesMouseState(t *testing.T) {
	b := newMouseFilesystemBoard(t)
	clickTarget(b, targetForTask(t, b, 1), tea.MouseButtonLeft)
	b.SetHideEmptyColumns(true)
	if b.pointer.lastClickID != 0 || b.pointer.pressed {
		t.Fatalf("hide-empty change did not invalidate pointer state: %#v", b.pointer)
	}
}

func TestMouseLayoutProperties(t *testing.T) {
	for _, titleLines := range []int{1, 2, 3} {
		for _, width := range []int{1, 4, 8, 9, 39, 40, 79, 80, 121, 300} {
			for _, height := range []int{1, 2, 3, 4, 8, 19, 40} {
				t.Run(fmt.Sprintf("title_%d/%dx%d", titleLines, width, height), func(t *testing.T) {
					b := newMouseTestBoard()
					b.cfg.TUI.TitleLines = titleLines
					b.width = width
					b.height = height
					if height%2 == 0 {
						b.err = errors.New("visible error chrome")
					}
					b.columns[0].tasks[1].Title = "日本語の長いタイトル with mixed widths"
					_ = b.View()
					assertValidMouseLayout(t, b)
				})
			}
		}
	}
}

func TestMouseLayoutAccountsForScrollIndicatorsAndVisibleEmptyColumns(t *testing.T) {
	b := newMouseTestBoard()
	b.columns[0].scrollOff = 1
	b.columns = b.columns[:3] // hidden and archived columns are not rendered targets
	_ = b.View()

	target := targetForTask(t, b, 2)
	if target.rect.y0 != 2 {
		t.Fatalf("scrolled card starts at y=%d, want 2 after header and up indicator", target.rect.y0)
	}
	if len(b.layout.columns) != 3 {
		t.Fatalf("rendered column targets=%d, want 3", len(b.layout.columns))
	}
	if b.layout.columns[1].status != "todo" {
		t.Fatalf("empty rendered column status=%q, want todo", b.layout.columns[1].status)
	}
	for _, card := range b.layout.cards {
		if card.col == 1 {
			t.Fatalf("empty column unexpectedly has card target %#v", card)
		}
	}
}

func TestMouseLayoutClipsUnusedRightSideAndBottomChrome(t *testing.T) {
	b := newMouseTestBoard()
	b.width = 300 // columns cap at 50, leaving unused right-side space
	b.height = 12
	b.err = errors.New("error")
	_ = b.View()
	assertValidMouseLayout(t, b)

	last := b.layout.columns[len(b.layout.columns)-1]
	if last.rect.x1 != len(b.columns)*50 {
		t.Fatalf("last column ends at x=%d, want %d", last.rect.x1, len(b.columns)*50)
	}
	if b.columnAt(299, 1) != nil {
		t.Fatal("unused right-side space was treated as a column")
	}
	if b.columnAt(1, b.height-b.chromeHeight()) != nil {
		t.Fatal("bottom chrome was treated as a column")
	}
}

func TestMouseLayoutUnicodeCardHeightMatchesTarget(t *testing.T) {
	b := newMouseTestBoard()
	b.cfg.TUI.TitleLines = 3
	b.columns[0].tasks[0].Title = "日本語のタイトル 日本語のタイトル"
	b.width = 40
	_ = b.View()

	target := targetForTask(t, b, 1)
	want := b.cardHeight(b.columns[0].tasks[0], b.columnWidth())
	if got := target.rect.y1 - target.rect.y0; got != want {
		t.Fatalf("Unicode card target height=%d, want rendered height %d", got, want)
	}
}

func assertValidMouseLayout(t testing.TB, b *Board) {
	t.Helper()
	screen := rect{x0: 0, y0: 0, x1: max(0, b.width), y1: max(0, b.height-b.chromeHeight())}

	for i, col := range b.layout.columns {
		if col.rect.empty() {
			t.Fatalf("column %d has empty rect %#v", i, col.rect)
		}
		if col.rect != col.rect.intersect(screen) {
			t.Fatalf("column %d escapes screen: rect=%#v screen=%#v", i, col.rect, screen)
		}
		for j := i + 1; j < len(b.layout.columns); j++ {
			if rectsOverlap(col.rect, b.layout.columns[j].rect) {
				t.Fatalf("columns %d and %d overlap: %#v %#v", i, j, col.rect, b.layout.columns[j].rect)
			}
		}
	}

	for i, card := range b.layout.cards {
		if card.rect.empty() {
			t.Fatalf("card %d has empty rect %#v", card.taskID, card.rect)
		}
		if card.rect != card.rect.intersect(screen) {
			t.Fatalf("card %d escapes screen: rect=%#v screen=%#v", card.taskID, card.rect, screen)
		}
		var containing int
		for _, col := range b.layout.columns {
			if card.rect == card.rect.intersect(col.rect) {
				containing++
			}
		}
		if containing != 1 {
			t.Fatalf("card %d belongs to %d rendered columns", card.taskID, containing)
		}
		for j := i + 1; j < len(b.layout.cards); j++ {
			if card.col == b.layout.cards[j].col && rectsOverlap(card.rect, b.layout.cards[j].rect) {
				t.Fatalf("cards %d and %d overlap: %#v %#v",
					card.taskID, b.layout.cards[j].taskID, card.rect, b.layout.cards[j].rect)
			}
		}
	}
}

func rectsOverlap(a, b rect) bool {
	return a.x0 < b.x1 && b.x0 < a.x1 && a.y0 < b.y1 && b.y0 < a.y1
}

func FuzzMouseLayout(f *testing.F) {
	f.Add(120, 40, 2, 6, 0, false)
	f.Add(1, 1, 1, 1, 0, true)
	f.Add(300, 8, 3, 24, 10, false)
	f.Fuzz(func(t *testing.T, width, height, titleLines, taskCount, scrollOff int, withError bool) {
		width = fuzzRange(width, 1, 400)
		height = fuzzRange(height, 1, 100)
		titleLines = fuzzRange(titleLines, 1, 3)
		taskCount = fuzzRange(taskCount, 0, 50)

		cfg := config.NewDefault("Fuzz")
		cfg.TUI.TitleLines = titleLines
		tasks := make([]*task.Task, taskCount)
		for i := range taskCount {
			tasks[i] = &task.Task{
				ID:       i + 1,
				Title:    strings.Repeat("日本語 mixed title ", i%7+1),
				Status:   cfg.BoardStatuses()[i%len(cfg.BoardStatuses())],
				Priority: cfg.Priorities[i%len(cfg.Priorities)],
				Updated:  mouseTestTime,
			}
		}
		b := &Board{
			cfg:          cfg,
			tasks:        tasks,
			columns:      columnsForTasks(cfg.BoardStatuses(), tasks),
			width:        width,
			height:       height,
			now:          func() time.Time { return mouseTestTime },
			mouseNow:     func() time.Time { return mouseTestTime },
			mouseEnabled: true,
			sortField:    "priority",
		}
		if len(b.columns) > 0 && len(b.columns[0].tasks) > 0 {
			b.columns[0].scrollOff = fuzzRange(scrollOff, 0, len(b.columns[0].tasks)-1)
		}
		if withError {
			b.err = errors.New("fuzz error chrome")
		}
		_ = b.View()
		assertValidMouseLayout(t, b)
	})
}

func fuzzRange(v, low, high int) int {
	if high <= low {
		return low
	}
	if v < 0 {
		v = -v
	}
	return low + v%(high-low+1)
}
