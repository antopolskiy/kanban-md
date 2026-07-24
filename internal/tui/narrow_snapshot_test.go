package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSnapshot_NarrowBoard(t *testing.T) {
	b, _ := setupTestBoard(t)
	b.Update(tea.WindowSizeMsg{Width: 48, Height: 24})
	assertGolden(t, "narrow_board", trimSnapshotLineEnds(b.View()))
}
