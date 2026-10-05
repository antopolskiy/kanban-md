package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWorkflowRelationOpensChild(t *testing.T) {
	b := setupParentChildrenBoard(t)
	b = sendKey(b, "enter")
	b = sendSpecialKey(b, tea.KeyTab)
	b = sendSpecialKey(b, tea.KeyEnter)
	if got := b.View(); !containsStr(got, "Task #2: Backlog child") {
		t.Fatalf("Tab, Enter should open direct child #2:\n%s", got)
	}
}

func TestWorkflowDepthFiltersExactLevel(t *testing.T) {
	b := setupParentChildrenBoard(t)
	b = sendKey(b, "v")
	if got := b.View(); !containsStr(got, "level[0]") || !containsStr(got, "1 card") {
		t.Fatalf("v should show the only active level-zero task:\n%s", got)
	}
}
